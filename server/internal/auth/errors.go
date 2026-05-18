package auth

import "errors"

var (
	// ErrInvalidCredentials is returned when login credentials do not match.
	ErrInvalidCredentials = errors.New("auth: invalid credentials")
	// ErrEmailAlreadyExists is returned when registering a duplicate email.
	ErrEmailAlreadyExists = errors.New("auth: email already registered")
	// ErrTokenExpired is returned when a token exceeds its expiration time.
	ErrTokenExpired = errors.New("auth: token expired")
	// ErrTokenRevoked is returned when a token has been marked as revoked in the database.
	ErrTokenRevoked = errors.New("auth: token revoked")
	// ErrTokenReplay is returned when a revoked refresh token is presented, triggering replay detection.
	ErrTokenReplay = errors.New("auth: refresh token replay detected")
	// ErrProviderNotFound is returned when an invalid OAuth provider is requested.
	ErrProviderNotFound = errors.New("auth: oauth provider not found")
	// ErrInvalidOAuthState is returned when state is mismatching or expired.
	ErrInvalidOAuthState = errors.New("auth: invalid or expired oauth state")
	// ErrSessionNotFound is returned when the session cannot be found.
	ErrSessionNotFound = errors.New("auth: session not found")
	// ErrUserNotFound is returned when the user record is missing.
	ErrUserNotFound = errors.New("auth: user not found")
	// ErrTokenNotFound is returned when the specific token is missing.
	ErrTokenNotFound = errors.New("auth: token not found")
)
