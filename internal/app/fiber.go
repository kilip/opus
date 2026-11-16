package app

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"strings"
	"time"

	sentryfiber "github.com/getsentry/sentry-go/fiberv3"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	"github.com/gofiber/fiber/v3/middleware/static"
	"github.com/kilip/opus/ent"
	"github.com/kilip/opus/internal/auth"
	"github.com/kilip/opus/internal/platform/csrf"
	"github.com/kilip/opus/internal/platform/eventbus"
	"github.com/kilip/opus/internal/platform/httpx"
	"github.com/kilip/opus/internal/task"
	"github.com/kilip/opus/ui"
)

type HealthStatus struct {
	Status string `json:"status"`
}

// healthCheck godoc
// @Summary System health check
// @Description Returns system operational status
// @ID healthCheck
// @Tags system
// @Produce json
// @Success 200 {object} httpx.Response[HealthStatus]
// @Router /health [get]
func healthCheck(c fiber.Ctx) error {
	return httpx.OK(c, HealthStatus{Status: "ok"})
}

// GetFiber initializes HTTP server.
func GetFiber(cfg Config, logger *slog.Logger, db *ent.Client, bus *eventbus.Bus) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:      cfg.AppName,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
		BodyLimit:    cfg.BodyLimit,
		ErrorHandler: httpx.ErrorHandler(logger),
	})

	app.Use(requestid.New())
	app.Use(recover.New())
	if cfg.SentryDSN != "" {
		app.Use(sentryfiber.New(sentryfiber.Options{
			Repanic:         true,
			WaitForDelivery: true,
		}))
	}
	app.Use(func(c fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		if err != nil {
			var fErr *fiber.Error
			isClientErr := errors.As(err, &fErr) && fErr.Code < 500
			if !isClientErr && cfg.SentryDSN != "" {
				if hub := sentryfiber.GetHubFromContext(c); hub != nil {
					hub.CaptureException(err)
				}
			}
			_ = c.App().ErrorHandler(c, err)
		}
		logger.InfoContext(c.Context(), "http request",
			"request_id", requestid.FromContext(c),
			"method", c.Method(),
			"path", c.Path(),
			"status", c.Response().StatusCode(),
			"latency_ms", time.Since(start).Milliseconds(),
			"ip", c.IP(),
		)
		return err
	})

	app.Use(csrf.New(csrf.Config{
		BaseURL:        cfg.BaseURL,
		TrustedOrigins: cfg.TrustedOrigins,
	}))

	// ponytail: serve dynamic runtime configuration for SPA from system environment
	app.Get("/config.js", func(c fiber.Ctx) error {
		c.Set(fiber.HeaderContentType, "application/javascript; charset=utf-8")
		return c.SendString(fmt.Sprintf("window.__CONFIG__ = Object.assign(window.__CONFIG__ || {}, {\"APP_VERSION\": %q, \"SENTRY_DSN\": %q});\n", cfg.AppVersion, cfg.SentryDSN))
	})

	api := app.Group("/api")
	api.Get("/health", healthCheck)

	if db != nil {
		reg := auth.NewProviderRegistry(auth.ProviderConfig{
			BaseURL:              cfg.BaseURL,
			GitHubClientID:       cfg.GitHubClientID,
			GitHubClientSecret:   cfg.GitHubClientSecret,
			GoogleClientID:       cfg.GoogleClientID,
			GoogleClientSecret:   cfg.GoogleClientSecret,
			FacebookClientID:     cfg.FacebookClientID,
			FacebookClientSecret: cfg.FacebookClientSecret,
		})
		authFeature := auth.NewWithMailer(db, auth.AuthConfig{
			BaseURL:       cfg.BaseURL,
			SessionSecret: cfg.SessionSecret,
			SessionTTL:    cfg.SessionTTL,
			Registry:      reg,
		}, NewMailer(cfg, logger))
		authRoutes := api.Group("/auth")
		setupRoutes := api.Group("/setup")
		authFeature.RegisterSetup(setupRoutes) // GET /status, POST /otp, POST "" (public)
		authFeature.RegisterPublic(authRoutes) // POST /otp/request, POST /otp/verify (public)
		api.Use(authFeature.Middleware())
		authFeature.Register(authRoutes) // GET /me, POST /logout (protected)
	}
	task.New(bus).Register(api)

	// ponytail: serves embedded SPA without custom Cache-Control headers; add immutable asset caching when CDN/traffic requires.
	if distFS, err := ui.Dist(); err == nil {
		app.Use(static.New("", static.Config{
			FS: distFS,
			Next: func(c fiber.Ctx) bool {
				return strings.HasPrefix(c.Path(), "/api")
			},
			NotFoundHandler: func(c fiber.Ctx) error {
				index, err := fs.ReadFile(distFS, "index.html")
				if err != nil {
					return fiber.ErrNotFound
				}
				c.Set(fiber.HeaderContentType, fiber.MIMETextHTMLCharsetUTF8)
				return c.Send(index)
			},
		}))
	}

	return app
}
