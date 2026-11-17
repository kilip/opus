// Package project is the project feature. Other packages depend on this package only,
// never on its subpackages.
package project

import (
	"context"
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/opus/ent"
	"github.com/kilip/opus/internal/project/domain"
	"github.com/kilip/opus/internal/project/handler"
	"github.com/kilip/opus/internal/project/repo"
	"github.com/kilip/opus/internal/project/service"
)

// Sentinel errors re-exported from domain.
var (
	ErrProjectNotFound      = domain.ErrProjectNotFound
	ErrInvalidSlug          = domain.ErrInvalidSlug
	ErrDuplicateSlug        = domain.ErrDuplicateSlug
	ErrInvalidRepo          = domain.ErrInvalidRepo
	ErrPatchForbidden       = domain.ErrPatchForbidden
	ErrProjectHasTasks      = domain.ErrProjectHasTasks
	ErrInvalidPath          = domain.ErrInvalidPath
	ErrNodeNotFound         = domain.ErrNodeNotFound
	ErrInvalidTrackerLabels = domain.ErrInvalidTrackerLabels
)

// Re-exported domain types.
type (
	Record                       = domain.Project
	CreateProjectRequest         = domain.CreateProjectRequest
	UpdateProjectRequest         = domain.UpdateProjectRequest
	ProjectNodeResponse          = domain.ProjectNodeResponse
	UpdateProjectNodePathRequest = domain.UpdateProjectNodePathRequest
	TaskCounter                  = service.TaskCounter
	NodeReader                   = service.NodeReader
	TrackerLabelValidator        = service.TrackerLabelValidator
)

// Option configures the Project feature.
type Option func(*options)

type options struct {
	taskCounter           service.TaskCounter
	nodeReader            service.NodeReader
	trackerLabelValidator service.TrackerLabelValidator
}

// WithTaskCounter configures the TaskCounter hook for project deletion guard.
func WithTaskCounter(tc service.TaskCounter) Option {
	return func(o *options) {
		o.taskCounter = tc
	}
}

// WithNodeReader configures the NodeReader for project node path mapping.
func WithNodeReader(nr service.NodeReader) Option {
	return func(o *options) {
		o.nodeReader = nr
	}
}

// WithTrackerLabelValidator configures the TrackerLabelValidator for project tracker_labels overrides.
func WithTrackerLabelValidator(v service.TrackerLabelValidator) Option {
	return func(o *options) {
		o.trackerLabelValidator = v
	}
}

// Project is the project feature facade.
type Project struct {
	handler *handler.Handler
	svc     *service.Project
}

// New returns a new Project feature facade backed by Ent.
func New(client *ent.Client, opts ...Option) *Project {
	var o options
	for _, opt := range opts {
		opt(&o)
	}
	r := repo.New(client)
	svc := service.New(r, service.Config{
		TaskCounter:           o.taskCounter,
		NodeReader:            o.nodeReader,
		TrackerLabelValidator: o.trackerLabelValidator,
	})
	return &Project{
		handler: handler.New(svc),
		svc:     svc,
	}
}

// Register mounts the project HTTP routes on r.
func (p *Project) Register(r fiber.Router) {
	p.handler.Register(r)
}

// List returns all projects ordered by id.
func (p *Project) List(ctx context.Context) ([]Record, error) {
	return p.svc.List(ctx)
}

// Get returns a single project by id, or domain.ErrProjectNotFound.
func (p *Project) Get(ctx context.Context, id string) (domain.Project, error) {
	return p.svc.Get(ctx, id)
}

// Exists checks if a project exists by id.
func (p *Project) Exists(ctx context.Context, id string) (bool, error) {
	if p == nil {
		return false, nil
	}
	_, err := p.svc.Get(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrProjectNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
