package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// ErrUnauthorized indicates missing, invalid, or expired authentication.
var ErrUnauthorized = errors.New("UNAUTHORIZED")

// User represents the leaf user entity within auth domain.
type User struct {
	ID    uuid.UUID
	Email string
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
