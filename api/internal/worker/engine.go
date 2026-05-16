package worker

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/kilip/opus/api/internal/model"
	"github.com/kilip/opus/api/internal/repository/queue"
	"github.com/kilip/opus/api/internal/service"
)

type workerEngine struct {
	driver      queue.QueueDriver
	concurrency int
	handlers    map[string]service.HandlerFunc
	stopChan    chan struct{}
	wg          sync.WaitGroup
	logger      *slog.Logger
	mu          sync.RWMutex
}

// NewWorkerEngine creates a new background job worker engine.
func NewWorkerEngine(driver queue.QueueDriver, concurrency int, logger *slog.Logger) service.Worker {
	if concurrency <= 0 {
		concurrency = 1
	}
	return &workerEngine{
		driver:      driver,
		concurrency: concurrency,
		handlers:    make(map[string]service.HandlerFunc),
		stopChan:    make(chan struct{}),
		logger:      logger.With("component", "worker_engine"),
	}
}

// Register registers a handler for a specific job type.
func (e *workerEngine) Register(jobType string, handler service.HandlerFunc) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.handlers[jobType] = handler
	e.logger.Debug("Registered job handler", "type", jobType)
}

// Start starts the worker pool.
func (e *workerEngine) Start(ctx context.Context) error {
	e.logger.Info("Starting worker pool", "workers", e.concurrency)
	for i := 0; i < e.concurrency; i++ {
		e.wg.Add(1)
		go e.worker(ctx, i)
	}
	return nil
}

// Stop stops the worker pool gracefully.
func (e *workerEngine) Stop() error {
	e.logger.Info("Stopping worker pool")
	close(e.stopChan)
	e.wg.Wait()
	e.logger.Info("Worker pool stopped")
	return nil
}

func (e *workerEngine) worker(ctx context.Context, id int) {
	defer e.wg.Done()
	logger := e.logger.With("worker_id", id)
	logger.Debug("Worker started")

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-e.stopChan:
			return
		case <-ticker.C:
			job, err := e.driver.Pop(ctx)
			if err != nil {
				logger.Error("Failed to pop job", "error", err)
				continue
			}
			if job == nil {
				continue
			}

			e.process(ctx, job)
		}
	}
}

func (e *workerEngine) process(ctx context.Context, job *model.Job) {
	logger := e.logger.With("job_id", job.ID, "job_type", job.Type)
	logger.Info("Processing job")

	e.mu.RLock()
	handler, ok := e.handlers[job.Type]
	e.mu.RUnlock()

	if !ok {
		err := fmt.Errorf("no handler registered for job type: %s", job.Type)
		logger.Error("Job processing failed: handler not found")
		e.failJob(ctx, job, err)
		return
	}

	err := handler(ctx, job)
	if err != nil {
		logger.Error("Job processing failed", "error", err)
		e.failJob(ctx, job, err)
		return
	}

	logger.Info("Job completed successfully")
	if err := e.driver.UpdateStatus(ctx, job.ID, model.StatusCompleted, ""); err != nil {
		logger.Error("Failed to update job status to completed", "error", err)
	}
}

func (e *workerEngine) failJob(ctx context.Context, job *model.Job, err error) {
	job.Retries++
	job.Error = err.Error()

	if job.Retries >= job.MaxRetries {
		e.logger.Error("Job failed permanently", "job_id", job.ID, "retries", job.Retries, "max_retries", job.MaxRetries)
		if moveErr := e.driver.MoveToDead(ctx, job); moveErr != nil {
			e.logger.Error("Failed to move job to dead letter", "job_id", job.ID, "error", moveErr)
		}
		return
	}

	// Exponential backoff: 2^retries * minute
	backoff := time.Duration(1<<uint(job.Retries)) * time.Minute
	job.ScheduledAt = time.Now().Add(backoff)
	job.Status = model.StatusPending

	e.logger.Info("Rescheduling job for retry", "job_id", job.ID, "next_run", job.ScheduledAt, "retry", job.Retries)
	if resErr := e.driver.Push(ctx, job); resErr != nil {
		e.logger.Error("Failed to reschedule job", "job_id", job.ID, "error", resErr)
	}
}
