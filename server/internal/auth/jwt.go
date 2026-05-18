package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TokenPair encapsulates cryptographically secure random string and JWT representations.
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	SessionID    string `json:"-"`
}

// GenerateTokenPair builds a stateful access and refresh token pair mapping.
func GenerateTokenPair(secret string, claims Claims, accessTTL, refreshTTL time.Duration) (*TokenPair, error) {
	now := time.Now()
	claims.IssuedAt = jwt.NewNumericDate(now)
	claims.ExpiresAt = jwt.NewNumericDate(now.Add(accessTTL))

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(secret))
	if err != nil {
		return nil, err
	}

	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return nil, err
	}
	refreshTokenStr := hex.EncodeToString(bytes)

	return &TokenPair{
		AccessToken:  tokenStr,
		RefreshToken: refreshTokenStr,
		SessionID:    claims.SessionID,
	}, nil
}

// HashToken calculates standard SHA-256 signatures for database checks.
func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
