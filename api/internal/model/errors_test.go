package model

import (
	"errors"
	"fmt"
	"testing"
)

func TestErrors_Comparison(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		target   error
		expected bool
	}{
		{
			name:     "UserNotFound exact match",
			err:      ErrUserNotFound,
			target:   ErrUserNotFound,
			expected: true,
		},
		{
			name:     "UserNotFound wrapped match",
			err:      fmt.Errorf("wrapped: %w", ErrUserNotFound),
			target:   ErrUserNotFound,
			expected: true,
		},
		{
			name:     "SessionNotFound mismatch",
			err:      ErrSessionNotFound,
			target:   ErrUserNotFound,
			expected: false,
		},
		{
			name:     "InvalidToken exact match",
			err:      ErrInvalidToken,
			target:   ErrInvalidToken,
			expected: true,
		},
		{
			name:     "TokenExpired exact match",
			err:      ErrTokenExpired,
			target:   ErrTokenExpired,
			expected: true,
		},
		{
			name:     "TokenRevoked exact match",
			err:      ErrTokenRevoked,
			target:   ErrTokenRevoked,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := errors.Is(tt.err, tt.target); got != tt.expected {
				t.Errorf("errors.Is() = %v, want %v", got, tt.expected)
			}
		})
	}
}
