package domain

import "errors"

var (
	ErrTaskNotFound      = errors.New("TASK_NOT_FOUND")
	ErrTaskTitleEmpty    = errors.New("TASK_TITLE_REQUIRED")
	ErrActivityBodyEmpty = errors.New("TASK_ACTIVITY_BODY_REQUIRED")
)
