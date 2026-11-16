// Package project is the project feature. Other packages depend on this package only,
// never on its subpackages.
package project

import (
	"github.com/gofiber/fiber/v3"
	"github.com/kilip/opus/ent"
	"github.com/kilip/opus/internal/project/domain"
	"github.com/kilip/opus/internal/project/handler"
	"github.com/kilip/opus/internal/project/repo"
	"github.com/kilip/opus/internal/project/service"
)

// Sentinel errors re-exported from domain.
var (
	ErrProjectNotFound = domain.ErrProjectNotFound
	ErrInvalidSlug     = domain.ErrInvalidSlug
	ErrDuplicateSlug   = domain.ErrDuplicateSlug
	ErrInvalidRepo     = domain.ErrInvalidRepo
	ErrPatchForbidden  = domain.ErrPatchForbidden
	ErrProjectHasTasks = domain.ErrProjectHasTasks
)

// Re-exported domain types.
type (
	CreateProjectRequest = domain.CreateProjectRequest
	UpdateProjectRequest = domain.UpdateProjectRequest
	TaskCounter          = service.TaskCounter
)

// Option configures the Project feature.
type Option func(*options)

type options struct {
	taskCounter service.TaskCounter
}

// WithTaskCounter configures the TaskCounter hook for project deletion guard.
func WithTaskCounter(tc service.TaskCounter) Option {
	return func(o *options) {
		o.taskCounter = tc
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
		TaskCounter: o.taskCounter,
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
