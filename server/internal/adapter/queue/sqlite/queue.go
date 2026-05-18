// Package sqlite provides a SQLite-backed Queue implementation.
package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/kilip/opus/server/internal/shared/logger"
	"github.com/kilip/opus/server/internal/shared/queue"
)

const pollInterval = 500 * time.Millisecond

// SQLiteQueue implements queue.Queue using a shared *sql.DB.
type SQLiteQueue struct {
	db          *sql.DB
	concurrency int
	logger      logger.Logger
	handlers    map[string]queue.Handler
	handlersMu  sync.RWMutex
	started     atomic.Bool
	wg          sync.WaitGroup
	cancelLoop  context.CancelFunc
}

// NewSQLiteQueue constructs a SQLiteQueue using the provided shared database connection.
func NewSQLiteQueue(db *sql.DB, concurrency int, log logger.Logger) (*SQLiteQueue, error) {
	if err := migrate(db); err != nil {
		return nil, fmt.Errorf("sqlite.NewSQLiteQueue: %w", err)
	}

	return &SQLiteQueue{
		db:          db,
		concurrency: concurrency,
		logger:      log.With(logger.String("component", "sqlite_queue")),
		handlers:    make(map[string]queue.Handler),
	}, nil
}

func migrate(db *sql.DB) error {
	q1 := `CREATE TABLE IF NOT EXISTS opus_jobs (
		id          TEXT PRIMARY KEY,
		type        TEXT NOT NULL,
		queue       TEXT NOT NULL DEFAULT 'default',
		payload     BLOB NOT NULL,
		status      TEXT NOT NULL DEFAULT 'pending',
		priority    INTEGER NOT NULL DEFAULT 0,
		max_retries INTEGER NOT NULL DEFAULT 3,
		retry_count INTEGER NOT NULL DEFAULT 0,
		process_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);`
	if _, err := db.Exec(q1); err != nil {
		return err
	}

	q2 := `CREATE INDEX IF NOT EXISTS idx_opus_jobs_poll
		ON opus_jobs (queue, status, priority DESC, process_at ASC);`
	if _, err := db.Exec(q2); err != nil {
		return err
	}
	return nil
}

