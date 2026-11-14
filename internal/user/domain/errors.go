package domain

import "errors"

var (
	ErrNotFound   = errors.New("USER_NOT_FOUND")
	ErrEmailTaken = errors.New("USER_EMAIL_TAKEN")
)
