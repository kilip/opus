package repo

import (
	"context"

	"github.com/kilip/opus/ent"
	entuser "github.com/kilip/opus/ent/user"
	"github.com/kilip/opus/internal/user/domain"
)

// Ent is an Ent-backed user repository.
type Ent struct {
	client *ent.Client
}

// New creates a new Ent repository.
func New(client *ent.Client) *Ent {
	return &Ent{client: client}
}

// Count returns the total number of users.
func (r *Ent) Count(ctx context.Context) (int, error) {
	return r.client.User.Query().Count(ctx)
}

// ByEmail finds a user by email.
func (r *Ent) ByEmail(ctx context.Context, email string) (*domain.User, error) {
	u, err := r.client.User.Query().
		Where(entuser.EmailEQ(domain.NormalizeEmail(email))).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return toDomain(u), nil
}

// Create persists a new user with the given email.
func (r *Ent) Create(ctx context.Context, email string) (*domain.User, error) {
	u, err := r.client.User.Create().
		SetEmail(domain.NormalizeEmail(email)).
		Save(ctx)
	if err != nil {
		if ent.IsConstraintError(err) {
			return nil, domain.ErrEmailTaken
		}
		return nil, err
	}
	return toDomain(u), nil
}

func toDomain(u *ent.User) *domain.User {
	if u == nil {
		return nil
	}
	return &domain.User{
		ID:        u.ID,
		Email:     u.Email,
		CreatedAt: u.CreatedAt,
	}
}
