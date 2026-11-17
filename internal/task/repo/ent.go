package repo

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/kilip/opus/ent"
	"github.com/kilip/opus/ent/activity"
	"github.com/kilip/opus/ent/counter"
	entproject "github.com/kilip/opus/ent/project"
	enttask "github.com/kilip/opus/ent/task"
	"github.com/kilip/opus/internal/task/domain"
)

// Ent is an Ent-backed task repository.
type Ent struct {
	client *ent.Client
}

// New creates a new Ent task repository.
func New(client *ent.Client) *Ent {
	return &Ent{client: client}
}

// List returns all tasks ordered by id ascending, optionally filtered.
func (r *Ent) List(ctx context.Context, filters ...domain.TaskFilter) ([]domain.Task, error) {
	q := r.client.Task.Query().Order(enttask.ByID())
	if len(filters) > 0 && filters[0].ProjectID != "" {
		uid, err := uuid.Parse(filters[0].ProjectID)
		if err != nil {
			return nil, domain.ErrInvalidProjectID
		}
		q = q.Where(enttask.ProjectIDEQ(uid))
	}
	tasks, err := q.All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Task, len(tasks))
	for i, t := range tasks {
		out[i] = toDomainTask(t)
	}
	return out, nil
}

// CountByProject returns the number of tasks associated with a project ID.
func (r *Ent) CountByProject(ctx context.Context, projectID string) (int, error) {
	uid, err := uuid.Parse(projectID)
	if err != nil {
		return 0, nil
	}
	return r.client.Task.Query().Where(enttask.ProjectIDEQ(uid)).Count(ctx)
}

// ProjectExists checks if a project with the given ID exists.
func (r *Ent) ProjectExists(ctx context.Context, projectID string) (bool, error) {
	uid, err := uuid.Parse(projectID)
	if err != nil {
		return false, nil
	}
	return r.client.Project.Query().Where(entproject.IDEQ(uid)).Exist(ctx)
}

// Get returns task by identifier, or domain.ErrTaskNotFound.
func (r *Ent) Get(ctx context.Context, id string) (domain.Task, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return domain.Task{}, domain.ErrTaskNotFound
	}
	t, err := r.client.Task.Get(ctx, uid)
	if err != nil {
		if ent.IsNotFound(err) {
			return domain.Task{}, domain.ErrTaskNotFound
		}
		return domain.Task{}, err
	}
	return toDomainTask(t), nil
}

// FindByExternal returns the task matching (source, externalID), or domain.ErrTaskNotFound.
func (r *Ent) FindByExternal(ctx context.Context, source domain.TaskSource, externalID string) (*domain.Task, error) {
	if source == "" || externalID == "" {
		return nil, domain.ErrTaskNotFound
	}
	t, err := r.client.Task.Query().
		Where(
			enttask.SourceEQ(string(source)),
			enttask.ExternalIDEQ(externalID),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, domain.ErrTaskNotFound
		}
		return nil, err
	}
	res := toDomainTask(t)
	return &res, nil
}

// Create persists a new task, allocating OP-<n> key (unless provided) and initial activity in the same transaction.
func (r *Ent) Create(ctx context.Context, in CreateInput) (domain.Task, error) {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return domain.Task{}, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	res, err := createInTx(ctx, tx, in)
	if err != nil {
		return domain.Task{}, err
	}
	if err = tx.Commit(); err != nil {
		return domain.Task{}, err
	}
	return res, nil
}

