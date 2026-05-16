package service

import (
	"context"

	"github.com/kilip/opus/api/internal/model"
)

// Queue is the entry point for enqueuing background jobs.
type Queue interface {
	// Push submits a job to the queue for asynchronous execution.
	Push(ctx context.Context, job *model.Job) error
}

// Scheduler manages cron-based recurring job schedules.
type Scheduler interface {
	// AddCron registers or updates a cron schedule.
	AddCron(ctx context.Context, schedule *model.CronSchedule) error
	// RemoveCron deactivates a cron schedule by ID.
	RemoveCron(ctx context.Context, id string) error
	// Start begins the scheduler background ticker.
	Start(ctx context.Context) error
	// Stop gracefully halts the scheduler.
	Stop() error
}
