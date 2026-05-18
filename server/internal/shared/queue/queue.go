// Package queue provides the interfaces and types for the Opus asynchronous
// messaging infrastructure.
package queue

import (
	"context"
	"time"
)

// JobStatus represents the lifecycle state of a queued job.
type JobStatus string

const (
	// JobStatusPending indicates the job is waiting to be processed.
	JobStatusPending JobStatus = "pending"
	// JobStatusRunning indicates the job is currently being processed.
	JobStatusRunning JobStatus = "running"
	// JobStatusCompleted indicates the job completed successfully.
	JobStatusCompleted JobStatus = "completed"
	// JobStatusFailed indicates the job failed and may be retried.
	JobStatusFailed JobStatus = "failed"
	// JobStatusDead indicates the job exceeded max retries and is dead-lettered.
	JobStatusDead JobStatus = "dead"
)

// Job represents a background task.
type Job struct {
	ID         string    `json:"id"`
	Type       string    `json:"type"`
	Queue      string    `json:"queue"`
	Payload    []byte    `json:"payload"`
	Status     JobStatus `json:"status"`
	Priority   int       `json:"priority"`
	MaxRetries int       `json:"max_retries"`
	RetryCount int       `json:"retry_count"`
	ProcessAt  time.Time `json:"process_at"`
	CreatedAt  time.Time `json:"created_at"`
}

// JobOption configures a Job before enqueueing.
type JobOption func(*Job)

// Handler processes a job.
type Handler func(ctx context.Context, job *Job) error

// WithPriority sets the job priority (higher executes first).
func WithPriority(priority int) JobOption {
	return func(j *Job) {
		j.Priority = priority
	}
}

// WithDelay delays execution by the given duration.
func WithDelay(d time.Duration) JobOption {
	return func(j *Job) {
		j.ProcessAt = time.Now().Add(d)
	}
}

// WithProcessAt sets a specific execution time.
func WithProcessAt(t time.Time) JobOption {
	return func(j *Job) {
		j.ProcessAt = t
	}
}

// WithMaxRetries sets the maximum number of retry attempts.
func WithMaxRetries(max int) JobOption {
	return func(j *Job) {
		j.MaxRetries = max
	}
}

// WithQueue specifies the queue name.
func WithQueue(q string) JobOption {
	return func(j *Job) {
		j.Queue = q
	}
}

// Mocks live in server/mocks/ — regenerate with task mocks.

// Queue defines the contract for all durable background job processing.
// Implementations must be safe for concurrent use.
type Queue interface {
	Enqueue(ctx context.Context, jobType string, payload []byte, opts ...JobOption) (string, error)
	RegisterHandler(jobType string, handler Handler)
	Start(ctx context.Context) error
	Shutdown(ctx context.Context) error
	Inspect(ctx context.Context, jobID string) (*Job, error)
}
