package handler

// ponytail: in-memory rate limiter; switch to Redis/distributed store for multi-replica deployments

import (
	"encoding/json"
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

func clientAuditMeta(c fiber.Ctx) domain.AuditMeta {
	return domain.AuditMeta{
		IP:        c.IP(),
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
			return c.IP()
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
	raw := c.Body()
	if len(raw) > 4096 {
		return ""
	}
	var body domain.OTPRequestBody
	if err := json.Unmarshal(raw, &body); err == nil && body.Email != "" {
		norm := user.NormalizeEmail(body.Email)
		if len(norm) > 254 {
			return ""
		}
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