// Import upserts a task by (source, externalID).
// If no fields differ on an existing task, no database write is performed.
func (r *Ent) Import(ctx context.Context, in CreateInput) (ImportResult, error) {
	existing, err := r.client.Task.Query().
		Where(
			enttask.SourceEQ(string(in.Source)),
			enttask.ExternalIDEQ(in.ExternalID),
		).
		Only(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return ImportResult{}, err
	}

	if ent.IsNotFound(err) {
		tx, err := r.client.Tx(ctx)
		if err != nil {
			return ImportResult{}, err
		}
		defer func() {
			if err != nil {
				_ = tx.Rollback()
			}
		}()

		created, err := createInTx(ctx, tx, in)
		if err != nil {
			return ImportResult{}, err
		}
		if err = tx.Commit(); err != nil {
			return ImportResult{}, err
		}
		return ImportResult{
			Task:    created,
			Created: true,
			Changed: true,
		}, nil
	}

	projectUID, err := uuid.Parse(in.ProjectID)
	if err != nil {
		return ImportResult{}, domain.ErrInvalidProjectID
	}

	prevStatus := domain.TaskStatus(existing.Status)
	wantLabels := normLabels(in.Labels)
	haveLabels := normLabels(existing.Labels)

	var (
		changed       bool
		statusChanged bool
	)

	if projectUID != existing.ProjectID ||
		(in.Key != "" && in.Key != existing.Key) ||
		in.ExternalURL != existing.ExternalURL ||
		in.SourceStatus != existing.SourceStatus ||
		in.Title != existing.Title ||
		in.Description != existing.Description ||
		(in.Status != nil && *in.Status != "" && *in.Status != prevStatus) ||
		(in.Priority != nil && *in.Priority != "" && string(*in.Priority) != existing.Priority) ||
		(in.Type != nil && *in.Type != "" && string(*in.Type) != existing.Type) ||
		!slices.Equal(wantLabels, haveLabels) {
		changed = true
	}

	if !changed {
		return ImportResult{
			Task:       toDomainTask(existing),
			PrevStatus: prevStatus,
		}, nil
	}

	tx, err := r.client.Tx(ctx)
	if err != nil {
		return ImportResult{}, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	u := tx.Task.UpdateOneID(existing.ID)
	if projectUID != existing.ProjectID {
		u.SetProjectID(projectUID)
	}
	if in.Key != "" && in.Key != existing.Key {
		u.SetKey(in.Key)
	}
	if in.ExternalURL != existing.ExternalURL {
		u.SetExternalURL(in.ExternalURL)
	}
	if in.SourceStatus != existing.SourceStatus {
		u.SetSourceStatus(in.SourceStatus)
	}
	if in.Title != existing.Title {
		u.SetTitle(in.Title)
	}
	if in.Description != existing.Description {
		u.SetDescription(in.Description)
	}
	if in.Priority != nil && *in.Priority != "" && string(*in.Priority) != existing.Priority {
		u.SetPriority(string(*in.Priority))
	}
	if in.Type != nil && *in.Type != "" && string(*in.Type) != existing.Type {
		u.SetType(string(*in.Type))
	}
	if !slices.Equal(wantLabels, haveLabels) {
		u.SetLabels(wantLabels)
	}
	if in.Status != nil && *in.Status != "" && *in.Status != prevStatus {
		next := *in.Status
		u.SetStatus(enttask.Status(next))
		_, err = tx.Activity.Create().
			SetTaskID(existing.ID).
			SetKind(string(domain.ActivityKindStatus)).
			SetBody("moved stage " + string(prevStatus) + " → " + string(next)).
			Save(ctx)
		if err != nil {
			return ImportResult{}, err
		}
		statusChanged = true
	}

	updated, err := u.Save(ctx)
	if err != nil {
		return ImportResult{}, err
	}
	if err = tx.Commit(); err != nil {
		return ImportResult{}, err
	}

	return ImportResult{
		Task:          toDomainTask(updated),
		PrevStatus:    prevStatus,
		Changed:       true,
		StatusChanged: statusChanged,
	}, nil
}

func createInTx(ctx context.Context, tx *ent.Tx, in CreateInput) (domain.Task, error) {
	status := domain.StatusTodo
	if in.Status != nil && *in.Status != "" {
		status = *in.Status
	}

	priority := domain.TaskPriorityP1
	if in.Priority != nil && *in.Priority != "" {
		priority = *in.Priority
	}

	kind := domain.TaskTypeFeat
	if in.Type != nil && *in.Type != "" {
		kind = *in.Type
	}

	source := domain.TaskSourceManual
	if in.Source != "" {
		source = in.Source
	}

	labels := normLabels(in.Labels)

	projectUID, err := uuid.Parse(in.ProjectID)
	if err != nil {
		return domain.Task{}, domain.ErrInvalidProjectID
	}

	key := in.Key
	if key == "" {
		// ponytail: SQLite single-writer lock ceiling; moving to another DB needs SELECT ... FOR UPDATE or upsert
		cnt, err := tx.Counter.Query().Where(counter.NameEQ("task_key")).Only(ctx)
		var nextVal int64
		if err != nil {
			if ent.IsNotFound(err) {
				nextVal = 1
				if _, err := tx.Counter.Create().SetName("task_key").SetValue(1).Save(ctx); err != nil {
					return domain.Task{}, err
				}
			} else {
				return domain.Task{}, err
			}
		} else {
			nextVal = cnt.Value + 1
			if _, err := tx.Counter.UpdateOneID(cnt.ID).SetValue(nextVal).Save(ctx); err != nil {
				return domain.Task{}, err
			}
		}
		key = fmt.Sprintf("OP-%d", nextVal)
	}

	taskCreate := tx.Task.Create().
		SetKey(key).
		SetProjectID(projectUID).
		SetSource(string(source)).
		SetTitle(in.Title).
		SetDescription(in.Description).
		SetStatus(enttask.Status(status)).
		SetPriority(string(priority)).
		SetType(string(kind)).
		SetDueDate(in.DueDate).
		SetLabels(labels)

	if in.ExternalID != "" {
		taskCreate.SetExternalID(in.ExternalID)
	}
	if in.ExternalURL != "" {
		taskCreate.SetExternalURL(in.ExternalURL)
	}
	if in.SourceStatus != "" {
		taskCreate.SetSourceStatus(in.SourceStatus)
	}
	if in.Assignee != nil {
		taskCreate.SetAssigneeID(in.Assignee.ID)
		taskCreate.SetAssigneeName(in.Assignee.Name)
	}

	t, err := taskCreate.Save(ctx)
	if err != nil {
		return domain.Task{}, err
	}

	// initial activity; author stored as NULL
	_, err = tx.Activity.Create().
		SetTaskID(t.ID).
		SetKind(string(domain.ActivityKindStatus)).
		SetBody("created task · stage " + string(status)).
		Save(ctx)
	if err != nil {
		return domain.Task{}, err
	}

	return toDomainTask(t), nil
}

// Update modifies an existing task, appending activities on changes.
func (r *Ent) Update(ctx context.Context, id string, in UpdateInput) (UpdateResult, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return UpdateResult{}, domain.ErrTaskNotFound
	}

	tx, err := r.client.Tx(ctx)
	if err != nil {
		return UpdateResult{}, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	t, err := tx.Task.Get(ctx, uid)
	if err != nil {
		if ent.IsNotFound(err) {
			return UpdateResult{}, domain.ErrTaskNotFound
		}
		return UpdateResult{}, err
	}

	u := tx.Task.UpdateOneID(uid)
	if in.Title != nil {
		u.SetTitle(*in.Title)
	}
	if in.Description != nil {
		u.SetDescription(*in.Description)
	}
	if in.Priority != nil && *in.Priority != "" {
		u.SetPriority(string(*in.Priority))
	}
	if in.Type != nil && *in.Type != "" {
		u.SetType(string(*in.Type))
	}
	if in.DueDate != nil {
		u.SetDueDate(*in.DueDate)
	}
	if in.Labels != nil {
		u.SetLabels(*in.Labels)
	}

	if in.ClearAssignee {
		if t.AssigneeID != nil || t.AssigneeName != nil {
			u.ClearAssigneeID()
			u.ClearAssigneeName()
			_, err = tx.Activity.Create().
				SetTaskID(uid).
				SetKind(string(domain.ActivityKindAssign)).
				SetBody("unassigned").
				Save(ctx)
			if err != nil {
				return UpdateResult{}, err
			}
		}
	} else if in.Assignee != nil {
		prevName := ""
		if t.AssigneeName != nil {
			prevName = *t.AssigneeName
		}
		if in.Assignee.Name != prevName {
			u.SetAssigneeID(in.Assignee.ID)
			u.SetAssigneeName(in.Assignee.Name)
			_, err = tx.Activity.Create().
				SetTaskID(uid).
				SetKind(string(domain.ActivityKindAssign)).
				SetBody("assigned to " + in.Assignee.Name).
				Save(ctx)
			if err != nil {
				return UpdateResult{}, err
			}
		}
	}

	prevStatus := domain.TaskStatus(t.Status)
	statusChanged := false
	if in.Status != nil && *in.Status != "" && *in.Status != prevStatus {
		prev := prevStatus
		next := *in.Status
		u.SetStatus(enttask.Status(next))
		_, err = tx.Activity.Create().
			SetTaskID(uid).
			SetKind(string(domain.ActivityKindStatus)).
			SetBody("moved stage " + string(prev) + " → " + string(next)).
			Save(ctx)
		if err != nil {
			return UpdateResult{}, err
		}
		statusChanged = true
	}

	updatedTask, err := u.Save(ctx)
	if err != nil {
		return UpdateResult{}, err
	}

	if err = tx.Commit(); err != nil {
		return UpdateResult{}, err
	}

	return UpdateResult{
		Task:          toDomainTask(updatedTask),
		PrevStatus:    prevStatus,
		StatusChanged: statusChanged,
	}, nil
}

// AddLabel adds label to task id if not already present.
func (r *Ent) AddLabel(ctx context.Context, id, label string) error {
	return r.mutateLabels(ctx, id, func(cur []string) []string {
		if slices.Contains(cur, label) {
			return cur
		}
		return append(slices.Clone(cur), label)
	})
}

// RemoveLabel removes label from task id if present.
func (r *Ent) RemoveLabel(ctx context.Context, id, label string) error {
	return r.mutateLabels(ctx, id, func(cur []string) []string {
		if !slices.Contains(cur, label) {
			return cur
		}
		out := make([]string, 0, len(cur))
		for _, l := range cur {
			if l != label {
				out = append(out, l)
			}
		}
		return out
	})
}

// SetLabels replaces the labels on task id.
func (r *Ent) SetLabels(ctx context.Context, id string, labels []string) error {
	next := normLabels(labels)
	return r.mutateLabels(ctx, id, func(_ []string) []string {
		return next
	})
}

func (r *Ent) mutateLabels(ctx context.Context, id string, transform func([]string) []string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return domain.ErrTaskNotFound
	}
	t, err := r.client.Task.Get(ctx, uid)
	if err != nil {
		if ent.IsNotFound(err) {
			return domain.ErrTaskNotFound
		}
		return err
	}
	cur := normLabels(t.Labels)
	next := normLabels(transform(cur))
	if slices.Equal(cur, next) {
		return nil
	}
	return r.client.Task.UpdateOneID(uid).SetLabels(next).Exec(ctx)
}

