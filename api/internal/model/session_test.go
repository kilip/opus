package model

import (
	"testing"
	"time"
)

func TestSession_IsExpired(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt time.Time
		expected  bool
	}{
		{
			name:      "not expired",
			expiresAt: time.Now().Add(time.Hour),
			expected:  false,
		},
		{
			name:      "expired",
			expiresAt: time.Now().Add(-time.Hour),
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Session{ExpiresAt: tt.expiresAt}
			if got := s.IsExpired(); got != tt.expected {
				t.Errorf("Session.IsExpired() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestSession_IsValid(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name      string
		revoked   bool
		expiresAt time.Time
		expected  bool
	}{
		{
			name:      "valid session",
			revoked:   false,
			expiresAt: now.Add(time.Hour),
			expected:  true,
		},
		{
			name:      "revoked session",
			revoked:   true,
			expiresAt: now.Add(time.Hour),
			expected:  false,
		},
		{
			name:      "expired session",
			revoked:   false,
			expiresAt: now.Add(-time.Hour),
			expected:  false,
		},
		{
			name:      "revoked and expired session",
			revoked:   true,
			expiresAt: now.Add(-time.Hour),
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Session{
				Revoked:   tt.revoked,
				ExpiresAt: tt.expiresAt,
			}
			if got := s.IsValid(); got != tt.expected {
				t.Errorf("Session.IsValid() = %v, want %v", got, tt.expected)
			}
		})
	}
}
