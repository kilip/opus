package model

import (
	"testing"
)

func TestUser_IsSocialLogin(t *testing.T) {
	tests := []struct {
		name     string
		user     User
		expected bool
	}{
		{
			name: "social login with google",
			user: User{
				Provider: "google",
			},
			expected: true,
		},
		{
			name: "social login with github",
			user: User{
				Provider: "github",
			},
			expected: true,
		},
		{
			name: "no social login",
			user: User{
				Provider: "",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.user.IsSocialLogin(); got != tt.expected {
				t.Errorf("User.IsSocialLogin() = %v, want %v", got, tt.expected)
			}
		})
	}
}
