package service

import (
	"context"

	"github.com/kilip/opus/internal/project/domain"
	"github.com/kilip/opus/internal/project/repo"
)

// TaskCounter returns the number of tasks associated with a project ID.
type TaskCounter func(ctx context.Context, projectID string) (int, error)

// Config configures the Project service.
type Config struct {
	TaskCounter TaskCounter
}

// Project owns project business logic.
type Project struct {
	repo        *repo.Ent
	taskCounter TaskCounter
}

// New returns a new Project service backed by Ent repo.
func New(r *repo.Ent, cfg ...Config) *Project {
	var c Config
	if len(cfg) > 0 {
		c = cfg[0]
	}
	return &Project{
		repo:        r,
		taskCounter: c.TaskCounter,
	}
}

// List returns all projects ordered by id.
func (s *Project) List(ctx context.Context) ([]domain.Project, error) {
	return s.repo.List(ctx)
}

// Get returns a single project by id, or domain.ErrProjectNotFound.
func (s *Project) Get(ctx context.Context, id string) (domain.Project, error) {
	return s.repo.Get(ctx, id)
}

// Create validates and creates a new project with defaults.
func (s *Project) Create(ctx context.Context, req domain.CreateProjectRequest) (domain.Project, error) {
	if err := domain.ValidateSlug(req.Slug); err != nil {
		return domain.Project{}, err
	}
	if err := domain.ValidateRepoProvider(req.RepoProvider); err != nil {
		return domain.Project{}, err
	}
	if err := domain.ValidateRepoRef(req.RepoRef); err != nil {
		return domain.Project{}, err
	}

	trackerProvider := req.TrackerProvider
	if trackerProvider == "" {
		trackerProvider = req.RepoProvider
	}
	trackerRef := req.TrackerRef
	if trackerRef == "" {
		trackerRef = req.RepoRef
	}
	defaultBranch := req.DefaultBranch
	if defaultBranch == "" {
		defaultBranch = domain.DefaultBranchMain
	}

	p := domain.Project{
		Slug:            req.Slug,
		Name:            req.Name,
		RepoProvider:    req.RepoProvider,
		RepoRef:         req.RepoRef,
		TrackerProvider: trackerProvider,
		TrackerRef:      trackerRef,
		DefaultBranch:   defaultBranch,
	}

	return s.repo.Create(ctx, p)
}

// Update updates fields of an existing project.
func (s *Project) Update(ctx context.Context, id string, req domain.UpdateProjectRequest) (domain.Project, error) {
	return s.repo.Update(ctx, id, req)
}

// Delete removes a project by id after checking the task counter guard.
func (s *Project) Delete(ctx context.Context, id string) error {
	if _, err := s.repo.Get(ctx, id); err != nil {
		return err
	}

	// ponytail: TaskCounter hook injected; when project-task relation is added to Ent schema, query task count directly via Ent
	if s.taskCounter != nil {
		count, err := s.taskCounter(ctx, id)
		if err != nil {
			return err
		}
		if count > 0 {
			return domain.ErrProjectHasTasks
		}
	}

	return s.repo.Delete(ctx, id)
}
