// Package service holds task business logic. It depends on domain and repo
// only; HTTP concerns stay in handler.
package service

import (
	"context"
	"log/slog"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/kilip/opus/internal/platform/eventbus"
	"github.com/kilip/opus/internal/task/domain"
	"github.com/kilip/opus/internal/task/repo"
)

// ProjectChecker checks whether a project exists.
type ProjectChecker interface {
	Exists(ctx context.Context, id string) (bool, error)
}

// Option configures the Task service.
type Option func(*Task)

// WithProjectChecker sets the ProjectChecker implementation.
func WithProjectChecker(pc ProjectChecker) Option {
	return func(s *Task) {
		s.projectChecker = pc
	}
}

// Task owns task and activity business rules.
type Task struct {
	repo           *repo.Ent
	bus            *eventbus.Bus
	projectChecker ProjectChecker
	mu             sync.Mutex
}

// New returns Task service backed by Ent repo and event bus.
func New(r *repo.Ent, bus *eventbus.Bus, opts ...Option) *Task {
	s := &Task{
		repo: r,
		bus:  bus,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// List returns every task, optionally filtered.
func (s *Task) List(ctx context.Context, filters ...domain.TaskFilter) ([]domain.Task, error) {
	return s.repo.List(ctx, filters...)
}

// CountByProject returns the number of tasks associated with a project ID.
func (s *Task) CountByProject(ctx context.Context, projectID string) (int, error) {
	return s.repo.CountByProject(ctx, projectID)
}

// Get returns task by Opus identifier, or domain.ErrTaskNotFound.
func (s *Task) Get(ctx context.Context, id string) (domain.Task, error) {
	return s.repo.Get(ctx, id)
}

// Create validates req and stores new task.
func (s *Task) Create(ctx context.Context, req domain.CreateTaskRequest) (domain.Task, error) {
	if strings.TrimSpace(req.ProjectID) == "" {
		return domain.Task{}, domain.ErrProjectIDRequired
	}
	if _, err := uuid.Parse(req.ProjectID); err != nil {
		return domain.Task{}, domain.ErrInvalidProjectID
	}
	if strings.TrimSpace(req.Title) == "" {
		return domain.Task{}, domain.ErrTaskTitleEmpty
	}

	var exists bool
	var err error
	if s.projectChecker != nil {
		exists, err = s.projectChecker.Exists(ctx, req.ProjectID)
	} else if s.repo != nil {
		exists, err = s.repo.ProjectExists(ctx, req.ProjectID)
	}
	if err != nil {
		return domain.Task{}, err
	}
	if !exists {
		return domain.Task{}, domain.ErrProjectNotFound
	}

	t, err := s.repo.Create(ctx, repo.CreateInput{
		ProjectID:   req.ProjectID,
		Title:       req.Title,
		Description: req.Description,
		Status:      req.Status,
		Priority:    req.Priority,
		Type:        req.Type,
		Assignee:    req.Assignee,
		DueDate:     req.DueDate,
		Labels:      req.Labels,
	})
	if err != nil {
		return domain.Task{}, err
	}

	if s.bus != nil {
		evt, err := eventbus.NewEvent(domain.EventReceived, t.ID, domain.TaskReceivedPayload{
			ID:         t.ID,
			Key:        t.Key,
			ProjectID:  t.ProjectID,
			Source:     t.Source,
			ExternalID: t.ExternalID,
			Status:     t.Status,
		})
		if err != nil {
			slog.Error("failed to create task.received event", "task_id", t.ID, "error", err)
		} else {
			// ponytail: publish errors are logged, not returned; becomes transactional when the outbox lands
			if err := s.bus.Publish(ctx, evt); err != nil {
				slog.Error("failed to publish task.received event", "task_id", t.ID, "error", err)
			}
		}
	}

	return t, nil
}

// Update validates req and applies it to task id.
func (s *Task) Update(ctx context.Context, id string, req domain.UpdateTaskRequest) (domain.Task, error) {
	if req.Title != nil && strings.TrimSpace(*req.Title) == "" {
		return domain.Task{}, domain.ErrTaskTitleEmpty
	}

	// ponytail: one service-level mutex around update+publish; per-task lock if contention shows up
	s.mu.Lock()
	defer s.mu.Unlock()

	res, err := s.repo.Update(ctx, id, repo.UpdateInput{
		Title:         req.Title,
		Description:   req.Description,
		Status:        req.Status,
		Priority:      req.Priority,
		Type:          req.Type,
		Assignee:      req.Assignee,
		ClearAssignee: req.ClearAssignee,
		DueDate:       req.DueDate,
		Labels:        req.Labels,
	})
	if err != nil {
		return domain.Task{}, err
	}

	if res.StatusChanged && s.bus != nil {
		evt, err := eventbus.NewEvent(domain.EventStatusChanged, res.Task.ID, domain.TaskStatusChangedPayload{
			ID:        res.Task.ID,
			ProjectID: res.Task.ProjectID,
			From:      res.PrevStatus,
			To:        res.Task.Status,
		})
		if err != nil {
			slog.Error("failed to create task.status_changed event", "task_id", res.Task.ID, "error", err)
		} else {
			// ponytail: publish errors are logged, not returned; becomes transactional when the outbox lands
			if err := s.bus.Publish(ctx, evt); err != nil {
				slog.Error("failed to publish task.status_changed event", "task_id", res.Task.ID, "error", err)
			}
		}
	}

	return res.Task, nil
}

// ListActivity returns chatter timeline of task id, oldest first.
func (s *Task) ListActivity(ctx context.Context, id string) ([]domain.Activity, error) {
	return s.repo.ListActivity(ctx, id)
}

// CreateActivity validates req and appends activity to task id.
func (s *Task) CreateActivity(ctx context.Context, id string, req domain.CreateActivityRequest) (domain.Activity, error) {
	if strings.TrimSpace(req.Body) == "" {
		return domain.Activity{}, domain.ErrActivityBodyEmpty
	}
	var kind domain.ActivityKind
	if req.Kind != nil {
		kind = *req.Kind
	}
	return s.repo.AddActivity(ctx, id, kind, "", req.Body)
}
