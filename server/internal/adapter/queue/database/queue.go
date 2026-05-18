package database

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/kilip/opus/server/ent"
	"github.com/kilip/opus/server/ent/job"
	"github.com/kilip/opus/server/internal/shared/logger"
	"github.com/kilip/opus/server/internal/shared/queue"
)

const pollInterval = 500 * time.Millisecond

// DatabaseQueue implements queue.Queue backed by Ent.
type DatabaseQueue struct {
	db           *ent.Client
	handlers     map[string]queue.Handler
	handlersMu   sync.RWMutex
	concurrency  int
	logger       logger.Logger
	started      atomic.Bool
	wg           sync.WaitGroup
	cancelLoop   context.CancelFunc
	RetryBackoff time.Duration
}

// NewDatabaseQueue returns a ready-to-use DatabaseQueue.
func NewDatabaseQueue(db *ent.Client, concurrency int, log logger.Logger) *DatabaseQueue {
	return &DatabaseQueue{
		db:           db,
		handlers:     make(map[string]queue.Handler),
		concurrency:  concurrency,
		logger:       log.With(logger.String("component", "database_queue")),
		RetryBackoff: 30 * time.Second,
	}
}

func (q *DatabaseQueue) Enqueue(ctx context.Context, jobType string, payload []byte, opts ...queue.JobOption) (string, error) {
	j := &queue.Job{
		ID:         uuid.NewString(),
		Type:       jobType,
		Payload:    payload,
		Queue:      "default",
		Priority:   0,
		MaxRetries: 3,
		ProcessAt:  time.Now(),
		CreatedAt:  time.Now(),
		Status:     queue.JobStatusPending,
	}
	for _, opt := range opts {
		opt(j)
	}

	_, err := q.db.Job.Create().
		SetID(j.ID).
		SetType(j.Type).
		SetQueue(j.Queue).
		SetPayload(j.Payload).
		SetStatus(string(j.Status)).
		SetPriority(j.Priority).
		SetMaxRetries(j.MaxRetries).
		SetProcessAt(j.ProcessAt).
		SetCreatedAt(j.CreatedAt).
		Save(ctx)

	if err != nil {
		return "", fmt.Errorf("database_queue: %w", err)
	}

	return j.ID, nil
}

func (q *DatabaseQueue) RegisterHandler(jobType string, handler queue.Handler) {
	if q.started.Load() {
		panic("queue: RegisterHandler called after Start()")
	}
	q.handlersMu.Lock()
	defer q.handlersMu.Unlock()
	q.handlers[jobType] = handler
}

func (q *DatabaseQueue) Start(ctx context.Context) error {
	if !q.started.CompareAndSwap(false, true) {
		return nil
	}

	loopCtx, cancel := context.WithCancel(context.Background())
	q.cancelLoop = cancel
	q.wg.Add(1)

	go func() {
		defer q.wg.Done()
		sem := make(chan struct{}, q.concurrency)
		ticker := time.NewTicker(pollInterval)
		defer ticker.Stop()

		for {
			select {
			case <-loopCtx.Done():
				return
			case <-ticker.C:
			pollLoop:
				for {
					select {
					case sem <- struct{}{}:
						j, err := q.claimNext(loopCtx)
						if err != nil {
							if !ent.IsNotFound(err) {
								q.logger.ErrorCtx(loopCtx, "queue poll error", err)
							}
							<-sem
							break pollLoop
						}

						if j == nil {
							<-sem
							break pollLoop
						}

						q.wg.Add(1)
						go func(job *queue.Job) {
							defer q.wg.Done()
							defer func() { <-sem }()
							q.processJob(loopCtx, job)
						}(j)
					default:
						break pollLoop
					}
				}
			}
		}
	}()

	return nil
}

func (q *DatabaseQueue) claimNext(ctx context.Context) (*queue.Job, error) {
	tx, err := q.db.Tx(ctx)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	record, err := tx.Job.Query().
		Where(
			job.StatusEQ(string(queue.JobStatusPending)),
			job.ProcessAtLTE(now),
		).
		Order(ent.Desc(job.FieldPriority), ent.Asc(job.FieldProcessAt)).
		First(ctx)

	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}

	_, err = tx.Job.UpdateOne(record).
		SetStatus(string(queue.JobStatusRunning)).
		SetUpdatedAt(time.Now()).
		Save(ctx)

	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &queue.Job{
		ID:         record.ID,
		Type:       record.Type,
		Queue:      record.Queue,
		Payload:    record.Payload,
		Status:     queue.JobStatus(record.Status),
		Priority:   record.Priority,
		MaxRetries: record.MaxRetries,
		RetryCount: record.RetryCount,
		ProcessAt:  record.ProcessAt,
		CreatedAt:  record.CreatedAt,
	}, nil
}

func (q *DatabaseQueue) processJob(ctx context.Context, j *queue.Job) {
	q.handlersMu.RLock()
	handler, ok := q.handlers[j.Type]
	q.handlersMu.RUnlock()

	var err error
	if !ok {
		err = fmt.Errorf("no handler for job type: %s", j.Type)
	} else {
		err = handler(ctx, j)
	}

	if err != nil {
		j.RetryCount++
		if j.RetryCount >= j.MaxRetries {
			j.Status = queue.JobStatusDead
		} else {
			j.Status = queue.JobStatusFailed
			delaySecs := j.RetryCount * j.RetryCount
			j.ProcessAt = time.Now().Add(time.Duration(delaySecs) * q.RetryBackoff)
		}
	} else {
		j.Status = queue.JobStatusCompleted
	}

	statusStr := string(j.Status)
	if j.Status == queue.JobStatusFailed {
		statusStr = string(queue.JobStatusPending)
	}

	_, execErr := q.db.Job.UpdateOneID(j.ID).
		SetStatus(statusStr).
		SetRetryCount(j.RetryCount).
		SetProcessAt(j.ProcessAt).
		SetUpdatedAt(time.Now()).
		Save(context.Background())

	if execErr != nil {
		q.logger.ErrorCtx(context.Background(), "failed to update job status in database", execErr)
	}
}

func (q *DatabaseQueue) Shutdown(ctx context.Context) error {
	if q.cancelLoop != nil {
		q.cancelLoop()
	}

	c := make(chan struct{})
	go func() {
		defer close(c)
		q.wg.Wait()
	}()

	select {
	case <-c:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (q *DatabaseQueue) Inspect(ctx context.Context, jobID string) (*queue.Job, error) {
	record, err := q.db.Job.Query().Where(job.IDEQ(jobID)).Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("database_queue: job not found")
		}
		return nil, fmt.Errorf("database_queue: %w", err)
	}
	return &queue.Job{
		ID:         record.ID,
		Type:       record.Type,
		Queue:      record.Queue,
		Payload:    record.Payload,
		Status:     queue.JobStatus(record.Status),
		Priority:   record.Priority,
		MaxRetries: record.MaxRetries,
		RetryCount: record.RetryCount,
		ProcessAt:  record.ProcessAt,
		CreatedAt:  record.CreatedAt,
	}, nil
}
