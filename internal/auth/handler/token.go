package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/opus/internal/auth/domain"
	"github.com/kilip/opus/internal/auth/repo"
)

// hashToken returns the HMAC-SHA256 base64url hash of raw keyed with secret.
func hashToken(raw, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(raw))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// resolveSession reads the opus_session cookie, validates it, and returns the domain session.
// Returns domain.ErrUnauthorized on any failure.
func resolveSession(c fiber.Ctx, r repo.Repo, secret string) (*domain.Session, error) {
	raw := c.Cookies("opus_session")
	if raw == "" {
		return nil, domain.ErrUnauthorized
	}
	hash := hashToken(raw, secret)
	sess, err := r.FindByTokenHash(c.Context(), hash)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}
	if time.Now().After(sess.ExpiresAt) {
		return nil, domain.ErrUnauthorized
	}
	return sess, nil
}
