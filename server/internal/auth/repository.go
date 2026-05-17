package auth

import "context"

//go:generate mockgen -destination=mock_repository.go -package=auth github.com/kilip/opus/server/internal/auth Repository

// Repository defines the interface for authentication storage operations.
type Repository interface {
	// FindByEmail retrieves a user by their email address.
	FindByEmail(ctx context.Context, email string) (*User, error)
}