// ListActivity returns activities for the task, or domain.ErrTaskNotFound.
func (r *Ent) ListActivity(ctx context.Context, taskID string) ([]domain.Activity, error) {
	uid, err := uuid.Parse(taskID)
	if err != nil {
		return nil, domain.ErrTaskNotFound
	}

	exists, err := r.client.Task.Query().Where(enttask.IDEQ(uid)).Exist(ctx)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, domain.ErrTaskNotFound
	}

	acts, err := r.client.Activity.Query().
		Where(activity.TaskIDEQ(uid)).
		Order(activity.ByID()).
		All(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]domain.Activity, len(acts))
	for i, a := range acts {
		out[i] = toDomainActivity(a)
	}
	return out, nil
}

// AddActivity appends a new activity to the task, updating the task's updated_at.
func (r *Ent) AddActivity(ctx context.Context, taskID string, kind domain.ActivityKind, author, body string) (domain.Activity, error) {
	uid, err := uuid.Parse(taskID)
	if err != nil {
		return domain.Activity{}, domain.ErrTaskNotFound
	}

	if kind == "" {
		kind = domain.ActivityKindComment
	}

	tx, err := r.client.Tx(ctx)
	if err != nil {
		return domain.Activity{}, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	exists, err := tx.Task.Query().Where(enttask.IDEQ(uid)).Exist(ctx)
	if err != nil {
		return domain.Activity{}, err
	}
	if !exists {
		return domain.Activity{}, domain.ErrTaskNotFound
	}

	if err := tx.Task.UpdateOneID(uid).SetUpdatedAt(time.Now()).Exec(ctx); err != nil {
		return domain.Activity{}, err
	}

	actCreate := tx.Activity.Create().
		SetTaskID(uid).
		SetKind(string(kind)).
		SetBody(body)
	if author != "" {
		actCreate.SetAuthor(author)
	}
	act, err := actCreate.Save(ctx)
	if err != nil {
		return domain.Activity{}, err
	}

	if err = tx.Commit(); err != nil {
		return domain.Activity{}, err
	}

	return toDomainActivity(act), nil
}

func normLabels(labels []string) []string {
	if labels == nil {
		return []string{}
	}
	return labels
}

func toDomainTask(t *ent.Task) domain.Task {
	res := domain.Task{
		ID:           t.ID.String(),
		Key:          t.Key,
		ProjectID:    t.ProjectID.String(),
		Source:       domain.TaskSource(t.Source),
		ExternalURL:  t.ExternalURL,
		Title:        t.Title,
		Description:  t.Description,
		Status:       domain.TaskStatus(t.Status),
		SourceStatus: t.SourceStatus,
		Priority:     domain.TaskPriority(t.Priority),
		Type:         domain.TaskType(t.Type),
		DueDate:      t.DueDate,
		Labels:       t.Labels,
		CreatedAt:    t.CreatedAt,
		UpdatedAt:    t.UpdatedAt,
	}
	if t.ExternalID != nil {
		res.ExternalID = *t.ExternalID
	}
	if t.AssigneeID != nil || t.AssigneeName != nil {
		var aID, aName string
		if t.AssigneeID != nil {
			aID = *t.AssigneeID
		}
		if t.AssigneeName != nil {
			aName = *t.AssigneeName
		}
		res.Assignee = &domain.Assignee{
			ID:   aID,
			Name: aName,
		}
	}
	if res.Labels == nil {
		res.Labels = []string{}
	}
	return res
}

func toDomainActivity(a *ent.Activity) domain.Activity {
	var author string
	if a.Author != nil {
		author = *a.Author
	}
	return domain.Activity{
		ID:        a.ID.String(),
		Kind:      domain.ActivityKind(a.Kind),
		Author:    author,
		Body:      a.Body,
		CreatedAt: a.CreatedAt,
	}
}
