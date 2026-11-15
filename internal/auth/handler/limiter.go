package handler

// ponytail: in-memory rate limiter; switch to Redis/distributed store for multi-replica deployments

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/kilip/opus/internal/auth/domain"
	"github.com/kilip/opus/internal/platform/httpx"
	"github.com/kilip/opus/internal/user"
)

const (
	defaultMaxEmailRequests = 3
	defaultMaxIPRequests    = 10
	defaultWindow           = 10 * time.Minute
	limiterEmailKey         = "limiter_normalized_email"
)

// RateLimitConfig configures OTP request rate limits.
type RateLimitConfig struct {
	MaxEmailRequests int
	MaxIPRequests    int
	Window           time.Duration
	Disabled         bool
}

// clientIP extracts the client IP address from proxy headers or connection.
// ponytail: direct proxy header check; configure Fiber TrustProxy when behind strict edge proxy
func clientIP(c fiber.Ctx) string {
	if ip := c.Get(fiber.HeaderXForwardedFor); ip != "" {
		if comma := strings.Index(ip, ","); comma != -1 {
			return strings.TrimSpace(ip[:comma])
		}
		return strings.TrimSpace(ip)
	}
	if ip := c.Get("X-Real-IP"); ip != "" {
		return strings.TrimSpace(ip)
	}
	return c.IP()
}

func clientAuditMeta(c fiber.Ctx) domain.AuditMeta {
	return domain.AuditMeta{
		IP:        clientIP(c),
		UserAgent: c.Get(fiber.HeaderUserAgent),
	}
}

func newIPLimiter(cfg RateLimitConfig) fiber.Handler {
	if cfg.Disabled {
		return func(c fiber.Ctx) error { return c.Next() }
	}
	maxRequests := cfg.MaxIPRequests
	if maxRequests <= 0 {
		maxRequests = defaultMaxIPRequests
	}
	window := cfg.Window
	if window <= 0 {
		window = defaultWindow
	}

	return limiter.New(limiter.Config{
		Max:        maxRequests,
		Expiration: window,
		KeyGenerator: func(c fiber.Ctx) string {
			return clientIP(c)
		},
		LimitReached: func(c fiber.Ctx) error {
			return httpx.NewError(fiber.StatusTooManyRequests, domain.ErrRateLimited)
		},
	})
}

func extractNormalizedEmail(c fiber.Ctx) string {
	if v, ok := c.Locals(limiterEmailKey).(string); ok {
		return v
	}
	var body domain.OTPRequestBody
	if err := json.Unmarshal(c.Body(), &body); err == nil && body.Email != "" {
		norm := user.NormalizeEmail(body.Email)
		c.Locals(limiterEmailKey, norm)
		return norm
	}
	return ""
}

func newEmailLimiter(cfg RateLimitConfig) fiber.Handler {
	if cfg.Disabled {
		return func(c fiber.Ctx) error { return c.Next() }
	}
	maxRequests := cfg.MaxEmailRequests
	if maxRequests <= 0 {
		maxRequests = defaultMaxEmailRequests
	}
	window := cfg.Window
	if window <= 0 {
		window = defaultWindow
	}

	return limiter.New(limiter.Config{
		Max:        maxRequests,
		Expiration: window,
		Next: func(c fiber.Ctx) bool {
			return extractNormalizedEmail(c) == ""
		},
		KeyGenerator: func(c fiber.Ctx) string {
			return extractNormalizedEmail(c)
		},
		LimitReached: func(c fiber.Ctx) error {
			return httpx.NewError(fiber.StatusTooManyRequests, domain.ErrRateLimited)
		},
	})
}
