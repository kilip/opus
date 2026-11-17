// Package repo provides session storage backed by Ent.
package repo

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/kilip/opus/ent"
	"github.com/kilip/opus/ent/account"
	"github.com/kilip/opus/ent/authsession"
	"github.com/kilip/opus/ent/user"
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
	// CountUsers returns the number of user records in the database.
	CountUsers(ctx context.Context) (int, error)
	// CreateUserAndSession atomically creates an owner user and a session in one transaction.
	CreateUserAndSession(ctx context.Context, email, tokenHash string, expiresAt time.Time) (*domain.Session, error)
	// FindAccount looks up a linked account by provider and provider user id.
	FindAccount(ctx context.Context, provider, providerUserID string) (*domain.Account, error)
	// CreateAccount links an external provider account to an existing user.
	CreateAccount(ctx context.Context, userID uuid.UUID, provider, providerUserID, email string) (*domain.Account, error)
	// ListAccountsByUserID returns all accounts linked to userID ordered by creation time.
	ListAccountsByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Account, error)
	// DeleteAccount removes the account with the given ID belonging to userID.
	// Returns domain.ErrAccountNotFound if the account does not exist or belongs to another user.
	DeleteAccount(ctx context.Context, userID, accountID uuid.UUID) error
	// GetOwner returns the owner user, or nil if no users exist.
	GetOwner(ctx context.Context) (*domain.User, error)
	// RecordAuditLog writes an audit log entry.
	RecordAuditLog(ctx context.Context, userID *uuid.UUID, event string, meta domain.AuditMeta) error
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
	s, err := r.db.AuthSession.Create().
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
	s, err := r.db.AuthSession.Query().
		Where(authsession.TokenHashEQ(hash)).
		WithUser().
		Only(ctx)
	if err != nil {
		return nil, err
	}
	return toDomain(s, s.Edges.User), nil
}

func (r *EntRepo) DeleteByTokenHash(ctx context.Context, hash string) error {
	_, err := r.db.AuthSession.Delete().
		Where(authsession.TokenHashEQ(hash)).
		Exec(ctx)
	return err
}

func (r *EntRepo) CountUsers(ctx context.Context) (int, error) {
	return r.db.User.Query().Count(ctx)
}

func (r *EntRepo) CreateUserAndSession(ctx context.Context, email, tokenHash string, expiresAt time.Time) (*domain.Session, error) {
	tx, err := r.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	u, err := tx.User.Create().SetEmail(email).Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	s, err := tx.AuthSession.Create().SetUserID(u.ID).SetTokenHash(tokenHash).SetExpiresAt(expiresAt).Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return toDomain(s, u), nil
}

func (r *EntRepo) FindAccount(ctx context.Context, providerName, providerUserID string) (*domain.Account, error) {
	acc, err := r.db.Account.Query().
		Where(account.ProviderEQ(providerName), account.ProviderUserIDEQ(providerUserID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return toDomainAccount(acc), nil
}

func (r *EntRepo) CreateAccount(ctx context.Context, userID uuid.UUID, providerName, providerUserID, email string) (*domain.Account, error) {
	acc, err := r.db.Account.Create().
		SetUserID(userID).
		SetProvider(providerName).
		SetProviderUserID(providerUserID).
		SetEmail(email).
		SetEmailVerified(true).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return toDomainAccount(acc), nil
}

func (r *EntRepo) ListAccountsByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Account, error) {
	accs, err := r.db.Account.Query().
		Where(account.UserIDEQ(userID)).
		Order(ent.Asc(account.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*domain.Account, len(accs))
	for i, a := range accs {
		result[i] = toDomainAccount(a)
	}
	return result, nil
}

func (r *EntRepo) DeleteAccount(ctx context.Context, userID, accountID uuid.UUID) error {
	n, err := r.db.Account.Delete().
		Where(account.IDEQ(accountID), account.UserIDEQ(userID)).
		Exec(ctx)
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrAccountNotFound
	}
	return nil
}

func (r *EntRepo) GetOwner(ctx context.Context) (*domain.User, error) {
	u, err := r.db.User.Query().Order(ent.Asc(user.FieldCreatedAt)).First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &domain.User{ID: u.ID, Email: u.Email}, nil
}

func (r *EntRepo) RecordAuditLog(ctx context.Context, userID *uuid.UUID, event string, meta domain.AuditMeta) error {
	create := r.db.AuditLog.Create().
		SetEvent(event).
		SetIP(meta.IP).
		SetUserAgent(meta.UserAgent)
	if userID != nil && *userID != uuid.Nil {
		create.SetUserID(*userID)
	}
	_, err := create.Save(ctx)
	return err
}

func toDomainAccount(a *ent.Account) *domain.Account {
	return &domain.Account{
		ID:             a.ID,
		UserID:         a.UserID,
		Provider:       a.Provider,
		ProviderUserID: a.ProviderUserID,
		Email:          a.Email,
		CreatedAt:      a.CreatedAt,
	}
}

func toDomain(s *ent.AuthSession, u *ent.User) *domain.Session {
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
