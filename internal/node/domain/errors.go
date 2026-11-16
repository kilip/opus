package domain

import "errors"

var (
	// ErrTokenNotFound indicates the join token does not exist.
	ErrTokenNotFound = errors.New("JOIN_TOKEN_NOT_FOUND")
	// ErrTokenExpired indicates the join token has expired.
	ErrTokenExpired = errors.New("JOIN_TOKEN_EXPIRED")
	// ErrTokenAlreadyUsed indicates the join token has already been consumed.
	ErrTokenAlreadyUsed = errors.New("JOIN_TOKEN_ALREADY_USED")
	// ErrNodeNameConflict indicates the node name is already registered.
	ErrNodeNameConflict = errors.New("NODE_NAME_CONFLICT")
	// ErrInvalidNodeName indicates the node name is empty or does not meet criteria.
	ErrInvalidNodeName = errors.New("INVALID_NODE_NAME")
	// ErrRateLimited indicates too many join requests were sent.
	ErrRateLimited = errors.New("RATE_LIMITED")
)
