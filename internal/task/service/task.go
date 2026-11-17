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

// FindByExternal returns task by (source, externalID), or domain.ErrTaskNotFound.
func (s *Task) FindByExternal(ctx context.Context, source domain.TaskSource, externalID string) (*domain.Task, error) {
	return s.repo.FindByExternal(ctx, source, externalID)
}

// Create validates req and stores new task.
func (s *Task) Create(ctx context.Context, req domain.CreateTaskRequest) (domain.Task, error) {
	if err := s.validateProjectAndTitle(ctx, req.ProjectID, req.Title); err != nil {
		return domain.Task{}, err
	}
	if req.Status != nil && *req.Status != "" && !req.Status.IsValid() {
		return domain.Task{}, domain.ErrInvalidStatus
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

	s.publishReceived(ctx, t)
	return t, nil
}

// Import upserts a task by (source, externalID) and emits events only on creation or real status transition.
func (s *Task) Import(ctx context.Context, req domain.ImportTaskRequest) (domain.Task, error) {
	if strings.TrimSpace(string(req.Source)) == "" {
		return domain.Task{}, domain.ErrSourceRequired
	}
	if strings.TrimSpace(req.ExternalID) == "" {
		return domain.Task{}, domain.ErrExternalIDRequired
	}
	if err := s.validateProjectAndTitle(ctx, req.ProjectID, req.Title); err != nil {
		return domain.Task{}, err
	}
	if req.Status != "" && !req.Status.IsValid() {
		return domain.Task{}, domain.ErrInvalidStatus
	}

	var statusPtr *domain.TaskStatus
	if req.Status != "" {
		st := req.Status
		statusPtr = &st
	}
	var priorityPtr *domain.TaskPriority
	if req.Priority != "" {
		p := req.Priority
		priorityPtr = &p
	}
	var typePtr *domain.TaskType
	if req.Type != "" {
		tp := req.Type
		typePtr = &tp
	}

	// ponytail: one service-level mutex around write+publish; per-task lock if contention shows up
	s.mu.Lock()
	defer s.mu.Unlock()

	res, err := s.repo.Import(ctx, repo.CreateInput{
		ProjectID:    req.ProjectID,
		Key:          req.Key,
		Source:       req.Source,
		ExternalID:   req.ExternalID,
		ExternalURL:  req.ExternalURL,
		SourceStatus: req.SourceStatus,
		Title:        req.Title,
		Description:  req.Description,
		Status:       statusPtr,
		Priority:     priorityPtr,
		Type:         typePtr,
		Labels:       req.Labels,
		SyncedAt:     req.SyncedAt,
	})
	if err != nil {
		return domain.Task{}, err
	}

	if res.Created {
		s.publishReceived(ctx, res.Task)
	} else if res.StatusChanged {
		s.publishStatusChanged(ctx, res.Task.ID, res.Task.ProjectID, res.PrevStatus, res.Task.Status)
	}

	return res.Task, nil
}

// Update validates req and applies it to task id.
func (s *Task) Update(ctx context.Context, id string, req domain.UpdateTaskRequest) (domain.Task, error) {
	if req.Title != nil && strings.TrimSpace(*req.Title) == "" {
		return domain.Task{}, domain.ErrTaskTitleEmpty
	}
	if req.Status != nil && *req.Status != "" && !req.Status.IsValid() {
		return domain.Task{}, domain.ErrInvalidStatus
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

	if res.StatusChanged {
		s.publishStatusChanged(ctx, res.Task.ID, res.Task.ProjectID, res.PrevStatus, res.Task.Status)
	}

	return res.Task, nil
}

// AddLabel adds label to task id without publishing events.
func (s *Task) AddLabel(ctx context.Context, id, label string) error {
	label = strings.TrimSpace(label)
	if label == "" {
		_, err := s.repo.Get(ctx, id)
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.repo.AddLabel(ctx, id, label)
}

// RemoveLabel removes label from task id without publishing events.
func (s *Task) RemoveLabel(ctx context.Context, id, label string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.repo.RemoveLabel(ctx, id, label)
}

// SetLabels replaces all labels on task id without publishing events.
func (s *Task) SetLabels(ctx context.Context, id string, labels []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.repo.SetLabels(ctx, id, labels)
}

// ListActivity returns chatter timeline of task id, oldest first.
func (s *Task) ListActivity(ctx context.Context, id string) ([]domain.Activity, error) {
	return s.repo.ListActivity(ctx, id)
}

// CreateActivity validates req and appends activity to task id with no author.
func (s *Task) CreateActivity(ctx context.Context, id string, req domain.CreateActivityRequest) (domain.Activity, error) {
	var kind domain.ActivityKind
	if req.Kind != nil {
		kind = *req.Kind
	}
	return s.AddActivity(ctx, id, kind, "", req.Body)
}

// AddActivity validates body and appends an activity with optional author to task id.
func (s *Task) AddActivity(ctx context.Context, id string, kind domain.ActivityKind, author, body string) (domain.Activity, error) {
	if strings.TrimSpace(body) == "" {
		return domain.Activity{}, domain.ErrActivityBodyEmpty
	}
	return s.repo.AddActivity(ctx, id, kind, strings.TrimSpace(author), body)
}

func (s *Task) validateProjectAndTitle(ctx context.Context, projectID, title string) error {
	if strings.TrimSpace(projectID) == "" {
		return domain.ErrProjectIDRequired
	}
	if _, err := uuid.Parse(projectID); err != nil {
		return domain.ErrInvalidProjectID
	}
	if strings.TrimSpace(title) == "" {
		return domain.ErrTaskTitleEmpty
	}

	var exists bool
	var err error
	if s.projectChecker != nil {
		exists, err = s.projectChecker.Exists(ctx, projectID)
	} else if s.repo != nil {
		exists, err = s.repo.ProjectExists(ctx, projectID)
	}
	if err != nil {
		return err
	}
	if !exists {
		return domain.ErrProjectNotFound
	}
	return nil
}

func (s *Task) publishReceived(ctx context.Context, t domain.Task) {
	if s.bus == nil {
		return
	}
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
		return
	}
	// ponytail: publish errors are logged, not returned; becomes transactional when the outbox lands
	if err := s.bus.Publish(ctx, evt); err != nil {
		slog.Error("failed to publish task.received event", "task_id", t.ID, "error", err)
	}
}

func (s *Task) publishStatusChanged(ctx context.Context, id, projectID string, from, to domain.TaskStatus) {
	if s.bus == nil {
		return
	}
	evt, err := eventbus.NewEvent(domain.EventStatusChanged, id, domain.TaskStatusChangedPayload{
		ID:        id,
		ProjectID: projectID,
		From:      from,
		To:        to,
	})
	if err != nil {
		slog.Error("failed to create task.status_changed event", "task_id", id, "error", err)
		return
	}
	// ponytail: publish errors are logged, not returned; becomes transactional when the outbox lands
	if err := s.bus.Publish(ctx, evt); err != nil {
		slog.Error("failed to publish task.status_changed event", "task_id", id, "error", err)
	}
}
