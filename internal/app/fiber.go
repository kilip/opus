package app

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"path"
	"strings"
	"sync/atomic"
	"time"

	sentryfiber "github.com/getsentry/sentry-go/fiberv3"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/helmet"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	"github.com/gofiber/fiber/v3/middleware/static"
	"github.com/kilip/opus/ent"
	"github.com/kilip/opus/internal/agent"
	"github.com/kilip/opus/internal/auth"
	"github.com/kilip/opus/internal/node"
	"github.com/kilip/opus/internal/platform/csrf"
	"github.com/kilip/opus/internal/platform/eventbus"
	"github.com/kilip/opus/internal/platform/httpx"
	"github.com/kilip/opus/internal/project"
	"github.com/kilip/opus/internal/run"
	"github.com/kilip/opus/internal/session"
	"github.com/kilip/opus/internal/settings"
	"github.com/kilip/opus/internal/task"
	"github.com/kilip/opus/internal/tracker"
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

// readyCheck godoc
// @Summary System readiness check
// @Description Verifies database connectivity and server readiness
// @ID readyCheck
// @Tags system
// @Produce json
// @Success 200 {object} httpx.Response[HealthStatus]
// @Failure 503 {object} httpx.ErrorResponse
// @Router /readyz [get]
func readyCheckHandler(db *ent.Client, shuttingDown *atomic.Bool) fiber.Handler {
	return func(c fiber.Ctx) error {
		if shuttingDown != nil && shuttingDown.Load() {
			return httpx.NewError(fiber.StatusServiceUnavailable, errors.New("shutting down"))
		}
		if db == nil {
			return httpx.NewError(fiber.StatusServiceUnavailable, errors.New("database not configured"))
		}
		pingCtx, cancel := context.WithTimeout(c.Context(), time.Second)
		defer cancel()
		if _, err := db.Meta.Query().Limit(1).Count(pingCtx); err != nil {
			return httpx.NewError(fiber.StatusServiceUnavailable, err)
		}
		return httpx.OK(c, HealthStatus{Status: "ok"})
	}
}

