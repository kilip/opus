package domain

import "errors"

var (
	ErrTaskNotFound      = errors.New("TASK_NOT_FOUND")
	ErrTaskTitleEmpty    = errors.New("TASK_TITLE_REQUIRED")
	ErrActivityBodyEmpty = errors.New("TASK_ACTIVITY_BODY_REQUIRED")
	ErrProjectIDRequired = errors.New("PROJECT_ID_REQUIRED")
	ErrInvalidProjectID  = errors.New("INVALID_PROJECT_ID")
	ErrProjectNotFound   = errors.New("PROJECT_NOT_FOUND")
	ErrPatchForbidden    = errors.New("TASK_PATCH_FORBIDDEN")
)
