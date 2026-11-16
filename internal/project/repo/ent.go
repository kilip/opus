package repo

import (
	"context"

	"github.com/google/uuid"
	"github.com/kilip/opus/ent"
	entproject "github.com/kilip/opus/ent/project"
	"github.com/kilip/opus/internal/project/domain"
)

// Ent is an Ent-backed project repository.
type Ent struct {
	client *ent.Client
}

// New creates a new Ent project repository.
func New(client *ent.Client) *Ent {
	return &Ent{client: client}
}

func toDomain(p *ent.Project) domain.Project {
	if p == nil {
		return domain.Project{}
	}
	return domain.Project{
		ID:              p.ID.String(),
		Slug:            p.Slug,
		Name:            p.Name,
		RepoProvider:    p.RepoProvider,
		RepoRef:         p.RepoRef,
		TrackerProvider: p.TrackerProvider,
		TrackerRef:      p.TrackerRef,
		DefaultBranch:   p.DefaultBranch,
		CreatedAt:       p.CreatedAt,
		UpdatedAt:       p.UpdatedAt,
	}
}

// List returns all projects ordered by id ascending.
func (r *Ent) List(ctx context.Context) ([]domain.Project, error) {
	projects, err := r.client.Project.Query().
		Order(entproject.ByID()).
		All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Project, len(projects))
	for i, p := range projects {
		out[i] = toDomain(p)
	}
	return out, nil
}

// Get returns a single project by id, or domain.ErrProjectNotFound.
func (r *Ent) Get(ctx context.Context, id string) (domain.Project, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return domain.Project{}, domain.ErrProjectNotFound
	}
	p, err := r.client.Project.Get(ctx, uid)
	if err != nil {
		if ent.IsNotFound(err) {
			return domain.Project{}, domain.ErrProjectNotFound
		}
		return domain.Project{}, err
	}
	return toDomain(p), nil
}

// Create persists a new project.
func (r *Ent) Create(ctx context.Context, in domain.Project) (domain.Project, error) {
	p, err := r.client.Project.Create().
		SetSlug(in.Slug).
		SetName(in.Name).
		SetRepoProvider(in.RepoProvider).
		SetRepoRef(in.RepoRef).
		SetTrackerProvider(in.TrackerProvider).
		SetTrackerRef(in.TrackerRef).
		SetDefaultBranch(in.DefaultBranch).
		Save(ctx)
	if err != nil {
		if ent.IsConstraintError(err) {
			return domain.Project{}, domain.ErrDuplicateSlug
		}
		return domain.Project{}, err
	}
	return toDomain(p), nil
}

// Update modifies fields on an existing project.
func (r *Ent) Update(ctx context.Context, id string, in domain.UpdateProjectRequest) (domain.Project, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return domain.Project{}, domain.ErrProjectNotFound
	}
	u := r.client.Project.UpdateOneID(uid)
	if in.Name != nil {
		u.SetName(*in.Name)
	}
	if in.DefaultBranch != nil {
		u.SetDefaultBranch(*in.DefaultBranch)
	}
	if in.TrackerProvider != nil {
		u.SetTrackerProvider(*in.TrackerProvider)
	}
	if in.TrackerRef != nil {
		u.SetTrackerRef(*in.TrackerRef)
	}
	p, err := u.Save(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return domain.Project{}, domain.ErrProjectNotFound
		}
		if ent.IsConstraintError(err) {
			return domain.Project{}, domain.ErrDuplicateSlug
		}
		return domain.Project{}, err
	}
	return toDomain(p), nil
}

// Delete removes a project by id.
func (r *Ent) Delete(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return domain.ErrProjectNotFound
	}
	err = r.client.Project.DeleteOneID(uid).Exec(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return domain.ErrProjectNotFound
		}
		return err
	}
	return nil
}
