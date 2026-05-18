package entgo

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kilip/opus/server/ent"
	"github.com/kilip/opus/server/ent/authaccount"
	"github.com/kilip/opus/server/ent/authoauthstate"
	"github.com/kilip/opus/server/ent/authtoken"
	"github.com/kilip/opus/server/ent/user"
	"github.com/kilip/opus/server/internal/auth"
)

// AuthRepo mounts Ent Go clients to provide persistent query layers.
type AuthRepo struct {
	client *ent.Client
}

// NewAuthRepo maps standard database query functions.
func NewAuthRepo(client *ent.Client) *AuthRepo {
	return &AuthRepo{client: client}
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// CreateUserWithWorkspace handles relational record construction inside atomic transactions.
func (r *AuthRepo) CreateUserWithWorkspace(ctx context.Context, u *auth.User, a *auth.Account, workspaceName string) (*auth.User, error) {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("auth.repo.CreateUserWithWorkspace: %w", err)
	}

	w, err := tx.Workspace.Create().
		SetID(u.WorkspaceID).
		SetName(workspaceName).
		Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		return nil, fmt.Errorf("auth.repo.CreateUserWithWorkspace: %w", err)
	}

	create := tx.User.Create().
		SetID(u.ID).
		SetName(u.Name).
		SetEmail(u.Email).
		SetAvatarURL(u.AvatarURL).
		SetProvider(u.Provider).
		SetProviderID(u.ProviderID).
		SetWorkspaceID(w.ID)

	if u.PasswordHash != "" {
		create.SetPasswordHash(u.PasswordHash)
	}

	createdUser, err := create.Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		return nil, fmt.Errorf("auth.repo.CreateUserWithWorkspace: %w", err)
	}

	if a != nil {
		_, err = tx.AuthAccount.Create().
			SetID(a.ID).
			SetUserID(createdUser.ID).
			SetAccountID(a.AccountID).
			SetProviderID(a.ProviderID).
			Save(ctx)
		if err != nil {
			_ = tx.Rollback()
			return nil, fmt.Errorf("auth.repo.CreateUserWithWorkspace: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("auth.repo.CreateUserWithWorkspace: %w", err)
	}

	return &auth.User{
		ID:           createdUser.ID,
		Name:         createdUser.Name,
		Email:        createdUser.Email,
		AvatarURL:    createdUser.AvatarURL,
		Provider:     createdUser.Provider,
		ProviderID:   createdUser.ProviderID,
		WorkspaceID:  createdUser.WorkspaceID,
		PasswordHash: derefString(createdUser.PasswordHash),
		CreatedAt:    createdUser.CreatedAt,
		UpdatedAt:    createdUser.UpdatedAt,
	}, nil
}

// FindUserByEmail fetches a user record matching the given email.
func (r *AuthRepo) FindUserByEmail(ctx context.Context, email string) (*auth.User, error) {
	u, err := r.client.User.Query().Where(user.EmailEQ(email)).Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, auth.ErrUserNotFound
		}
		return nil, fmt.Errorf("auth.repo.FindUserByEmail: %w", err)
	}
	return &auth.User{
		ID:           u.ID,
		Email:        u.Email,
		Name:         u.Name,
		AvatarURL:    u.AvatarURL,
		Provider:     u.Provider,
		ProviderID:   u.ProviderID,
		WorkspaceID:  u.WorkspaceID,
		PasswordHash: derefString(u.PasswordHash),
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    u.UpdatedAt,
	}, nil
}

// FindUserByID resolves database columns to local domain schemas.
func (r *AuthRepo) FindUserByID(ctx context.Context, id string) (*auth.User, error) {
	u, err := r.client.User.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, auth.ErrUserNotFound
		}
		return nil, fmt.Errorf("auth.repo.FindUserByID: %w", err)
	}
	return &auth.User{
		ID:           u.ID,
		Email:        u.Email,
		Name:         u.Name,
		AvatarURL:    u.AvatarURL,
		Provider:     u.Provider,
		ProviderID:   u.ProviderID,
		WorkspaceID:  u.WorkspaceID,
		PasswordHash: derefString(u.PasswordHash),
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    u.UpdatedAt,
	}, nil
}

// FindAccountByProvider locates user credentials mapped to target OAuth targets.
func (r *AuthRepo) FindAccountByProvider(ctx context.Context, providerID, accountID string) (*auth.Account, *auth.User, error) {
	acc, err := r.client.AuthAccount.Query().
		Where(authaccount.ProviderIDEQ(providerID), authaccount.AccountIDEQ(accountID)).
		WithUser().
		Only(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("auth.repo.FindAccountByProvider: %w", err)
	}

	u := acc.Edges.User
	domainUser := &auth.User{
		ID:           u.ID,
		Email:        u.Email,
		Name:         u.Name,
		AvatarURL:    u.AvatarURL,
		Provider:     u.Provider,
		ProviderID:   u.ProviderID,
		WorkspaceID:  u.WorkspaceID,
		PasswordHash: derefString(u.PasswordHash),
	}

	domainAccount := &auth.Account{
		ID:         acc.ID,
		UserID:     acc.UserID,
		AccountID:  acc.AccountID,
		ProviderID: acc.ProviderID,
	}

	return domainAccount, domainUser, nil
}

