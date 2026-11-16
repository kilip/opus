// Package service holds task business logic. It depends on domain and repo
// only; HTTP concerns stay in handler.
package service

import (
	"context"
	"log/slog"
	"strings"
	"sync"

	"github.com/kilip/opus/internal/platform/eventbus"
	"github.com/kilip/opus/internal/task/domain"
	"github.com/kilip/opus/internal/task/repo"
)

// Task owns task and activity business rules.
type Task struct {
	repo *repo.Memory
	bus  *eventbus.Bus
	mu   sync.Mutex
}

// New returns Task service backed by Memory repo and event bus.
func New(r *repo.Memory, bus *eventbus.Bus) *Task {
	return &Task{
		repo: r,
		bus:  bus,
	}
}

// List returns every task.
func (s *Task) List() []domain.Task {
	return s.repo.List()
}

// Get returns task by Opus identifier, or domain.ErrTaskNotFound.
func (s *Task) Get(id string) (domain.Task, error) {
	return s.repo.Get(id)
}

// Create validates req and stores new task.
func (s *Task) Create(req domain.CreateTaskRequest) (domain.Task, error) {
	if strings.TrimSpace(req.Title) == "" {
		return domain.Task{}, domain.ErrTaskTitleEmpty
	}
	t := s.repo.Create(repo.CreateInput{
		Title:       req.Title,
		Description: req.Description,
		Status:      req.Status,
		Priority:    req.Priority,
		Type:        req.Type,
		Assignee:    req.Assignee,
		DueDate:     req.DueDate,
		Labels:      req.Labels,
	})

	if s.bus != nil {
		evt, err := eventbus.NewEvent(domain.EventReceived, t.ID, domain.TaskReceivedPayload{
			ID:         t.ID,
			Key:        t.Key,
			Source:     t.Source,
			ExternalID: t.ExternalID,
			Status:     t.Status,
		})
		if err != nil {
			slog.Error("failed to create task.received event", "task_id", t.ID, "error", err)
		} else {
			// ponytail: publish errors are logged, not returned; becomes transactional when the outbox lands
			if err := s.bus.Publish(context.Background(), evt); err != nil {
				slog.Error("failed to publish task.received event", "task_id", t.ID, "error", err)
			}
		}
	}

	return t, nil
}

// Update validates req and applies it to task id.
func (s *Task) Update(id string, req domain.UpdateTaskRequest) (domain.Task, error) {
	if req.Title != nil && strings.TrimSpace(*req.Title) == "" {
		return domain.Task{}, domain.ErrTaskTitleEmpty
	}

	// ponytail: one service-level mutex around update+publish; per-task lock if contention shows up
	s.mu.Lock()
	defer s.mu.Unlock()

	res, err := s.repo.Update(id, repo.UpdateInput{
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
			ID:   res.Task.ID,
			From: res.PrevStatus,
			To:   res.Task.Status,
		})
		if err != nil {
			slog.Error("failed to create task.status_changed event", "task_id", res.Task.ID, "error", err)
		} else {
			// ponytail: publish errors are logged, not returned; becomes transactional when the outbox lands
			if err := s.bus.Publish(context.Background(), evt); err != nil {
				slog.Error("failed to publish task.status_changed event", "task_id", res.Task.ID, "error", err)
			}
		}
	}

	return res.Task, nil
}

// ListActivity returns chatter timeline of task id, oldest first.
func (s *Task) ListActivity(id string) ([]domain.Activity, error) {
	return s.repo.ListActivity(id)
}

// CreateActivity validates req and appends activity to task id.
func (s *Task) CreateActivity(id string, req domain.CreateActivityRequest) (domain.Activity, error) {
	if strings.TrimSpace(req.Body) == "" {
		return domain.Activity{}, domain.ErrActivityBodyEmpty
	}
	var kind domain.ActivityKind
	if req.Kind != nil {
		kind = *req.Kind
	}
	return s.repo.AddActivity(id, kind, req.Author, req.Body)
}
