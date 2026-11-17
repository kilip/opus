package repo

import (
	"context"

	"github.com/google/uuid"
	"github.com/kilip/opus/ent"
	enttask "github.com/kilip/opus/ent/task"
	enttrackerpr "github.com/kilip/opus/ent/trackerpr"
	"github.com/kilip/opus/internal/tracker/domain"
)

// UpsertPRInput carries fields to create or update a TrackerPR row.
type UpsertPRInput struct {
	ProjectID     string
	Number        int
	TaskID        *string
	State         domain.PRState
	IsDraft       bool
	ReviewState   string
	ChecksSummary domain.ChecksSummary
	URL           string
}

// GetPR returns the tracked PR for (projectID, number), or nil if not found.
func (r *Ent) GetPR(ctx context.Context, projectID string, number int) (*domain.TrackerPR, error) {
	pid, err := uuid.Parse(projectID)
	if err != nil {
		return nil, domain.ErrInvalidProjectID
	}

	row, err := r.client.TrackerPR.Query().
		Where(
			enttrackerpr.ProjectIDEQ(pid),
			enttrackerpr.NumberEQ(number),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	out := toPRDomain(row)
	return &out, nil
}

// UpsertPR creates or updates a TrackerPR keyed by (project_id, number).
func (r *Ent) UpsertPR(ctx context.Context, in UpsertPRInput) (domain.TrackerPR, error) {
	pid, err := uuid.Parse(in.ProjectID)
	if err != nil {
		return domain.TrackerPR{}, domain.ErrInvalidProjectID
	}

	var tid *uuid.UUID
	if in.TaskID != nil && *in.TaskID != "" {
		parsed, parseErr := uuid.Parse(*in.TaskID)
		if parseErr != nil {
			return domain.TrackerPR{}, domain.ErrInvalidTaskID
		}
		tid = &parsed
	}

	existing, err := r.client.TrackerPR.Query().
		Where(
			enttrackerpr.ProjectIDEQ(pid),
			enttrackerpr.NumberEQ(in.Number),
		).
		Only(ctx)
	if err != nil {
		if !ent.IsNotFound(err) {
			return domain.TrackerPR{}, err
		}
		create := r.client.TrackerPR.Create().
			SetProjectID(pid).
			SetNumber(in.Number).
			SetState(enttrackerpr.State(in.State)).
			SetIsDraft(in.IsDraft).
			SetReviewState(in.ReviewState).
			SetChecksSummary(enttrackerpr.ChecksSummary(in.ChecksSummary)).
			SetURL(in.URL)
		if tid != nil {
			create.SetTaskID(*tid)
		}
		created, createErr := create.Save(ctx)
		if createErr != nil {
			return domain.TrackerPR{}, createErr
		}
		return toPRDomain(created), nil
	}

	upd := r.client.TrackerPR.UpdateOne(existing).
		SetState(enttrackerpr.State(in.State)).
		SetIsDraft(in.IsDraft).
		SetReviewState(in.ReviewState).
		SetChecksSummary(enttrackerpr.ChecksSummary(in.ChecksSummary)).
		SetURL(in.URL)
	if tid != nil {
		upd.SetTaskID(*tid)
	} else {
		upd.ClearTaskID()
	}

	updated, err := upd.Save(ctx)
	if err != nil {
		return domain.TrackerPR{}, err
	}
	return toPRDomain(updated), nil
}

// ListPRsByTask returns all tracked PRs linked to taskID ordered by PR number.
// Returns domain.ErrInvalidTaskID on malformed UUID and domain.ErrTaskNotFound if the task does not exist.
func (r *Ent) ListPRsByTask(ctx context.Context, taskID string) ([]domain.TrackerPR, error) {
	tid, err := uuid.Parse(taskID)
	if err != nil {
		return nil, domain.ErrInvalidTaskID
	}

	exists, err := r.client.Task.Query().
		Where(enttask.IDEQ(tid)).
		Exist(ctx)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, domain.ErrTaskNotFound
	}

	rows, err := r.client.TrackerPR.Query().
		Where(enttrackerpr.TaskIDEQ(tid)).
		Order(enttrackerpr.ByNumber()).
		All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.TrackerPR, len(rows))
	for i, row := range rows {
		out[i] = toPRDomain(row)
	}
	return out, nil
}

// ListPRsByProject returns all tracked PRs for projectID ordered by PR number.
func (r *Ent) ListPRsByProject(ctx context.Context, projectID string) ([]domain.TrackerPR, error) {
	pid, err := uuid.Parse(projectID)
	if err != nil {
		return nil, domain.ErrInvalidProjectID
	}

	rows, err := r.client.TrackerPR.Query().
		Where(enttrackerpr.ProjectIDEQ(pid)).
		Order(enttrackerpr.ByNumber()).
		All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.TrackerPR, len(rows))
	for i, row := range rows {
		out[i] = toPRDomain(row)
	}
	return out, nil
}

func toPRDomain(row *ent.TrackerPR) domain.TrackerPR {
	var taskID *string
	if row.TaskID != nil {
		s := row.TaskID.String()
		taskID = &s
	}
	return domain.TrackerPR{
		ID:            row.ID.String(),
		ProjectID:     row.ProjectID.String(),
		TaskID:        taskID,
		Number:        row.Number,
		State:         domain.PRState(row.State),
		IsDraft:       row.IsDraft,
		ReviewState:   row.ReviewState,
		ChecksSummary: domain.ChecksSummary(row.ChecksSummary),
		URL:           row.URL,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
	}
}
