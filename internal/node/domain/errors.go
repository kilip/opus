package domain

import "errors"

var (
	// ErrTokenNotFound indicates the join token does not exist.
	ErrTokenNotFound = errors.New("JOIN_TOKEN_NOT_FOUND")
	// ErrTokenExpired indicates the join token has expired.
	ErrTokenExpired = errors.New("JOIN_TOKEN_EXPIRED")
	// ErrTokenAlreadyUsed indicates the join token has already been consumed.
	ErrTokenAlreadyUsed = errors.New("JOIN_TOKEN_ALREADY_USED")
)
