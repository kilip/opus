package auth_test

import (
	"testing"

	"github.com/kilip/opus/server/internal/auth"
)

func TestAuthDomainDefinitions(t *testing.T) {
	// Assert error definitions
	if auth.ErrInvalidCredentials.Error() != "auth: invalid credentials" {
		t.Errorf("unexpected error message: %s", auth.ErrInvalidCredentials.Error())
	}
	if auth.ErrEmailAlreadyExists.Error() != "auth: email already registered" {
		t.Errorf("unexpected error message: %s", auth.ErrEmailAlreadyExists.Error())
	}

	// Assert updated User struct fields
	u := auth.User{
		ID:           "test-uuid-v7",
		Email:        "user@test.com",
		Name:         "Test User",
		AvatarURL:    "https://avatar.com/test",
		Provider:     "google",
		ProviderID:   "g123",
		WorkspaceID:  "ws-uuid-v7",
		PasswordHash: "hashedpass",
	}

	if u.ID != "test-uuid-v7" || u.WorkspaceID != "ws-uuid-v7" {
		t.Errorf("incorrect fields on User struct")
	}

	// Assert Config definition
	c := auth.Config{
		JWTSecret:       "super-secret-key",
		AccessTokenTTL:  "15m",
		RefreshTokenTTL: "168h",
	}

	if c.JWTSecret != "super-secret-key" {
		t.Errorf("incorrect configuration fields")
	}
}
