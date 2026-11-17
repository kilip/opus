package repo

import (
	"context"
	"maps"
	"time"

	"github.com/google/uuid"
	"github.com/kilip/opus/ent"
	entproject "github.com/kilip/opus/ent/project"
	enttrackersettings "github.com/kilip/opus/ent/trackersettings"
	enttrackersync "github.com/kilip/opus/ent/trackersync"
	"github.com/kilip/opus/internal/project"
	"github.com/kilip/opus/internal/tracker/domain"
)

// Ent is an Ent-backed tracker repository.
type Ent struct {
	client *ent.Client
}

// New creates a new Ent tracker repository.
func New(client *ent.Client) *Ent {
	return &Ent{client: client}
}

// GetSettings returns the global tracker status-label overrides, or nil if unset.
func (r *Ent) GetSettings(ctx context.Context) (map[string]string, error) {
	row, err := r.client.TrackerSettings.Query().
		Order(enttrackersettings.ByID()).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return maps.Clone(row.StatusLabels), nil
}

// UpsertSettings replaces the singleton global tracker status-label overrides.
func (r *Ent) UpsertSettings(ctx context.Context, statusLabels map[string]string) (map[string]string, error) {
	cloned := maps.Clone(statusLabels)
	if cloned == nil {
		cloned = map[string]string{}
	}

	row, err := r.client.TrackerSettings.Query().
		Order(enttrackersettings.ByID()).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			created, createErr := r.client.TrackerSettings.Create().
				SetStatusLabels(cloned).
				Save(ctx)
			if createErr != nil {
				return nil, createErr
			}
			return maps.Clone(created.StatusLabels), nil
		}
		return nil, err
	}

	updated, err := r.client.TrackerSettings.UpdateOne(row).
		SetStatusLabels(cloned).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return maps.Clone(updated.StatusLabels), nil
}

// GetSyncByProject returns the sync state for projectID.
// Returns domain.ErrInvalidProjectID on malformed UUID and domain.ErrProjectNotFound if the project does not exist.
func (r *Ent) GetSyncByProject(ctx context.Context, projectID string) (domain.TrackerSync, error) {
	uid, err := uuid.Parse(projectID)
	if err != nil {
		return domain.TrackerSync{}, domain.ErrInvalidProjectID
	}

	exists, err := r.client.Project.Query().
		Where(entproject.IDEQ(uid)).
		Exist(ctx)
	if err != nil {
		return domain.TrackerSync{}, err
	}
	if !exists {
		return domain.TrackerSync{}, domain.ErrProjectNotFound
	}

	row, err := r.client.TrackerSync.Query().
		Where(enttrackersync.ProjectIDEQ(uid)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return domain.TrackerSync{ProjectID: projectID}, nil
		}
		return domain.TrackerSync{}, err
	}
	return toSyncDomain(row), nil
}

// ListSyncs returns every persisted TrackerSync row ordered by ID.
func (r *Ent) ListSyncs(ctx context.Context) ([]domain.TrackerSync, error) {
	rows, err := r.client.TrackerSync.Query().
		Order(enttrackersync.ByID()).
		All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.TrackerSync, len(rows))
	for i, row := range rows {
		out[i] = toSyncDomain(row)
	}
	return out, nil
}

// SaveSyncSuccess upserts a successful sync state for projectID, advancing cursor and clearing last_error.
func (r *Ent) SaveSyncSuccess(ctx context.Context, projectID, cursor string, syncedAt time.Time) (domain.TrackerSync, error) {
	uid, err := uuid.Parse(projectID)
	if err != nil {
		return domain.TrackerSync{}, domain.ErrInvalidProjectID
	}

	row, err := r.client.TrackerSync.Query().
		Where(enttrackersync.ProjectIDEQ(uid)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			created, createErr := r.client.TrackerSync.Create().
				SetProjectID(uid).
				SetCursor(cursor).
				SetLastSyncAt(syncedAt).
				SetLastError("").
				Save(ctx)
			if createErr != nil {
				return domain.TrackerSync{}, createErr
			}
			return toSyncDomain(created), nil
		}
		return domain.TrackerSync{}, err
	}

	updated, err := r.client.TrackerSync.UpdateOne(row).
		SetCursor(cursor).
		SetLastSyncAt(syncedAt).
		SetLastError("").
		Save(ctx)
	if err != nil {
		return domain.TrackerSync{}, err
	}
	return toSyncDomain(updated), nil
}

// UpsertLastError records lastError on a project's TrackerSync without changing cursor or last_sync_at.
func (r *Ent) UpsertLastError(ctx context.Context, projectID, lastError string) error {
	uid, err := uuid.Parse(projectID)
	if err != nil {
		return domain.ErrInvalidProjectID
	}

	row, err := r.client.TrackerSync.Query().
		Where(enttrackersync.ProjectIDEQ(uid)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			_, createErr := r.client.TrackerSync.Create().
				SetProjectID(uid).
				SetLastError(lastError).
				Save(ctx)
			return createErr
		}
		return err
	}

	return r.client.TrackerSync.UpdateOne(row).
		SetLastError(lastError).
		Exec(ctx)
}

// ClearLastError clears last_error on existing TrackerSync row for projectID without touching cursor or last_sync_at.
func (r *Ent) ClearLastError(ctx context.Context, projectID string) error {
	uid, err := uuid.Parse(projectID)
	if err != nil {
		return domain.ErrInvalidProjectID
	}

	_, err = r.client.TrackerSync.Update().
		Where(
			enttrackersync.ProjectIDEQ(uid),
			enttrackersync.LastErrorNEQ(""),
		).
		SetLastError("").
		Save(ctx)
	return err
}

// ListProjects returns all projects ordered by ID.
func (r *Ent) ListProjects(ctx context.Context) ([]project.Record, error) {
	rows, err := r.client.Project.Query().
		Order(entproject.ByID()).
		All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]project.Record, len(rows))
	for i, p := range rows {
		var labels map[string]string
		if len(p.TrackerLabels) > 0 {
			labels = maps.Clone(p.TrackerLabels)
		}
		out[i] = project.Record{
			ID:              p.ID.String(),
			Slug:            p.Slug,
			Name:            p.Name,
			RepoProvider:    p.RepoProvider,
			RepoRef:         p.RepoRef,
			TrackerProvider: p.TrackerProvider,
			TrackerRef:      p.TrackerRef,
			DefaultBranch:   p.DefaultBranch,
			TrackerLabels:   labels,
			CreatedAt:       p.CreatedAt,
			UpdatedAt:       p.UpdatedAt,
		}
	}
	return out, nil
}

func toSyncDomain(row *ent.TrackerSync) domain.TrackerSync {
	return domain.TrackerSync{
		ID:         row.ID.String(),
		ProjectID:  row.ProjectID.String(),
		Cursor:     row.Cursor,
		LastSyncAt: row.LastSyncAt,
		LastError:  row.LastError,
	}
}