// FindAccountByUserIDAndProvider retrieves standard credential bindings.
func (r *AuthRepo) FindAccountByUserIDAndProvider(ctx context.Context, userID, providerID string) (*auth.Account, error) {
	acc, err := r.client.AuthAccount.Query().
		Where(authaccount.UserID(userID), authaccount.ProviderID(providerID)).
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("auth.repo.FindAccountByUserIDAndProvider: %w", err)
	}
	return &auth.Account{
		ID:         acc.ID,
		UserID:     acc.UserID,
		AccountID:  acc.AccountID,
		ProviderID: acc.ProviderID,
	}, nil
}

// LinkAccount maps federated credentials to existing emails.
func (r *AuthRepo) LinkAccount(ctx context.Context, a *auth.Account) error {
	_, err := r.client.AuthAccount.Create().
		SetID(a.ID).
		SetUserID(a.UserID).
		SetAccountID(a.AccountID).
		SetProviderID(a.ProviderID).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("auth.repo.LinkAccount: %w", err)
	}
	return nil
}

// CreateSession establishes session mappings.
func (r *AuthRepo) CreateSession(ctx context.Context, s *auth.Session) error {
	_, err := r.client.AuthSession.Create().
		SetID(s.ID).
		SetUserID(s.UserID).
		SetWorkspaceID(s.WorkspaceID).
		SetIPAddress(s.IPAddress).
		SetUserAgent(s.UserAgent).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("auth.repo.CreateSession: %w", err)
	}
	return nil
}

// FindSessionByID resolves session objects.
func (r *AuthRepo) FindSessionByID(ctx context.Context, sessionID string) (*auth.Session, error) {
	s, err := r.client.AuthSession.Get(ctx, sessionID)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, auth.ErrSessionNotFound
		}
		return nil, fmt.Errorf("auth.repo.FindSessionByID: %w", err)
	}
	return &auth.Session{
		ID:          s.ID,
		UserID:      s.UserID,
		WorkspaceID: s.WorkspaceID,
		IPAddress:   s.IPAddress,
		UserAgent:   s.UserAgent,
		CreatedAt:   s.CreatedAt,
	}, nil
}

// CreateToken persists stateful token record hashes.
func (r *AuthRepo) CreateToken(ctx context.Context, t *auth.Token) error {
	_, err := r.client.AuthToken.Create().
		SetID(t.ID).
		SetSessionID(t.SessionID).
		SetUserID(t.UserID).
		SetType(t.Type).
		SetHash(t.Hash).
		SetExpiresAt(t.ExpiresAt).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("auth.repo.CreateToken: %w", err)
	}
	return nil
}

// FindTokenByHash retrieves token records for validation.
func (r *AuthRepo) FindTokenByHash(ctx context.Context, hash string) (*auth.Token, error) {
	t, err := r.client.AuthToken.Query().Where(authtoken.HashEQ(hash)).Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, auth.ErrTokenNotFound
		}
		return nil, fmt.Errorf("auth.repo.FindTokenByHash: %w", err)
	}

	return &auth.Token{
		ID:        t.ID,
		SessionID: t.SessionID,
		UserID:    t.UserID,
		Type:      t.Type,
		Hash:      t.Hash,
		ExpiresAt: t.ExpiresAt,
		RevokedAt: t.RevokedAt,
		CreatedAt: t.CreatedAt,
	}, nil
}

// RevokeToken marks single items invalid.
func (r *AuthRepo) RevokeToken(ctx context.Context, tokenID string) error {
	_, err := r.client.AuthToken.UpdateOneID(tokenID).SetRevokedAt(time.Now()).Save(ctx)
	if err != nil {
		return fmt.Errorf("auth.repo.RevokeToken: %w", err)
	}
	return nil
}

// RevokeSessionTokens invalidates all associated signatures.
func (r *AuthRepo) RevokeSessionTokens(ctx context.Context, sessionID string) error {
	_, err := r.client.AuthToken.Update().
		Where(authtoken.SessionID(sessionID)).
		SetRevokedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("auth.repo.RevokeSessionTokens: %w", err)
	}
	return nil
}

// CreateOAuthState registers state signatures.
func (r *AuthRepo) CreateOAuthState(ctx context.Context, state, providerID string, expiresAt time.Time) error {
	u, err := uuid.NewV7()
	if err != nil {
		return fmt.Errorf("auth.repo.CreateOAuthState: failed to generate uuid: %w", err)
	}
	_, err = r.client.AuthOauthState.Create().
		SetID(u.String()).
		SetState(state).
		SetProviderID(providerID).
		SetExpiresAt(expiresAt).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("auth.repo.CreateOAuthState: %w", err)
	}
	return nil
}

// ValidateOAuthState verifies single-use state signatures and deletes records.
func (r *AuthRepo) ValidateOAuthState(ctx context.Context, state string) (string, error) {
	record, err := r.client.AuthOauthState.Query().Where(authoauthstate.StateEQ(state)).Only(ctx)
	if err != nil {
		return "", auth.ErrInvalidOAuthState
	}

	providerID := record.ProviderID
	if time.Now().After(record.ExpiresAt) {
		_ = r.client.AuthOauthState.DeleteOne(record).Exec(ctx)
		return "", auth.ErrInvalidOAuthState
	}

	_ = r.client.AuthOauthState.DeleteOne(record).Exec(ctx)
	return providerID, nil
}
