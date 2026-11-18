// Package auth is the auth feature. Other packages import only this package,
// never its subpackages.
package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"log/slog"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/kilip/opus/ent"
	"github.com/kilip/opus/internal/auth/domain"
	"github.com/kilip/opus/internal/auth/handler"
	"github.com/kilip/opus/internal/auth/provider"
	"github.com/kilip/opus/internal/auth/repo"
	"github.com/kilip/opus/internal/auth/service"
	"github.com/kilip/opus/internal/platform/httpx"
	"github.com/kilip/opus/internal/platform/mailer"
)

// contextKey is a private type for context values.
type contextKey string

const userIDKey contextKey = "auth_user_id"

// SessionCookie is the name of the HTTP session cookie.
const SessionCookie = handler.SessionCookie

var ErrRateLimited = domain.ErrRateLimited

// RateLimitConfig configures OTP request rate limits.
type RateLimitConfig = handler.RateLimitConfig

// Provider is an alias for provider.Provider.
type Provider = provider.Provider

// ProviderRegistry is an alias for provider.Registry.
type ProviderRegistry = provider.Registry

// ProviderConfig is an alias for provider.Config.
type ProviderConfig = provider.Config

// NewProviderRegistry creates an OAuth2 provider registry.
func NewProviderRegistry(cfg ProviderConfig) *ProviderRegistry {
	return provider.NewRegistry(cfg)
}

// AuthConfig holds auth-specific configuration.
type AuthConfig struct {
	BaseURL       string
	SessionSecret string
	SessionTTL    time.Duration
	RateLimit     RateLimitConfig
	Registry      *provider.Registry
}

// Auth is the auth feature facade.
type Auth struct {
	handler *handler.Handler
	svc     *service.Auth
}

// New creates an Auth feature from an Ent client and config.
// If cfg.SessionSecret is empty, a random ephemeral secret is generated.
// ponytail: ephemeral secret; sessions do not survive restarts without AUTH_SECRET
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
		slog.Warn("AUTH_SECRET not set, using random secret",
			"warning", "sessions will not survive restarts")
	}

	svc := service.New(repo.New(db), repo.NewOTPRepo(db), m, service.Config{
		Secret:     secret,
		SessionTTL: cfg.SessionTTL,
		Registry:   cfg.Registry,
	})
	cc := handler.CookieConfig{
		Secure: strings.HasPrefix(cfg.BaseURL, "https"),
		MaxAge: int(cfg.SessionTTL.Seconds()),
	}
	return &Auth{handler: handler.New(svc, cc, cfg.RateLimit), svc: svc}
}

// RegisterPublic mounts public OTP routes (before auth middleware).
func (a *Auth) RegisterPublic(r fiber.Router) {
	a.handler.RegisterPublic(r)
}

// RegisterSetup mounts public setup routes (before auth middleware).
func (a *Auth) RegisterSetup(r fiber.Router) {
	a.handler.RegisterSetup(r)
}

// Register mounts auth routes (me, logout) on r.
func (a *Auth) Register(r fiber.Router) {
	a.handler.Register(r)
}

// Middleware returns a Fiber handler that enforces session authentication.
// It populates the user ID in locals so UserID() can retrieve it.
func (a *Auth) Middleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		sess, err := a.svc.Session(c.Context(), c.Cookies(handler.SessionCookie))
		if err != nil {
			return httpx.Unauthorized()
		}
		c.Locals(string(userIDKey), sess.UserID)
		return c.Next()
	}
}

// ValidateSession validates a session cookie token and returns the user ID.
func (a *Auth) ValidateSession(ctx context.Context, token string) (uuid.UUID, error) {
	sess, err := a.svc.Session(ctx, token)
	if err != nil {
		return uuid.Nil, err
	}
	return sess.UserID, nil
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
