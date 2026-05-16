package worker

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"github.com/kilip/opus/api/internal/model"
	"github.com/kilip/opus/api/internal/repository/queue"
	"github.com/kilip/opus/api/internal/service"
)

type cronScheduler struct {
	driver   queue.QueueDriver
	logger   *slog.Logger
	parser   cron.Parser
	stopChan chan struct{}
	wg       sync.WaitGroup
}

// NewScheduler creates a new cron-based job scheduler.
func NewScheduler(driver queue.QueueDriver, logger *slog.Logger) service.Scheduler {
	return &cronScheduler{
		driver:   driver,
		logger:   logger.With("component", "scheduler"),
		parser:   cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow),
		stopChan: make(chan struct{}),
	}
}

// AddCron registers or updates a cron schedule.
func (s *cronScheduler) AddCron(ctx context.Context, m *model.CronSchedule) error {
	// Validate cron expression
	if _, err := s.parser.Parse(m.CronExpr); err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}

	if m.ID == "" {
		m.ID = uuid.New().String()
	}

	s.logger.Info("Adding/Updating cron schedule", "name", m.Name, "expr", m.CronExpr)
	return s.driver.UpsertCron(ctx, m)
}

// RemoveCron deactivates a cron schedule.
func (s *cronScheduler) RemoveCron(ctx context.Context, id string) error {
	s.logger.Info("Removing cron schedule", "id", id)
	return s.driver.DeleteCron(ctx, id)
}

// Start starts the scheduler background ticker.
func (s *cronScheduler) Start(ctx context.Context) error {
	s.logger.Info("Starting cron scheduler")
	s.wg.Add(1)
	go s.run(ctx)
	return nil
}

// Stop stops the scheduler gracefully.
func (s *cronScheduler) Stop() error {
	s.logger.Info("Stopping cron scheduler")
	close(s.stopChan)
	s.wg.Wait()
	s.logger.Info("Cron scheduler stopped")
	return nil
}

func (s *cronScheduler) run(ctx context.Context) {
	defer s.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Run once at start
	s.processCrons(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.processCrons(ctx)
		}
	}
}

func (s *cronScheduler) processCrons(ctx context.Context) {
	crons, err := s.driver.ListPendingCrons(ctx)
	if err != nil {
		s.logger.Error("Failed to list pending crons", "error", err)
		return
	}

	if len(crons) > 0 {
		s.logger.Debug("Processing pending crons", "count", len(crons))
	}

	for _, c := range crons {
		s.fireCron(ctx, c)
	}
}

func (s *cronScheduler) fireCron(ctx context.Context, c *model.CronSchedule) {
	logger := s.logger.With("cron_id", c.ID, "cron_name", c.Name)
	logger.Info("Firing cron schedule")

	sched, err := s.parser.Parse(c.CronExpr)
	if err != nil {
		logger.Error("Failed to parse cron expression during execution", "error", err)
		return
	}

	now := time.Now()
	nextRun := sched.Next(now)

	job := &model.Job{
		ID:          uuid.New().String(),
		Type:        c.JobType,
		Payload:     c.Payload,
		Priority:    1,
		Status:      model.StatusPending,
		MaxRetries:  3,
		ScheduledAt: now,
		UserID:      c.UserID,
	}

	if err := s.driver.Push(ctx, job); err != nil {
		logger.Error("Failed to enqueue cron job", "error", err)
		return
	}

	if err := s.driver.UpdateCronNextRun(ctx, c.ID, now, nextRun); err != nil {
		logger.Error("Failed to update cron next run", "error", err)
	}
}
