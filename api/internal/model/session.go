package model

import "time"

type Session struct {
	ID        string
	TokenHash string
	UserID    string
	ExpiresAt time.Time
	Revoked   bool
	CreatedAt time.Time
}
