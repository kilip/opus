// Package repo provides session storage backed by Ent.
package repo

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/kilip/opus/ent"
	"github.com/kilip/opus/ent/session"
	"github.com/kilip/opus/internal/auth/domain"
)

// Repo is the session repository interface.
type Repo interface {
	// CreateSession persists a new session and returns it with User populated.
	CreateSession(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) (*domain.Session, error)
	// FindByTokenHash looks up a session by its HMAC hash; loads User edge.
	FindByTokenHash(ctx context.Context, hash string) (*domain.Session, error)
	// DeleteByTokenHash removes the session with the given hash (noop if absent).
	DeleteByTokenHash(ctx context.Context, hash string) error
}

// EntRepo implements Repo using an Ent client.
type EntRepo struct {
	db *ent.Client
}

// New returns an EntRepo backed by db.
func New(db *ent.Client) *EntRepo {
	return &EntRepo{db: db}
}

func (r *EntRepo) CreateSession(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) (*domain.Session, error) {
	s, err := r.db.Session.Create().
		SetUserID(userID).
		SetTokenHash(tokenHash).
		SetExpiresAt(expiresAt).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	u, err := r.db.User.Get(ctx, userID)
	if err != nil {
		return nil, err
	}
	return toDomain(s, u), nil
}

func (r *EntRepo) FindByTokenHash(ctx context.Context, hash string) (*domain.Session, error) {
	s, err := r.db.Session.Query().
		Where(session.TokenHashEQ(hash)).
		WithUser().
		Only(ctx)
	if err != nil {
		return nil, err
	}
	return toDomain(s, s.Edges.User), nil
}

func (r *EntRepo) DeleteByTokenHash(ctx context.Context, hash string) error {
	_, err := r.db.Session.Delete().
		Where(session.TokenHashEQ(hash)).
		Exec(ctx)
	return err
}

func toDomain(s *ent.Session, u *ent.User) *domain.Session {
	ds := &domain.Session{
		ID:        s.ID,
		UserID:    s.UserID,
		TokenHash: s.TokenHash,
		ExpiresAt: s.ExpiresAt,
		CreatedAt: s.CreatedAt,
	}
	if u != nil {
		ds.User = &domain.User{ID: u.ID, Email: u.Email}
	}
	return ds
}
