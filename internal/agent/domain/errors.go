package domain

import "errors"

// Stable error codes for MCP tool responses.
const (
	CodeNotFound       = "not_found"
	CodeForbiddenScope = "forbidden_scope"
	CodeTooLarge       = "too_large"
	CodeInvalidStatus  = "invalid_status"
	CodeRateLimited    = "rate_limited"
	CodeInvalidInput   = "invalid_input"
)

// Sentinel errors for the agent feature.
var (
	ErrNotFound       = errors.New("not_found")
	ErrForbiddenScope = errors.New("forbidden_scope")
	ErrTooLarge       = errors.New("too_large")
	ErrInvalidStatus  = errors.New("invalid_status")
	ErrRateLimited    = errors.New("rate_limited")
	ErrInvalidInput   = errors.New("invalid_input")
)

// Code returns the stable error code for an agent domain error, or "" if unknown.
func Code(err error) string {
	switch {
	case errors.Is(err, ErrNotFound):
		return CodeNotFound
	case errors.Is(err, ErrForbiddenScope):
		return CodeForbiddenScope
	case errors.Is(err, ErrTooLarge):
		return CodeTooLarge
	case errors.Is(err, ErrInvalidStatus):
		return CodeInvalidStatus
	case errors.Is(err, ErrRateLimited):
		return CodeRateLimited
	case errors.Is(err, ErrInvalidInput):
		return CodeInvalidInput
	default:
		return ""
	}
}
