package app

import (
	"io/fs"
	"log/slog"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	"github.com/gofiber/fiber/v3/middleware/static"
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
func GetFiber(cfg Config, logger *slog.Logger) *fiber.App {
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
	app.Use(func(c fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		if err != nil {
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

	api := app.Group("/api")
	api.Get("/health", healthCheck)
	task.New().Register(api)

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
