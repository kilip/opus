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
	// ErrInvalidTaskID indicates the task ID path parameter is malformed.
	ErrInvalidTaskID = errors.New("INVALID_TASK_ID")
	// ErrTaskNotFound indicates the target task does not exist.
	ErrTaskNotFound = errors.New("TASK_NOT_FOUND")
)
