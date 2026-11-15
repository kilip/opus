// Package auth is the auth feature. Other packages import only this package,
// never its subpackages.
package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"log/slog"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/kilip/opus/ent"
	"github.com/kilip/opus/internal/auth/handler"
	"github.com/kilip/opus/internal/auth/repo"
	"github.com/kilip/opus/internal/platform/httpx"
	"github.com/kilip/opus/internal/platform/mailer"
)

// contextKey is a private type for context values.
type contextKey string

const userIDKey contextKey = "auth_user_id"

// AuthConfig holds auth-specific configuration.
type AuthConfig struct {
	BaseURL       string
	SessionSecret string
	SessionTTL    time.Duration
}

// Auth is the auth feature facade.
type Auth struct {
	handler *handler.Handler
	secret  string
	repo    repo.Repo
}

// New creates an Auth feature from an Ent client and config.
// If cfg.SessionSecret is empty, a random ephemeral secret is generated.
// ponytail: ephemeral secret; sessions do not survive restarts without SESSION_SECRET
func New(db *ent.Client, cfg AuthConfig) *Auth {
	return NewWithMailer(db, cfg, nil)
}

// NewWithMailer creates an Auth feature with a mailer for OTP support.
func NewWithMailer(db *ent.Client, cfg AuthConfig, m mailer.Mailer) *Auth {
	secret := cfg.SessionSecret
	if secret == "" {
		b := make([]byte, 32)
		_, _ = rand.Read(b)
		secret = base64.RawURLEncoding.EncodeToString(b)
		slog.Warn("SESSION_SECRET not set, using random secret",
			"warning", "sessions will not survive restarts")
	}

	r := repo.New(db)
	otpR := repo.NewOTPRepo(db)
	cc := handler.CookieConfig{
		Secure:  strings.HasPrefix(cfg.BaseURL, "https"),
		MaxAge:  int(cfg.SessionTTL.Seconds()),
		BaseURL: cfg.BaseURL,
	}
	h := handler.NewWithOTP(r, otpR, m, secret, cc)
	return &Auth{handler: h, secret: secret, repo: r}
}

// RegisterPublic mounts public OTP routes (before auth middleware).
func (a *Auth) RegisterPublic(r fiber.Router) {
	a.handler.RegisterPublic(r)
}

// Register mounts auth routes (me, logout) on r.
func (a *Auth) Register(r fiber.Router) {
	a.handler.Register(r)
}

// Middleware returns a Fiber handler that enforces session authentication.
// It populates the user ID in locals so UserID() can retrieve it.
func (a *Auth) Middleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		raw := c.Cookies("opus_session")
		if raw == "" {
			return httpx.Unauthorized()
		}
		// ponytail: duplicate hash logic avoids a cross-call back into handler subpackage
		mac := hmacHash(raw, a.secret)
		sess, err := a.repo.FindByTokenHash(c.Context(), mac)
		if err != nil || sess == nil {
			return httpx.Unauthorized()
		}
		if time.Now().After(sess.ExpiresAt) {
			return httpx.Unauthorized()
		}
		c.Locals(string(userIDKey), sess.UserID)
		return c.Next()
	}
}

// UserID returns the authenticated user's UUID from the Fiber context.
// Returns uuid.Nil if not set (unauthenticated or middleware not applied).
func UserID(c fiber.Ctx) uuid.UUID {
	v := c.Locals(string(userIDKey))
	if id, ok := v.(uuid.UUID); ok {
		return id
	}
	return uuid.Nil
}

// hmacHash returns the HMAC-SHA256 base64url hash of raw keyed with secret.
func hmacHash(raw, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(raw))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
