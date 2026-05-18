package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/kilip/opus/server/internal/shared/logger"
	"github.com/kilip/opus/server/internal/shared/queue"
)

// RedisQueue implements queue.Queue backed by Asynq + Redis.
type RedisQueue struct {
	client   *asynq.Client
	server   *asynq.Server
	mux      *asynq.ServeMux
	opt      asynq.RedisConnOpt
	handlers map[string]queue.Handler
	logger   logger.Logger
	started  bool
}

// NewRedisQueue creates an Asynq client and server connected to Redis at the given URL.
func NewRedisQueue(redisURL string, concurrency int, log logger.Logger) (*RedisQueue, error) {
	opt, err := asynq.ParseRedisURI(redisURL)
	if err != nil {
		return nil, fmt.Errorf("redis_queue: %w", err)
	}

	client := asynq.NewClient(opt)
	server := asynq.NewServer(opt, asynq.Config{
		Concurrency: concurrency,
		Queues: map[string]int{
			"agent":   10, // agent tasks: highest weight
			"default": 5,
			"email":   3,
		},
	})

	return &RedisQueue{
		client:   client,
		server:   server,
		mux:      asynq.NewServeMux(),
		opt:      opt,
		handlers: make(map[string]queue.Handler),
		logger:   log.With(logger.String("component", "redis_queue")),
	}, nil
}

func (q *RedisQueue) Enqueue(ctx context.Context, jobType string, payload []byte, opts ...queue.JobOption) (string, error) {
	j := &queue.Job{
		ID:         uuid.NewString(),
		Type:       jobType,
		Payload:    payload,
		Queue:      "default",
		Priority:   0,
		MaxRetries: 3,
		ProcessAt:  time.Now(),
	}
	for _, opt := range opts {
		opt(j)
	}

	asynqOpts := []asynq.Option{
		asynq.TaskID(j.ID),
		asynq.MaxRetry(j.MaxRetries),
		asynq.Queue(j.Queue),
	}
	if j.ProcessAt.After(time.Now()) {
		asynqOpts = append(asynqOpts, asynq.ProcessAt(j.ProcessAt))
	}

	task := asynq.NewTask(jobType, payload)
	info, err := q.client.EnqueueContext(ctx, task, asynqOpts...)
	if err != nil {
		return "", fmt.Errorf("redis_queue: %w", err)
	}
	return info.ID, nil
}

func (q *RedisQueue) RegisterHandler(jobType string, handler queue.Handler) {
	if q.started {
		panic("queue: RegisterHandler called after Start()")
	}
	q.handlers[jobType] = handler
	q.mux.HandleFunc(jobType, func(ctx context.Context, task *asynq.Task) error {
		j := &queue.Job{
			ID:      task.ResultWriter().TaskID(),
			Type:    task.Type(),
			Payload: task.Payload(),
			Queue:   "default", // Asynq doesn't easily expose the current queue name in HandlerFunc
		}
		return handler(ctx, j)
	})
}

func (q *RedisQueue) Start(ctx context.Context) error {
	if q.started {
		return nil
	}
	q.started = true
	go func() {
		if err := q.server.Run(q.mux); err != nil {
			q.logger.ErrorCtx(context.Background(), "redis queue runner failed", err)
		}
	}()
	return nil
}

func (q *RedisQueue) Shutdown(ctx context.Context) error {
	q.server.Shutdown()
	return q.client.Close()
}

func (q *RedisQueue) Inspect(ctx context.Context, jobID string) (*queue.Job, error) {
	inspector := asynq.NewInspector(q.opt)
	// Asynq inspector requires a queue name; search across known queues.
	for _, qname := range []string{"agent", "default", "email"} {
		info, err := inspector.GetTaskInfo(qname, jobID)
		if err != nil {
			continue
		}
		return &queue.Job{
			ID:         info.ID,
			Type:       info.Type,
			Payload:    info.Payload,
			Queue:      info.Queue,
			MaxRetries: info.MaxRetry,
			RetryCount: info.Retried,
			ProcessAt:  info.NextProcessAt,
			CreatedAt:  time.Time{}, // TaskInfo doesn't expose enqueued time
		}, nil
	}
	return nil, context.DeadlineExceeded
}
