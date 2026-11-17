package domain

import "errors"

var (
	ErrRunNotFound       = errors.New("RUN_NOT_FOUND")
	ErrInvalidState      = errors.New("RUN_INVALID_STATE")
	ErrInvalidMode       = errors.New("RUN_INVALID_MODE")
	ErrInvalidTransition = errors.New("RUN_INVALID_TRANSITION")
	ErrTaskIDRequired    = errors.New("RUN_TASK_ID_REQUIRED")
	ErrProjectIDRequired = errors.New("RUN_PROJECT_ID_REQUIRED")
)