// GetFiber initializes HTTP server.
func GetFiber(cfg Config, logger *slog.Logger, db *ent.Client, bus *eventbus.Bus, opts ...Option) (*fiber.App, error) {
	var o options
	for _, opt := range opts {
		if opt != nil {
			opt(&o)
		}
	}
	fiberCfg := fiber.Config{
		AppName:      cfg.AppName,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
		BodyLimit:    cfg.BodyLimit,
		ErrorHandler: httpx.ErrorHandler(logger),
	}
	if len(cfg.TrustedProxies) > 0 {
		fiberCfg.TrustProxy = true
		fiberCfg.TrustProxyConfig = fiber.TrustProxyConfig{
			Proxies: cfg.TrustedProxies,
		}
		fiberCfg.ProxyHeader = fiber.HeaderXForwardedFor
	}
	app := fiber.New(fiberCfg)

	app.Use(requestid.New())
	app.Use(recover.New())
	helmetCfg := helmet.Config{
		XSSProtection:         "0",
		ContentTypeNosniff:    "nosniff",
		XFrameOptions:         "DENY",
		ReferrerPolicy:        "strict-origin-when-cross-origin",
		ContentSecurityPolicy: "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; connect-src 'self'; img-src 'self' data:; frame-ancestors 'none'",
		HSTSMaxAge:            0,
		HSTSExcludeSubdomains: true,
	}
	if strings.HasPrefix(strings.ToLower(cfg.BaseURL), "https://") {
		helmetCfg.HSTSMaxAge = 31536000
	}
	app.Use(helmet.New(helmetCfg))
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
		if c.Method() == fiber.MethodGet && (c.Path() == "/api/health" || c.Path() == "/api/readyz") {
			return err
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

	csrfMiddleware := csrf.New(csrf.Config{
		BaseURL:        cfg.BaseURL,
		TrustedOrigins: cfg.TrustedOrigins,
	})
	app.Use(func(c fiber.Ctx) error {
		if c.Path() == "/mcp" || strings.HasPrefix(c.Path(), "/mcp/") {
			return c.Next()
		}
		return csrfMiddleware(c)
	})

	// ponytail: serve dynamic runtime configuration for SPA from system environment
	app.Get("/config.js", func(c fiber.Ctx) error {
		c.Set(fiber.HeaderContentType, "application/javascript; charset=utf-8")
		return c.SendString(fmt.Sprintf("window.__CONFIG__ = Object.assign(window.__CONFIG__ || {}, {\"APP_VERSION\": %q, \"SENTRY_DSN\": %q});\n", cfg.AppVersion, cfg.SentryDSN))
	})

	api := app.Group("/api")
	api.Get("/health", healthCheck)
	api.Get("/readyz", readyCheckHandler(db, o.shuttingDown))

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
		authFeature, err := auth.NewWithMailer(db, auth.AuthConfig{
			BaseURL:       cfg.BaseURL,
			AppHome:       cfg.AppHome,
			SessionSecret: cfg.SessionSecret,
			SessionTTL:    cfg.SessionTTL,
			SetupToken:    cfg.AuthSetupToken,
			CookieSecure:  cfg.AuthCookieSecure,
			Registry:      reg,
		}, NewMailer(cfg, logger))
		if err != nil {
			return nil, err
		}
		authRoutes := api.Group("/auth")
		setupRoutes := api.Group("/setup")
		authFeature.RegisterSetup(setupRoutes) // GET /status, POST /otp, POST "" (public)
		authFeature.RegisterPublic(authRoutes) // POST /otp/request, POST /otp/verify (public)
		var nodeFeature *node.Node
		if o.node != nil {
			nodeFeature = o.node
		} else {
			nodeFeature = node.New(db, node.Config{
				BaseURL:        cfg.BaseURL,
				TrustedOrigins: cfg.TrustedOrigins,
				Bus:            bus,
			})
		}
		if err := nodeFeature.Bootstrap(context.Background()); err != nil {
			logger.Error("node registry bootstrap failed", "error", err)
		}
		nodeFeature.RegisterPublic(api)
		api.Use(authFeature.Middleware())
		authFeature.Register(authRoutes) // GET /me, POST /logout (protected)
		var (
			projectFeature  *project.Project
			runFeature      *run.Feature
			sessionFeature  *session.Session
			settingsFeature *settings.Settings
			taskFeature     *task.Task
			trackerFeature  *tracker.Tracker
		)
		if o.settings != nil {
			settingsFeature = o.settings
		} else {
			settingsFeature, err = settings.New(db, settings.Config{
				Secret: authFeature.SessionSecret(),
			})
			if err != nil {
				return nil, err
			}
		}
		if o.project != nil {
			projectFeature = o.project
		} else {
			projectFeature = project.New(db,
				project.WithNodeReader(nodeFeature),
				project.WithTaskCounter(func(ctx context.Context, projectID string) (int, error) {
					if taskFeature == nil {
						return 0, nil
					}
					return taskFeature.CountByProject(ctx, projectID)
				}),
				project.WithTrackerLabelValidator(func(ctx context.Context, labels map[string]string) error {
					if trackerFeature == nil {
						return nil
					}
					return trackerFeature.ValidateProjectLabels(ctx, labels)
				}),
			)
		}
		if o.task != nil {
			taskFeature = o.task
		} else {
			taskFeature = task.New(db, bus, task.WithProjectChecker(projectFeature))
		}
		if o.tracker != nil {
			trackerFeature = o.tracker
		} else {
			trackerFeature = tracker.New(db, tracker.Config{
				PollInterval: cfg.TrackerPollInterval,
				GitHubTokenProvider: func() string {
					return settingsFeature.EffectiveGitHubToken(context.Background())
				},
				Logger: logger,
				Tasks:  taskFeature,
				Bus:    bus,
			})
		}
		if o.session != nil {
			sessionFeature = o.session
		} else {
			sessionFeature = session.New(db, session.Config{Bus: bus})
			nodeFeature.Hub().OnEvent(func(nodeID string, env node.Envelope) {
				sessionFeature.HandleWorkerEvent(context.Background(), nodeID, env)
			})
		}
		if o.run != nil {
			runFeature = o.run
		} else {
			runFeature = run.New(db, run.Config{
				Bus:      bus,
				AppHome:  cfg.AppHome,
				Tasks:    taskFeature,
				Projects: projectFeature,
				Sessions: sessionFeature,
				Logger:   logger,
			})
		}
		taskFeature.Register(api)
		nodeFeature.Register(api)
		projectFeature.Register(api)
		runFeature.Register(api)
		sessionFeature.Register(api)
		settingsFeature.Register(api)
		trackerFeature.Register(api)

		agentFeature := agent.New(agent.Config{
			Tasks:    taskFeature,
			Runs:     runFeature,
			Projects: projectFeature,
			Sessions: sessionFeature,
			Nodes:    nodeFeature,
			Auth:     authFeature,
		})
		agentFeature.Register(app)
	}

	// ponytail: serves embedded SPA without custom Cache-Control headers; add immutable asset caching when CDN/traffic requires.
	distFS := o.distFS
	var distErr error
	if distFS == nil {
		distFS, distErr = ui.Dist()
	}
	if distErr == nil && distFS != nil {
		app.Get("/*", static.New("", static.Config{
			FS: distFS,
			Next: func(c fiber.Ctx) bool {
				return strings.HasPrefix(c.Path(), "/api") ||
					c.Path() == "/mcp" ||
					strings.HasPrefix(c.Path(), "/mcp/")
			},
			NotFoundHandler: func(c fiber.Ctx) error {
				if path.Ext(c.Path()) != "" {
					return fiber.ErrNotFound
				}
				index, err := fs.ReadFile(distFS, "index.html")
				if err != nil {
					return fiber.ErrNotFound
				}
				c.Set(fiber.HeaderContentType, fiber.MIMETextHTMLCharsetUTF8)
				return c.Status(fiber.StatusOK).Send(index)
			},
		}))
	}

	return app, nil
}
