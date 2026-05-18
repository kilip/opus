package auth

import (
	"context"
	"time"
)

//go:generate mockgen -destination=mock_repository.go -package=auth github.com/kilip/opus/server/internal/auth Repository,PolicyService,OAuthProvider

// Repository handles abstract operations on Ent database drivers.
type Repository interface {
	// CreateUserWithWorkspace provisions a user and default workspace as a transactional unit.
	CreateUserWithWorkspace(ctx context.Context, user *User, account *Account, workspaceName string) (*User, error)

	// FindUserByEmail retrieves a user domain model by their email address.
	FindUserByEmail(ctx context.Context, email string) (*User, error)

	// FindUserByID retrieves a user domain model by their unique identifier.
	FindUserByID(ctx context.Context, id string) (*User, error)

	// FindAccountByProvider retrieves link account and associated user domain models by provider details.
	FindAccountByProvider(ctx context.Context, providerID, accountID string) (*Account, *User, error)

	// FindAccountByUserIDAndProvider retrieves a specific link account by user and provider.
	FindAccountByUserIDAndProvider(ctx context.Context, userID, providerID string) (*Account, error)

	// LinkAccount creates a new third-party identity linkage.
	LinkAccount(ctx context.Context, account *Account) error

	// CreateSession provisions a stateful session.
	CreateSession(ctx context.Context, session *Session) error

	// FindSessionByID retrieves a stateful session by its ID.
	FindSessionByID(ctx context.Context, sessionID string) (*Session, error)

	// CreateToken registers a stateful access or refresh token hash.
	CreateToken(ctx context.Context, token *Token) error

	// FindTokenByHash retrieves a token by its unique SHA-256 hash representation.
	FindTokenByHash(ctx context.Context, hash string) (*Token, error)

	// RevokeToken marks a token as revoked in the database.
	RevokeToken(ctx context.Context, tokenID string) error

	// RevokeSessionTokens marks all tokens of a session as revoked in the database.
	RevokeSessionTokens(ctx context.Context, sessionID string) error

	// CreateOAuthState registers a single-use CSRF OAuth state.
	CreateOAuthState(ctx context.Context, state, providerID string, expiresAt time.Time) error

	// ValidateOAuthState verifies and consumes an active OAuth state.
	ValidateOAuthState(ctx context.Context, state string) (string, error)
}

// PolicyService handles workspace roles using Casbin rule definitions.
type PolicyService interface {
	// Enforce evaluates role policies for a subject, domain, object, and action.
	Enforce(sub, dom, obj, act string) (bool, error)

	// AssignRole grants a workspace-level role to a user.
	AssignRole(ctx context.Context, user, domain, role string) (bool, error)

	// RevokeRole removes a workspace-level role from a user.
	RevokeRole(ctx context.Context, user, domain, role string) (bool, error)
}
