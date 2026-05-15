package model

import "errors"

var (
	ErrUserNotFound    = errors.New("user not found")
	ErrSessionNotFound = errors.New("session not found")
	ErrInvalidToken    = errors.New("invalid token")
	ErrTokenExpired    = errors.New("token expired")
	ErrTokenRevoked    = errors.New("token revoked")
)
