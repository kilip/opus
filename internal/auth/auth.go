// Package auth is the auth feature. Other packages import only this package,
// never its subpackages.
package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
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
	AppHome       string
	SessionSecret string
	SessionTTL    time.Duration
	SetupToken    string
	CookieSecure  string
	RateLimit     RateLimitConfig
	Registry      *provider.Registry
}

// Auth is the auth feature facade.
type Auth struct {
	handler *handler.Handler
	svc     *service.Auth
	secret  string
}

// New creates an Auth feature from an Ent client and config.
func New(db *ent.Client, cfg AuthConfig) (*Auth, error) {
	return NewWithMailer(db, cfg, nil)
}

// NewWithMailer creates an Auth feature with a mailer for OTP support.
func NewWithMailer(db *ent.Client, cfg AuthConfig, m mailer.Mailer) (*Auth, error) {
	secret, err := resolveSessionSecret(cfg.SessionSecret, cfg.AppHome)
	if err != nil {
		return nil, err
	}

	cookieSecure, err := resolveCookieSecure(cfg.CookieSecure, cfg.BaseURL)
	if err != nil {
		return nil, err
	}

	svc := service.New(repo.New(db), repo.NewOTPRepo(db), m, service.Config{
		Secret:     secret,
		SessionTTL: cfg.SessionTTL,
		Registry:   cfg.Registry,
		SetupToken: cfg.SetupToken,
	})

	if needs, err := svc.NeedsSetup(context.Background()); err == nil && needs {
		token := cfg.SetupToken
		if token == "" {
			b := make([]byte, 24)
			if _, randErr := rand.Read(b); randErr != nil {
				return nil, fmt.Errorf("generate setup token: %w", randErr)
			}
			token = base64.RawURLEncoding.EncodeToString(b)
			svc.SetSetupToken(token)
		}
		base := strings.TrimRight(cfg.BaseURL, "/")
		_, _ = fmt.Fprintf(os.Stderr, "\nOpus setup token: %s\nSetup URL: %s/onboarding\n\n", token, base)
	}

	cc := handler.CookieConfig{
		Secure: cookieSecure,
		MaxAge: int(cfg.SessionTTL.Seconds()),
	}
	return &Auth{handler: handler.New(svc, cc, cfg.RateLimit), svc: svc, secret: secret}, nil
}

// SessionSecret returns the resolved session signing secret.
func (a *Auth) SessionSecret() string {
	if a == nil {
		return ""
	}
	return a.secret
}

// ResolveSessionSecret returns the configured AUTH_SECRET or loads/generates $APP_HOME/auth_secret.
func ResolveSessionSecret(configured, appHome string) (string, error) {
	return resolveSessionSecret(configured, appHome)
}

func resolveSessionSecret(configured, appHome string) (string, error) {
	if configured != "" {
		if len(configured) < 32 {
			return "", fmt.Errorf("AUTH_SECRET must be at least 32 characters (got %d)", len(configured))
		}
		return configured, nil
	}

	if appHome != "" {
		secretPath := filepath.Join(appHome, "auth_secret")
		if raw, err := os.ReadFile(secretPath); err == nil {
			s := strings.TrimSpace(string(raw))
			if len(s) >= 32 {
				return s, nil
			}
		}
		b := make([]byte, 32)
		if _, err := rand.Read(b); err == nil {
			s := base64.RawURLEncoding.EncodeToString(b)
			if mkErr := os.MkdirAll(appHome, 0o700); mkErr == nil {
				f, openErr := os.OpenFile(secretPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
				if openErr == nil {
					_, writeErr := f.WriteString(s + "\n")
					_ = f.Close()
					if writeErr == nil {
						return s, nil
					}
				} else if errors.Is(openErr, os.ErrExist) {
					if raw, readErr := os.ReadFile(secretPath); readErr == nil {
						existing := strings.TrimSpace(string(raw))
						if len(existing) >= 32 {
							return existing, nil
						}
					}
				}
			}
		}
		slog.Warn("AUTH_SECRET not set and failed to persist auth_secret in APP_HOME, using ephemeral secret",
			"app_home", appHome,
			"warning", "sessions will not survive restarts")
	} else {
		slog.Warn("AUTH_SECRET not set, using random secret",
			"warning", "sessions will not survive restarts")
	}

	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func resolveCookieSecure(mode, baseURL string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "", "auto":
		return strings.HasPrefix(baseURL, "https://"), nil
	case "true":
		return true, nil
	case "false":
		return false, nil
	default:
		return false, fmt.Errorf("invalid AUTH_COOKIE_SECURE %q: must be auto, true, or false", mode)
	}
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
