// Package task is the task feature. Other packages depend on this package only,
// never on its subpackages.
package task

import (
	"context"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/opus/ent"
	"github.com/kilip/opus/internal/platform/eventbus"
	"github.com/kilip/opus/internal/task/domain"
	"github.com/kilip/opus/internal/task/handler"
	"github.com/kilip/opus/internal/task/repo"
	"github.com/kilip/opus/internal/task/service"
)

// Event type constants re-exported from domain.
const (
	EventReceived      = domain.EventReceived
	EventStatusChanged = domain.EventStatusChanged

	StatusBacklog    = domain.TaskStatusBacklog
	StatusTodo       = domain.TaskStatusTodo
	StatusInProgress = domain.TaskStatusInProgress
	StatusReview     = domain.TaskStatusReview
	StatusDone       = domain.TaskStatusDone
)

// Sentinel errors re-exported from domain.
var (
	ErrTaskNotFound      = domain.ErrTaskNotFound
	ErrTaskTitleEmpty    = domain.ErrTaskTitleEmpty
	ErrActivityBodyEmpty = domain.ErrActivityBodyEmpty
	ErrProjectIDRequired = domain.ErrProjectIDRequired
	ErrInvalidProjectID  = domain.ErrInvalidProjectID
	ErrProjectNotFound   = domain.ErrProjectNotFound
	ErrPatchForbidden    = domain.ErrPatchForbidden
)

// Re-exported domain types.
type (
	TaskStatus               = domain.TaskStatus
	Status                   = domain.TaskStatus
	TaskReceivedPayload      = domain.TaskReceivedPayload
	TaskStatusChangedPayload = domain.TaskStatusChangedPayload
	ProjectChecker           = service.ProjectChecker
)

// Option configures the Task feature.
type Option func(*options)

type options struct {
	projectChecker service.ProjectChecker
}

// WithProjectChecker sets the ProjectChecker implementation for project validation.
func WithProjectChecker(pc service.ProjectChecker) Option {
	return func(o *options) {
		o.projectChecker = pc
	}
}

// Task is the task feature facade.
type Task struct {
	handler *handler.Handler
	svc     *service.Task
}

// New returns a new Task feature facade backed by Ent.
func New(client *ent.Client, bus *eventbus.Bus, opts ...Option) *Task {
	var o options
	for _, opt := range opts {
		opt(&o)
	}
	r := repo.New(client)
	var svcOpts []service.Option
	if o.projectChecker != nil {
		svcOpts = append(svcOpts, service.WithProjectChecker(o.projectChecker))
	}
	svc := service.New(r, bus, svcOpts...)
	return &Task{
		handler: handler.New(svc),
		svc:     svc,
	}
}

// Register mounts the task HTTP routes on r.
func (t *Task) Register(r fiber.Router) {
	t.handler.Register(r)
}

// CountByProject returns the number of tasks associated with a project ID.
func (t *Task) CountByProject(ctx context.Context, projectID string) (int, error) {
	return t.svc.CountByProject(ctx, projectID)
}
