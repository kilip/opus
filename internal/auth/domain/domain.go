package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// ErrUnauthorized indicates missing, invalid, or expired authentication.
var ErrUnauthorized = errors.New("UNAUTHORIZED")

// ErrSetupClosed indicates initial owner setup is already complete.
var ErrSetupClosed = errors.New("SETUP_CLOSED")

// ErrProviderNotFound indicates an unknown or unconfigured OAuth provider.
var ErrProviderNotFound = errors.New("PROVIDER_NOT_FOUND")

// ErrAccountNotAllowed indicates the OAuth account is not linked or permitted to log in.
var ErrAccountNotAllowed = errors.New("ACCOUNT_NOT_ALLOWED")

// ErrSetupRequired indicates no owner exists yet so OAuth login cannot proceed.
var ErrSetupRequired = errors.New("SETUP_REQUIRED")

// ErrStateMismatch indicates an invalid or mismatched OAuth state parameter.
var ErrStateMismatch = errors.New("STATE_MISMATCH")

// User represents the leaf user entity within auth domain.
type User struct {
	ID    uuid.UUID
	Email string
}

// Account represents a linked external OAuth provider identity.
type Account struct {
	ID             uuid.UUID
	UserID         uuid.UUID
	Provider       string
	ProviderUserID string
	Email          string
}

// Session represents an authenticated session.
type Session struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string
	ExpiresAt time.Time
	CreatedAt time.Time
	User      *User
}
