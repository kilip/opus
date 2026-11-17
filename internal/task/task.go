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

// Event, status, source, priority, type, and activity kind constants re-exported from domain.
const (
	EventReceived      = domain.EventReceived
	EventStatusChanged = domain.EventStatusChanged

	StatusBacklog    = domain.StatusBacklog
	StatusTodo       = domain.StatusTodo
	StatusInProgress = domain.StatusInProgress
	StatusReview     = domain.StatusReview
	StatusBlocked    = domain.StatusBlocked
	StatusDone       = domain.StatusDone

	SourceGitHub = domain.TaskSourceGitHub
	SourceLinear = domain.TaskSourceLinear
	SourceSentry = domain.TaskSourceSentry
	SourceAgent  = domain.TaskSourceAgent
	SourceManual = domain.TaskSourceManual

	PriorityP0 = domain.TaskPriorityP0
	PriorityP1 = domain.TaskPriorityP1
	PriorityP2 = domain.TaskPriorityP2
	PriorityP3 = domain.TaskPriorityP3

	TypeFeat     = domain.TaskTypeFeat
	TypeBug      = domain.TaskTypeBug
	TypeChore    = domain.TaskTypeChore
	TypeRefactor = domain.TaskTypeRefactor
	TypeDocs     = domain.TaskTypeDocs

	ActivityKindStatus  = domain.ActivityKindStatus
	ActivityKindComment = domain.ActivityKindComment
	ActivityKindAssign  = domain.ActivityKindAssign
	ActivityKindCommit  = domain.ActivityKindCommit
	ActivityKindReview  = domain.ActivityKindReview
)

// Sentinel errors re-exported from domain.
var (
	ErrTaskNotFound       = domain.ErrTaskNotFound
	ErrTaskTitleEmpty     = domain.ErrTaskTitleEmpty
	ErrActivityBodyEmpty  = domain.ErrActivityBodyEmpty
	ErrProjectIDRequired  = domain.ErrProjectIDRequired
	ErrInvalidProjectID   = domain.ErrInvalidProjectID
	ErrProjectNotFound    = domain.ErrProjectNotFound
	ErrPatchForbidden     = domain.ErrPatchForbidden
	ErrInvalidStatus      = domain.ErrInvalidStatus
	ErrExternalIDRequired = domain.ErrExternalIDRequired
	ErrSourceRequired     = domain.ErrSourceRequired
	ErrAuthorForbidden    = domain.ErrAuthorForbidden
)

// Re-exported domain types.
type (
	Record                   = domain.Task
	TaskStatus               = domain.TaskStatus
	Status                   = domain.TaskStatus
	TaskSource               = domain.TaskSource
	TaskPriority             = domain.TaskPriority
	TaskType                 = domain.TaskType
	ActivityKind             = domain.ActivityKind
	Activity                 = domain.Activity
	Assignee                 = domain.Assignee
	ImportTaskRequest        = domain.ImportTaskRequest
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

// Get returns a task by Opus identifier, or ErrTaskNotFound.
func (t *Task) Get(ctx context.Context, id string) (Record, error) {
	return t.svc.Get(ctx, id)
}

// Import upserts a task by (source, externalId) for sibling features.
func (t *Task) Import(ctx context.Context, req ImportTaskRequest) (Record, error) {
	return t.svc.Import(ctx, req)
}

// FindByExternal returns the task matching (source, externalId), or ErrTaskNotFound.
func (t *Task) FindByExternal(ctx context.Context, source TaskSource, externalID string) (*Record, error) {
	return t.svc.FindByExternal(ctx, source, externalID)
}

// AddLabel adds a label to task id without publishing events.
func (t *Task) AddLabel(ctx context.Context, id, label string) error {
	return t.svc.AddLabel(ctx, id, label)
}

// RemoveLabel removes a label from task id without publishing events.
func (t *Task) RemoveLabel(ctx context.Context, id, label string) error {
	return t.svc.RemoveLabel(ctx, id, label)
}

// SetLabels replaces all labels on task id without publishing events.
func (t *Task) SetLabels(ctx context.Context, id string, labels []string) error {
	return t.svc.SetLabels(ctx, id, labels)
}

// AddActivity appends an activity entry with optional author to task id.
func (t *Task) AddActivity(ctx context.Context, id string, kind ActivityKind, author, body string) (Activity, error) {
	return t.svc.AddActivity(ctx, id, kind, author, body)
}
