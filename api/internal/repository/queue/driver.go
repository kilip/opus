package queue

import (
	"context"
	"time"

	"github.com/kilip/opus/api/internal/model"
)

// QueueDriver is the persistence abstraction for the queue system.
// Implementations: entgo.go (PostgreSQL/SQLite), redis.go (Redis).
type QueueDriver interface {
	// Push persists a job to the queue backend.
	Push(ctx context.Context, job *model.Job) error
	// Pop atomically retrieves and locks the highest-priority pending job.
	// Returns nil, nil if no job is available.
	Pop(ctx context.Context) (*model.Job, error)
	// UpdateStatus updates the status and error message of a job.
	UpdateStatus(ctx context.Context, id string, status model.JobStatus, errMsg string) error
	// MoveToDead moves a failed job to the dead letter store.
	MoveToDead(ctx context.Context, job *model.Job) error

	// UpsertCron creates or updates a cron schedule.
	UpsertCron(ctx context.Context, cron *model.CronSchedule) error
	// DeleteCron removes a cron schedule by ID.
	DeleteCron(ctx context.Context, id string) error
	// ListPendingCrons returns all active cron schedules due for execution.
	ListPendingCrons(ctx context.Context) ([]*model.CronSchedule, error)
	// UpdateCronNextRun updates LastRunAt and NextRunAt after a cron fires.
	UpdateCronNextRun(ctx context.Context, id string, lastRun, nextRun time.Time) error
}
