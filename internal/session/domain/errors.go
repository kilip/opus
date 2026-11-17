package domain

import "errors"

var (
	ErrSessionNotFound  = errors.New("SESSION_NOT_FOUND")
	ErrInvalidStatus    = errors.New("SESSION_INVALID_STATUS")
	ErrProviderRequired = errors.New("SESSION_PROVIDER_REQUIRED")
	ErrInvalidSessionID = errors.New("INVALID_SESSION_ID")
	ErrEventKindEmpty   = errors.New("SESSION_EVENT_KIND_REQUIRED")
)
