// Package service holds task business logic. It depends on domain and repo
// only; HTTP concerns stay in handler.
package service

import (
	"strings"

	"github.com/kilip/opus/internal/task/domain"
	"github.com/kilip/opus/internal/task/repo"
)

// Task owns task and activity business rules.
type Task struct {
	repo *repo.Memory
}

// New returns Task service backed by Memory repo.
func New(r *repo.Memory) *Task {
	return &Task{repo: r}
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
	return s.repo.Create(repo.CreateInput{
		Title:       req.Title,
		Description: req.Description,
		Status:      req.Status,
		Priority:    req.Priority,
		Type:        req.Type,
		Assignee:    req.Assignee,
		DueDate:     req.DueDate,
		Labels:      req.Labels,
	}), nil
}

// Update validates req and applies it to task id.
func (s *Task) Update(id string, req domain.UpdateTaskRequest) (domain.Task, error) {
	if req.Title != nil && strings.TrimSpace(*req.Title) == "" {
		return domain.Task{}, domain.ErrTaskTitleEmpty
	}
	return s.repo.Update(id, repo.UpdateInput{
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
