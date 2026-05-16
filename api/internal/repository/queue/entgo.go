package queue

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kilip/opus/api/ent"
	"github.com/kilip/opus/api/ent/cronschedule"
	"github.com/kilip/opus/api/ent/deadletter"
	"github.com/kilip/opus/api/ent/job"
	"github.com/kilip/opus/api/internal/model"
)

type entGoDriver struct {
	client *ent.Client
	mu     sync.Mutex // guards Pop() for SQLite dialect only
	driver string     // "sqlite" | "postgres"
}

// NewEntGoDriver creates a new EntGo-based queue driver.
func NewEntGoDriver(client *ent.Client, driver string) QueueDriver {
	return &entGoDriver{
		client: client,
		driver: driver,
	}
}

// Push persists a job to the database (upsert).
func (d *entGoDriver) Push(ctx context.Context, m *model.Job) error {
	err := d.client.Job.
		Create().
		SetID(m.ID).
		SetType(m.Type).
		SetPayload(m.Payload).
		SetPriority(m.Priority).
		SetStatus(string(m.Status)).
		SetRetries(m.Retries).
		SetMaxRetries(m.MaxRetries).
		SetScheduledAt(m.ScheduledAt).
		SetError(m.Error).
		SetUserID(m.UserID).
		OnConflict().
		UpdateNewValues().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to push job: %w", err)
	}
	return nil
}

// Pop atomically retrieves and locks the highest-priority pending job.
func (d *entGoDriver) Pop(ctx context.Context) (*model.Job, error) {
	if d.driver == "sqlite" || d.driver == "sqlite3" {
		d.mu.Lock()
		defer d.mu.Unlock()
	}

	tx, err := d.client.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}

	// Find the best job to run
	j, err := tx.Job.
		Query().
		Where(
			job.Status(string(model.StatusPending)),
			job.ScheduledAtLTE(time.Now()),
		).
		Order(
			ent.Desc(job.FieldPriority),
			ent.Asc(job.FieldScheduledAt),
		).
		First(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			_ = tx.Rollback()
			return nil, nil
		}
		_ = tx.Rollback()
		return nil, fmt.Errorf("failed to pop job: %w", err)
	}

	// Lock the job by setting it to running
	j, err = tx.Job.
		UpdateOne(j).
		SetStatus(string(model.StatusRunning)).
		Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		return nil, fmt.Errorf("failed to lock job: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit pop transaction: %w", err)
	}

	return d.mapJob(j), nil
}

// UpdateStatus updates the status and error message of a job.
func (d *entGoDriver) UpdateStatus(ctx context.Context, id string, status model.JobStatus, errMsg string) error {
	_, err := d.client.Job.
		UpdateOneID(id).
		SetStatus(string(status)).
		SetError(errMsg).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}
	return nil
}

// MoveToDead moves a failed job to the dead letter store.
func (d *entGoDriver) MoveToDead(ctx context.Context, m *model.Job) error {
	// Start transaction
	tx, err := d.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}

	// Create dead letter entry
	_, err = tx.DeadLetter.
		Create().
		SetID(m.ID).
		SetJobID(m.ID).
		SetType(m.Type).
		SetPayload(m.Payload).
		SetLastError(m.Error).
		SetRetries(m.Retries).
		SetUserID(m.UserID).
		Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("failed to create dead letter: %w", err)
	}

	// Delete original job
	err = tx.Job.
		DeleteOneID(m.ID).
		Exec(ctx)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("failed to delete original job: %w", err)
	}

	return tx.Commit()
}


// ListDeadLetters returns a paginated list of dead letter jobs.
func (d *entGoDriver) ListDeadLetters(ctx context.Context, userID string, limit, offset int) ([]*model.DeadLetter, error) {
	query := d.client.DeadLetter.Query()
	if userID != "" {
		query = query.Where(deadletter.UserID(userID))
	}
	dl, err := query.
		Limit(limit).
		Offset(offset).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list dead letters: %w", err)
	}

	result := make([]*model.DeadLetter, len(dl))
	for i, item := range dl {
		result[i] = &model.DeadLetter{
			ID:        item.ID,
			JobID:     item.JobID,
			Type:      item.Type,
			Payload:   item.Payload,
			LastError: item.LastError,
			Retries:   item.Retries,
			UserID:    item.UserID,
			CreatedAt: item.CreatedAt,
		}
	}
	return result, nil
}

