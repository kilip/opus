package auth_test

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/kilip/opus/server/internal/auth"
)

func TestJWTGenerationAndHashing(t *testing.T) {
	secret := "secret-jwt-key"
	claims := auth.Claims{
		SessionID:   "s-123",
		WorkspaceID: "w-123",
		Role:        "admin",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject: "user-123",
		},
	}

	pair, err := auth.GenerateTokenPair(secret, claims, 10*time.Minute, time.Hour)
	if err != nil {
		t.Fatalf("failed to generate token pair: %v", err)
	}

	if pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Fatalf("access or refresh token is empty")
	}

	// Validate access token claims
	token, err := jwt.ParseWithClaims(pair.AccessToken, &auth.Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		t.Fatalf("failed to parse access token: %v", err)
	}

	parsedClaims, ok := token.Claims.(*auth.Claims)
	if !ok || parsedClaims.Subject != "user-123" || parsedClaims.WorkspaceID != "w-123" {
		t.Errorf("claims mapping mismatch")
	}

	// Validate hashing
	hash1 := auth.HashToken("sample-string")
	hash2 := auth.HashToken("sample-string")
	if hash1 != hash2 {
		t.Errorf("hash mismatch")
	}
}
