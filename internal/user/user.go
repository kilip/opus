// Package user is the user feature. Other packages depend on this package only,
// never on its subpackages.
package user

import (
	"context"

	"github.com/kilip/opus/ent"
	"github.com/kilip/opus/internal/user/domain"
	"github.com/kilip/opus/internal/user/repo"
)

var (
	ErrNotFound   = domain.ErrNotFound
	ErrEmailTaken = domain.ErrEmailTaken
)

// User is the user feature facade.
type User struct {
	repo *repo.Ent
}

// New creates a new User feature facade.
func New(db *ent.Client) *User {
	return &User{repo: repo.New(db)}
}

// Count returns the total number of users.
func (u *User) Count(ctx context.Context) (int, error) {
	return u.repo.Count(ctx)
}

// ByEmail finds a user by email.
func (u *User) ByEmail(ctx context.Context, email string) (*domain.User, error) {
	return u.repo.ByEmail(ctx, email)
}

// Create persists a new user with the given email.
func (u *User) Create(ctx context.Context, email string) (*domain.User, error) {
	return u.repo.Create(ctx, email)
}
