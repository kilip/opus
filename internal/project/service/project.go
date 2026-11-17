package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/kilip/opus/internal/node"
	"github.com/kilip/opus/internal/project/domain"
	"github.com/kilip/opus/internal/project/repo"
)

// TaskCounter returns the number of tasks associated with a project ID.
type TaskCounter func(ctx context.Context, projectID string) (int, error)

// TrackerLabelValidator validates project-level tracker status-label overrides.
type TrackerLabelValidator func(ctx context.Context, labels map[string]string) error

// NodeReader provides node query capabilities to the project service.
type NodeReader interface {
	List(ctx context.Context) ([]node.NodeResponse, error)
	Get(ctx context.Context, id string) (node.NodeResponse, error)
}

// Config configures the Project service.
type Config struct {
	TaskCounter           TaskCounter
	NodeReader            NodeReader
	TrackerLabelValidator TrackerLabelValidator
}

// Project owns project business logic.
type Project struct {
	repo                  *repo.Ent
	taskCounter           TaskCounter
	nodeReader            NodeReader
	trackerLabelValidator TrackerLabelValidator
}

// New returns a new Project service backed by Ent repo.
func New(r *repo.Ent, cfg ...Config) *Project {
	var c Config
	if len(cfg) > 0 {
		c = cfg[0]
	}
	return &Project{
		repo:                  r,
		taskCounter:           c.TaskCounter,
		nodeReader:            c.NodeReader,
		trackerLabelValidator: c.TrackerLabelValidator,
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
	if req.ClearTrackerLabels || (req.TrackerLabels != nil && len(req.TrackerLabels) == 0) {
		req.ClearTrackerLabels = true
		req.TrackerLabels = nil
	} else if len(req.TrackerLabels) > 0 && s.trackerLabelValidator != nil {
		if err := s.trackerLabelValidator(ctx, req.TrackerLabels); err != nil {
			return domain.Project{}, fmt.Errorf("%w: %w", domain.ErrInvalidTrackerLabels, err)
		}
	}
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

// ListNodeMappings returns all nodes with their path mapping for the project.
// If a node has no mapping, its path is nil.
func (s *Project) ListNodeMappings(ctx context.Context, projectID string) ([]domain.ProjectNodeResponse, error) {
	if _, err := s.repo.Get(ctx, projectID); err != nil {
		return nil, err
	}
	if s.nodeReader == nil {
		return []domain.ProjectNodeResponse{}, nil
	}
	nodes, err := s.nodeReader.List(ctx)
	if err != nil {
		return nil, err
	}
	mappings, err := s.repo.ListNodeMappings(ctx, projectID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.ProjectNodeResponse, len(nodes))
	for i, n := range nodes {
		var p *string
		if path, ok := mappings[n.ID]; ok {
			p = path
		}
		out[i] = domain.ProjectNodeResponse{
			NodeID:     n.ID,
			NodeName:   n.Name,
			NodeRole:   string(n.Role),
			NodeStatus: string(n.Status),
			Path:       p,
		}
	}
	return out, nil
}

// SetNodePath upserts the path mapping for a project and node.
// A null or empty path clears the mapping to NULL.
// A non-empty path must start with "/".
func (s *Project) SetNodePath(ctx context.Context, projectID, nodeID string, path *string) (domain.ProjectNodeResponse, error) {
	if _, err := s.repo.Get(ctx, projectID); err != nil {
		return domain.ProjectNodeResponse{}, err
	}
	if s.nodeReader == nil {
		return domain.ProjectNodeResponse{}, domain.ErrNodeNotFound
	}
	n, err := s.nodeReader.Get(ctx, nodeID)
	if err != nil {
		if errors.Is(err, node.ErrNodeNotFound) {
			return domain.ProjectNodeResponse{}, domain.ErrNodeNotFound
		}
		return domain.ProjectNodeResponse{}, err
	}

	var normalizedPath *string
	if path != nil && *path != "" {
		if !strings.HasPrefix(*path, "/") {
			return domain.ProjectNodeResponse{}, domain.ErrInvalidPath
		}
		normalizedPath = path
	}

	if err := s.repo.SetNodePath(ctx, projectID, nodeID, normalizedPath); err != nil {
		return domain.ProjectNodeResponse{}, err
	}

	return domain.ProjectNodeResponse{
		NodeID:     n.ID,
		NodeName:   n.Name,
		NodeRole:   string(n.Role),
		NodeStatus: string(n.Status),
		Path:       normalizedPath,
	}, nil
}
