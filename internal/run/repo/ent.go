package repo

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/kilip/opus/ent"
	entnode "github.com/kilip/opus/ent/node"
	"github.com/kilip/opus/ent/projectnode"
	entrun "github.com/kilip/opus/ent/run"
	"github.com/kilip/opus/internal/run/domain"
)

// Ent implements run persistence backed by an Ent client.
type Ent struct {
	db *ent.Client
}

// New creates a new Ent run repository.
func New(db *ent.Client) *Ent {
	return &Ent{db: db}
}

// Create persists a new Run in the queued state.
func (r *Ent) Create(ctx context.Context, in domain.CreateInput, now time.Time) (domain.Run, error) {
	mode := in.Mode
	if mode == "" {
		mode = domain.ModeVibe
	}

	create := r.db.Run.Create().
		SetTaskID(in.TaskID).
		SetProjectID(in.ProjectID).
		SetMode(entrun.Mode(mode)).
		SetState(entrun.StateQueued).
		SetBranch(in.Branch).
		SetCreatedAt(now).
		SetUpdatedAt(now)
	if in.NodeID != nil {
		create.SetNodeID(*in.NodeID)
	}

	row, err := create.Save(ctx)
	if err != nil {
		return domain.Run{}, err
	}
	return toDomainRun(row), nil
}

// Get returns a Run by ID.
func (r *Ent) Get(ctx context.Context, id uuid.UUID) (domain.Run, error) {
	row, err := r.db.Run.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return domain.Run{}, domain.ErrRunNotFound
		}
		return domain.Run{}, err
	}
	return toDomainRun(row), nil
}

// List returns Runs matching filter ordered by creation time ascending.
func (r *Ent) List(ctx context.Context, filter domain.ListFilter) ([]domain.Run, error) {
	q := r.db.Run.Query()
	if filter.TaskID != nil {
		q = q.Where(entrun.TaskIDEQ(*filter.TaskID))
	}
	if filter.ProjectID != nil {
		q = q.Where(entrun.ProjectIDEQ(*filter.ProjectID))
	}
	if filter.State != nil {
		q = q.Where(entrun.StateEQ(entrun.State(*filter.State)))
	}

	rows, err := q.Order(ent.Asc(entrun.FieldCreatedAt), ent.Asc(entrun.FieldID)).All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Run, 0, len(rows))
	for _, row := range rows {
		out = append(out, toDomainRun(row))
	}
	return out, nil
}

// Transition validates and persists a state transition and optional field updates atomically.
func (r *Ent) Transition(
	ctx context.Context,
	id uuid.UUID,
	to domain.RunState,
	in domain.TransitionInput,
	now time.Time,
) (domain.Run, error) {
	tx, err := r.db.Tx(ctx)
	if err != nil {
		return domain.Run{}, err
	}

	row, err := tx.Run.Get(ctx, id)
	if err != nil {
		_ = tx.Rollback()
		if ent.IsNotFound(err) {
			return domain.Run{}, domain.ErrRunNotFound
		}
		return domain.Run{}, err
	}

	if err := domain.Transition(domain.RunState(row.State), to); err != nil {
		_ = tx.Rollback()
		return domain.Run{}, err
	}

	upd := tx.Run.UpdateOneID(id).
		SetState(entrun.State(to)).
		SetUpdatedAt(now)

	if (to == domain.StatePreparing || to == domain.StateRunning) && row.StartedAt == nil {
		upd.SetStartedAt(now)
	}
	if domain.IsTerminalState(to) {
		upd.SetEndedAt(now)
	}
	if to == domain.StateQueued {
		upd.ClearPausedUntil()
	}
	if in.NodeID != nil {
		upd.SetNodeID(*in.NodeID)
	}
	if in.SessionID != nil {
		upd.SetSessionID(*in.SessionID)
	}
	if in.Branch != nil {
		upd.SetBranch(*in.Branch)
	}
	if in.PRNumber != nil {
		upd.SetPrNumber(*in.PRNumber)
	}
	if len(in.Report) > 0 {
		upd.SetReport(in.Report)
	}
	if in.PausedUntil != nil {
		upd.SetPausedUntil(*in.PausedUntil)
	}
	if in.FailureReason != nil {
		upd.SetFailureReason(*in.FailureReason)
	}
	if in.IncrementAttempt {
		upd.SetStageAttempts(row.StageAttempts + 1)
	}

	updated, err := upd.Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		return domain.Run{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.Run{}, err
	}
	return toDomainRun(updated), nil
}

// SetReport persists the structured report JSON on a Run, replacing any prior report.
func (r *Ent) SetReport(ctx context.Context, id uuid.UUID, report json.RawMessage, now time.Time) (domain.Run, error) {
	row, err := r.db.Run.UpdateOneID(id).
		SetReport(report).
		SetUpdatedAt(now).
		Save(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return domain.Run{}, domain.ErrRunNotFound
		}
		return domain.Run{}, err
	}
	return toDomainRun(row), nil
}

// HasActiveRun reports whether taskID already has a non-terminal Run.
func (r *Ent) HasActiveRun(ctx context.Context, taskID uuid.UUID) (bool, error) {
	return r.db.Run.Query().
		Where(
			entrun.TaskIDEQ(taskID),
			entrun.StateIn(
				entrun.StateQueued,
				entrun.StatePreparing,
				entrun.StateRunning,
				entrun.StatePaused,
			),
		).
		Exist(ctx)
}

// ListProjectNodes returns nodes mapped to projectID with their online status and mapped path.
func (r *Ent) ListProjectNodes(ctx context.Context, projectID uuid.UUID) ([]domain.ProjectNodeCandidate, error) {
	rows, err := r.db.ProjectNode.Query().
		Where(projectnode.ProjectIDEQ(projectID)).
		WithNode().
		All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.ProjectNodeCandidate, 0, len(rows))
	for _, row := range rows {
		if row.Edges.Node == nil {
			continue
		}
		var path string
		if row.Path != nil {
			path = *row.Path
		}
		out = append(out, domain.ProjectNodeCandidate{
			NodeID:     row.Edges.Node.ID,
			Name:       row.Edges.Node.Name,
			Online:     row.Edges.Node.Status == entnode.StatusOnline,
			MappedPath: path,
		})
	}
	return out, nil
}

func toDomainRun(row *ent.Run) domain.Run {
	return domain.Run{
		ID:            row.ID,
		TaskID:        row.TaskID,
		ProjectID:     row.ProjectID,
		NodeID:        row.NodeID,
		SessionID:     row.SessionID,
		Mode:          domain.RunMode(row.Mode),
		State:         domain.RunState(row.State),
		StageAttempts: row.StageAttempts,
		Branch:        row.Branch,
		PRNumber:      row.PrNumber,
		Report:        row.Report,
		PausedUntil:   row.PausedUntil,
		FailureReason: row.FailureReason,
		CreatedAt:     row.CreatedAt,
		StartedAt:     row.StartedAt,
		EndedAt:       row.EndedAt,
		UpdatedAt:     row.UpdatedAt,
	}
}
