package domain

import "errors"

var (
	// ErrRateLimited indicates the upstream tracker CLI returned a rate-limit error.
	ErrRateLimited = errors.New("TRACKER_RATE_LIMITED")
	// ErrInvalidLabelConfig indicates a status-label mapping failed validation.
	ErrInvalidLabelConfig = errors.New("TRACKER_INVALID_LABEL_CONFIG")
	// ErrProjectNotFound indicates the target project does not exist.
	ErrProjectNotFound = errors.New("PROJECT_NOT_FOUND")
	// ErrInvalidProjectID indicates the project ID path parameter is malformed.
	ErrInvalidProjectID = errors.New("INVALID_PROJECT_ID")
)