// Enqueue inserts a new job into the database.
func (q *SQLiteQueue) Enqueue(ctx context.Context, jobType string, payload []byte, opts ...queue.JobOption) (string, error) {
	job := &queue.Job{
		ID:         uuid.NewString(),
		Type:       jobType,
		Queue:      "default",
		Payload:    payload,
		Status:     queue.JobStatusPending,
		Priority:   0,
		MaxRetries: 3,
		RetryCount: 0,
		ProcessAt:  time.Now(),
		CreatedAt:  time.Now(),
	}

	for _, opt := range opts {
		opt(job)
	}

	conn, err := q.db.Conn(ctx)
	if err != nil {
		return "", fmt.Errorf("sqlite.SQLiteQueue.Enqueue: %w", err)
	}
	defer func() { _ = conn.Close() }()

	_, _ = conn.ExecContext(ctx, "PRAGMA busy_timeout = 5000;")

	query := `INSERT INTO opus_jobs (id, type, queue, payload, status, priority, max_retries, retry_count, process_at, created_at, updated_at) 
	          VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err = conn.ExecContext(ctx, query, job.ID, job.Type, job.Queue, job.Payload, string(job.Status), job.Priority, job.MaxRetries, job.RetryCount, job.ProcessAt, job.CreatedAt, time.Now())
	if err != nil {
		return "", fmt.Errorf("sqlite.SQLiteQueue.Enqueue: %w", err)
	}

	return job.ID, nil
}

// Inspect retrieves a job by ID.
func (q *SQLiteQueue) Inspect(ctx context.Context, jobID string) (*queue.Job, error) {
	query := `SELECT id, type, queue, payload, status, priority, max_retries, retry_count, process_at, created_at FROM opus_jobs WHERE id = ?`
	row := q.db.QueryRowContext(ctx, query, jobID)

	var j queue.Job
	var statusStr string
	err := row.Scan(&j.ID, &j.Type, &j.Queue, &j.Payload, &statusStr, &j.Priority, &j.MaxRetries, &j.RetryCount, &j.ProcessAt, &j.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("sqlite.SQLiteQueue.Inspect: job not found")
		}
		return nil, fmt.Errorf("sqlite.SQLiteQueue.Inspect: %w", err)
	}
	j.Status = queue.JobStatus(statusStr)
	return &j, nil
}

// RegisterHandler registers a handler for a job type. Panics if called after Start().
func (q *SQLiteQueue) RegisterHandler(jobType string, handler queue.Handler) {
	if q.started.Load() {
		panic("queue: RegisterHandler called after Start()")
	}
	q.handlersMu.Lock()
	defer q.handlersMu.Unlock()
	q.handlers[jobType] = handler
}

// Start begins polling for jobs.
func (q *SQLiteQueue) Start(ctx context.Context) error {
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
						job, err := q.claimNext(loopCtx)
						if err != nil {
							if err != sql.ErrNoRows {
								q.logger.ErrorCtx(loopCtx, "queue poll error", err)
							}
							<-sem
							break pollLoop
						}

						if job == nil {
							<-sem
							break pollLoop
						}

						q.wg.Add(1)
						go func(j *queue.Job) {
							defer q.wg.Done()
							defer func() { <-sem }()
							q.processJob(loopCtx, j)
						}(job)
					default:
						// Concurrency limit reached, stop polling until next tick
						break pollLoop
					}
				}
			}
		}
	}()

	return nil
}

func (q *SQLiteQueue) claimNext(ctx context.Context) (*queue.Job, error) {
	conn, err := q.db.Conn(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = conn.Close() }()

	_, _ = conn.ExecContext(ctx, "PRAGMA busy_timeout = 5000;")
	_, err = conn.ExecContext(ctx, "BEGIN IMMEDIATE")
	if err != nil {
		return nil, err
	}

	// We use process_at <= ? with time.Now() to handle Go-to-SQLite timezone conversions correctly.
	query := `SELECT id, type, queue, payload, status, priority, max_retries, retry_count, process_at, created_at 
	          FROM opus_jobs 
	          WHERE status = 'pending' AND process_at <= ? 
	          ORDER BY priority DESC, process_at ASC LIMIT 1`

	row := conn.QueryRowContext(ctx, query, time.Now())
	var j queue.Job
	var statusStr string
	err = row.Scan(&j.ID, &j.Type, &j.Queue, &j.Payload, &statusStr, &j.Priority, &j.MaxRetries, &j.RetryCount, &j.ProcessAt, &j.CreatedAt)
	if err != nil {
		_, _ = conn.ExecContext(ctx, "ROLLBACK")
		return nil, err
	}
	j.Status = queue.JobStatus(statusStr)

	_, err = conn.ExecContext(ctx, "UPDATE opus_jobs SET status = 'running', updated_at = CURRENT_TIMESTAMP WHERE id = ?", j.ID)
	if err != nil {
		_, _ = conn.ExecContext(ctx, "ROLLBACK")
		return nil, err
	}

	_, err = conn.ExecContext(ctx, "COMMIT")
	if err != nil {
		return nil, err
	}

	return &j, nil
}

func (q *SQLiteQueue) processJob(ctx context.Context, job *queue.Job) {
	q.handlersMu.RLock()
	handler, ok := q.handlers[job.Type]
	q.handlersMu.RUnlock()

	var err error
	if !ok {
		err = fmt.Errorf("no handler for job type: %s", job.Type)
	} else {
		err = handler(ctx, job)
	}

	if err != nil {
		job.RetryCount++
		if job.RetryCount >= job.MaxRetries {
			job.Status = queue.JobStatusDead
		} else {
			job.Status = queue.JobStatusFailed
			delaySecs := job.RetryCount * job.RetryCount * 30
			job.ProcessAt = time.Now().Add(time.Duration(delaySecs) * time.Second)
		}
	} else {
		job.Status = queue.JobStatusCompleted
	}

	statusStr := string(job.Status)
	if job.Status == queue.JobStatusFailed {
		statusStr = string(queue.JobStatusPending)
	}

	query := `UPDATE opus_jobs SET status = ?, retry_count = ?, process_at = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	conn, connErr := q.db.Conn(context.Background())
	if connErr != nil {
		q.logger.ErrorCtx(context.Background(), "failed to get connection for job status update", connErr)
		return
	}
	defer func() { _ = conn.Close() }()

	_, _ = conn.ExecContext(context.Background(), "PRAGMA busy_timeout = 5000;")
	if _, execErr := conn.ExecContext(context.Background(), query, statusStr, job.RetryCount, job.ProcessAt, job.ID); execErr != nil {
		q.logger.ErrorCtx(context.Background(), "failed to update job status in database", execErr)
	}
}

// Shutdown gracefully stops polling and waits for active jobs.
func (q *SQLiteQueue) Shutdown(ctx context.Context) error {
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