// RetryDeadLetter moves a dead letter job back to the pending queue.
func (d *entGoDriver) RetryDeadLetter(ctx context.Context, userID string, id string) error {
	tx, err := d.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}

	dl, err := tx.DeadLetter.Query().
		Where(
			deadletter.ID(id),
			deadletter.UserID(userID),
		).
		Only(ctx)
	if err != nil {
		_ = tx.Rollback()
		if ent.IsNotFound(err) {
			return fmt.Errorf("dead letter not found or unauthorized")
		}
		return fmt.Errorf("failed to get dead letter: %w", err)
	}

	// Create job back
	_, err = tx.Job.
		Create().
		SetID(dl.JobID).
		SetType(dl.Type).
		SetPayload(dl.Payload).
		SetStatus(string(model.StatusPending)).
		SetRetries(0).
		SetMaxRetries(3). // Default to 3 for now
		SetScheduledAt(time.Now()).
		SetUserID(dl.UserID).
		Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("failed to recreate job: %w", err)
	}

	// Delete dead letter
	if err := tx.DeadLetter.DeleteOneID(id).Exec(ctx); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("failed to delete dead letter: %w", err)
	}

	return tx.Commit()
}

// DeleteDeadLetter removes a dead letter job without retrying.
func (d *entGoDriver) DeleteDeadLetter(ctx context.Context, userID string, id string) error {
	_, err := d.client.DeadLetter.Delete().
		Where(
			deadletter.ID(id),
			deadletter.UserID(userID),
		).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete dead letter: %w", err)
	}
	return nil
}

// UpsertCron creates or updates a cron schedule.
func (d *entGoDriver) UpsertCron(ctx context.Context, m *model.CronSchedule) error {
	// Manual upsert since OnConflict extension is not enabled
	exist, err := d.client.CronSchedule.
		Query().
		Where(
			cronschedule.Name(m.Name),
			cronschedule.UserID(m.UserID),
		).
		Exist(ctx)
	if err != nil {
		return fmt.Errorf("failed to check cron existence: %w", err)
	}

	if exist {
		_, err = d.client.CronSchedule.
			Update().
			Where(
				cronschedule.Name(m.Name),
				cronschedule.UserID(m.UserID),
			).
			SetCronExpression(m.CronExpr).
			SetJobType(m.JobType).
			SetPayload(m.Payload).
			SetIsActive(m.IsActive).
			Save(ctx)
	} else {
		_, err = d.client.CronSchedule.
			Create().
			SetID(m.ID).
			SetName(m.Name).
			SetCronExpression(m.CronExpr).
			SetJobType(m.JobType).
			SetPayload(m.Payload).
			SetIsActive(m.IsActive).
			SetUserID(m.UserID).
			Save(ctx)
	}

	if err != nil {
		return fmt.Errorf("failed to upsert cron schedule: %w", err)
	}
	return nil
}

// DeleteCron removes a cron schedule by ID.
func (d *entGoDriver) DeleteCron(ctx context.Context, id string) error {
	err := d.client.CronSchedule.
		DeleteOneID(id).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete cron schedule: %w", err)
	}
	return nil
}

// ListPendingCrons returns all active cron schedules due for execution.
func (d *entGoDriver) ListPendingCrons(ctx context.Context) ([]*model.CronSchedule, error) {
	crons, err := d.client.CronSchedule.
		Query().
		Where(
			cronschedule.IsActive(true),
			cronschedule.Or(
				cronschedule.NextRunAtIsNil(),
				cronschedule.NextRunAtLTE(time.Now()),
			),
		).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list pending crons: %w", err)
	}

	result := make([]*model.CronSchedule, len(crons))
	for i, c := range crons {
		result[i] = d.mapCron(c)
	}
	return result, nil
}

// UpdateCronNextRun updates LastRunAt and NextRunAt after a cron fires.
func (d *entGoDriver) UpdateCronNextRun(ctx context.Context, id string, lastRun, nextRun time.Time) error {
	_, err := d.client.CronSchedule.
		UpdateOneID(id).
		SetLastRunAt(lastRun).
		SetNextRunAt(nextRun).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("failed to update cron next run: %w", err)
	}
	return nil
}

func (d *entGoDriver) mapJob(j *ent.Job) *model.Job {
	return &model.Job{
		ID:          j.ID,
		Type:        j.Type,
		Payload:     j.Payload,
		Priority:    j.Priority,
		Status:      model.JobStatus(j.Status),
		Retries:     j.Retries,
		MaxRetries:  j.MaxRetries,
		ScheduledAt: j.ScheduledAt,
		CreatedAt:   j.CreatedAt,
		UpdatedAt:   j.UpdatedAt,
		UserID:      j.UserID,
		Error:       j.Error,
	}
}

func (d *entGoDriver) mapCron(c *ent.CronSchedule) *model.CronSchedule {
	return &model.CronSchedule{
		ID:        c.ID,
		Name:      c.Name,
		CronExpr:  c.CronExpression,
		JobType:   c.JobType,
		Payload:   c.Payload,
		IsActive:  c.IsActive,
		LastRunAt: c.LastRunAt,
		NextRunAt: c.NextRunAt,
		UserID:    c.UserID,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}
