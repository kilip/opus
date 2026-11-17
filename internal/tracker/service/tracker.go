package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/kilip/opus/internal/project"
	"github.com/kilip/opus/internal/tracker/domain"
	"github.com/kilip/opus/internal/tracker/repo"
)

// Tracker owns tracker status-label and sync-state business logic.
type Tracker struct {
	repo *repo.Ent
}

// New creates a new Tracker service backed by Ent repo.
func New(r *repo.Ent) *Tracker {
	return &Tracker{repo: r}
}

// GetProjectStatus returns the sync timestamp and last error for a project.
func (s *Tracker) GetProjectStatus(ctx context.Context, projectID string) (domain.TrackerStatusResponse, error) {
	if _, err := uuid.Parse(projectID); err != nil {
		return domain.TrackerStatusResponse{}, domain.ErrInvalidProjectID
	}
	syncState, err := s.repo.GetSyncByProject(ctx, projectID)
	if err != nil {
		return domain.TrackerStatusResponse{}, err
	}
	return domain.TrackerStatusResponse{
		LastSyncAt: syncState.LastSyncAt,
		LastError:  syncState.LastError,
	}, nil
}

// GetSettings returns the global tracker status-label map with defaults filled in.
func (s *Tracker) GetSettings(ctx context.Context) (domain.TrackerSettingsResponse, error) {
	global, err := s.repo.GetSettings(ctx)
	if err != nil {
		return domain.TrackerSettingsResponse{}, err
	}
	effective := domain.ResolveEffectiveLabels(global, nil)
	return domain.TrackerSettingsResponse{
		StatusLabels: effective.StatusToLabel,
	}, nil
}

// UpdateSettings validates and replaces global status-label overrides, checking for
// collisions against defaults and all existing project-level overrides.
func (s *Tracker) UpdateSettings(ctx context.Context, req domain.UpdateTrackerSettingsRequest) (domain.TrackerSettingsResponse, error) {
	if err := domain.ValidateGlobalLabels(req.StatusLabels); err != nil {
		return domain.TrackerSettingsResponse{}, err
	}

	projects, err := s.repo.ListProjects(ctx)
	if err != nil {
		return domain.TrackerSettingsResponse{}, err
	}
	for _, p := range projects {
		if len(p.TrackerLabels) == 0 {
			continue
		}
		if err := domain.ValidateProjectLabels(req.StatusLabels, p.TrackerLabels); err != nil {
			return domain.TrackerSettingsResponse{}, err
		}
	}

	saved, err := s.repo.UpsertSettings(ctx, req.StatusLabels)
	if err != nil {
		return domain.TrackerSettingsResponse{}, err
	}
	effective := domain.ResolveEffectiveLabels(saved, nil)
	return domain.TrackerSettingsResponse{
		StatusLabels: effective.StatusToLabel,
	}, nil
}

// ValidateProjectLabels validates project-level status-label overrides against global settings and defaults.
func (s *Tracker) ValidateProjectLabels(ctx context.Context, projectLabels map[string]string) error {
	global, err := s.repo.GetSettings(ctx)
	if err != nil {
		return err
	}
	return domain.ValidateProjectLabels(global, projectLabels)
}

// EffectiveLabels returns the resolved bidirectional status-to-label map for a project.
func (s *Tracker) EffectiveLabels(ctx context.Context, proj project.Record) (domain.EffectiveLabelMap, error) {
	global, err := s.repo.GetSettings(ctx)
	if err != nil {
		return domain.EffectiveLabelMap{}, err
	}
	return domain.ResolveEffectiveLabels(global, proj.TrackerLabels), nil
}

// ListTaskPRs returns tracked pull request snapshots linked to taskID.
func (s *Tracker) ListTaskPRs(ctx context.Context, taskID string) ([]domain.TrackerPR, error) {
	if _, err := uuid.Parse(taskID); err != nil {
		return nil, domain.ErrInvalidTaskID
	}
	return s.repo.ListPRsByTask(ctx, taskID)
}
