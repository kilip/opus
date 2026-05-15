package repository

import (
	"context"
	"fmt"

	"github.com/kilip/opus/api/ent"
	"github.com/kilip/opus/api/ent/session"
	"github.com/kilip/opus/api/internal/model"
)

type SessionRepository interface {
	Create(ctx context.Context, s *model.Session) (*model.Session, error)
	FindByTokenHash(ctx context.Context, hash string) (*model.Session, error)
	RevokeByID(ctx context.Context, id string) error
	RevokeAllByUserID(ctx context.Context, userID string) error
}

type sessionRepository struct {
	client *ent.Client
}

func NewSessionRepository(client *ent.Client) SessionRepository {
	return &sessionRepository{client: client}
}

func (r *sessionRepository) Create(ctx context.Context, s *model.Session) (*model.Session, error) {
	entSession, err := r.client.Session.Create().
		SetID(s.ID).
		SetTokenHash(s.TokenHash).
		SetUserID(s.UserID).
		SetExpiresAt(s.ExpiresAt).
		SetRevoked(s.Revoked).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("sessionRepository.Create: %w", err)
	}
	return r.mapToModel(entSession), nil
}

func (r *sessionRepository) FindByTokenHash(ctx context.Context, hash string) (*model.Session, error) {
	entSession, err := r.client.Session.Query().Where(session.TokenHash(hash)).Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, model.ErrSessionNotFound
		}
		return nil, fmt.Errorf("sessionRepository.FindByTokenHash: %w", err)
	}
	return r.mapToModel(entSession), nil
}

func (r *sessionRepository) RevokeByID(ctx context.Context, id string) error {
	err := r.client.Session.UpdateOneID(id).SetRevoked(true).Exec(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return model.ErrSessionNotFound
		}
		return fmt.Errorf("sessionRepository.RevokeByID: %w", err)
	}
	return nil
}

func (r *sessionRepository) RevokeAllByUserID(ctx context.Context, userID string) error {
	_, err := r.client.Session.Update().
		Where(session.UserID(userID)).
		SetRevoked(true).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("sessionRepository.RevokeAllByUserID: %w", err)
	}
	return nil
}

func (r *sessionRepository) mapToModel(s *ent.Session) *model.Session {
	return &model.Session{
		ID:        s.ID,
		TokenHash: s.TokenHash,
		UserID:    s.UserID,
		ExpiresAt: s.ExpiresAt,
		Revoked:   s.Revoked,
		CreatedAt: s.CreatedAt,
	}
}
