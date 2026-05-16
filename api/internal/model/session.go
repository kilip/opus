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

// IsExpired returns true if the session has expired
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// IsValid returns true if the session is not expired and not revoked
func (s *Session) IsValid() bool {
	return !s.Revoked && !s.IsExpired()
}
