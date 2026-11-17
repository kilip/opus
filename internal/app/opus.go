package app

import (
	"context"
	"log/slog"
	"net"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/opus/ent"
	"github.com/kilip/opus/internal/node"
	"github.com/kilip/opus/internal/platform/eventbus"
	"github.com/kilip/opus/internal/platform/mailer"
	"github.com/kilip/opus/internal/tracker"
)

// NewMailer creates a mailer.Mailer based on configuration.
// If SMTPHost is empty, it returns a console mailer and logs a startup warning.
// Otherwise, it returns an SMTP mailer configured with SMTP settings.
func NewMailer(cfg Config, logger *slog.Logger) mailer.Mailer {
	if cfg.SMTPHost == "" {
		if logger != nil {
			logger.Warn("APP_SMTP_HOST not set, falling back to console mailer")
		}
		return mailer.NewConsole(logger)
	}
	return mailer.NewSMTP(mailer.SMTPConfig{
		Host:     cfg.SMTPHost,
		Port:     cfg.SMTPPort,
		Username: cfg.SMTPUsername,
		Password: cfg.SMTPPassword,
		From:     cfg.SMTPFrom,
	})
}

// Option configures optional feature instances wired into App/Opus.
type Option func(*options)

type options struct {
	node    *node.Node
	tracker *tracker.Tracker
}

// WithNode injects a pre-configured Node feature into GetFiber and Opus.
func WithNode(n *node.Node) Option {
	return func(o *options) {
		o.node = n
	}
}

// WithTracker injects a pre-configured Tracker feature into GetFiber and Opus.
func WithTracker(t *tracker.Tracker) Option {
	return func(o *options) {
		o.tracker = t
	}
}

type Opus struct {
	cfg     Config
	fiber   *fiber.App
	logger  *slog.Logger
	db      *ent.Client
	bus     *eventbus.Bus
	node    *node.Node
	tracker *tracker.Tracker
}

func New(cfg Config, fiberApp *fiber.App, logger *slog.Logger, db *ent.Client, bus *eventbus.Bus, opts ...Option) *Opus {
	var o options
	for _, opt := range opts {
		if opt != nil {
			opt(&o)
		}
	}
	return &Opus{
		cfg:     cfg,
		fiber:   fiberApp,
		logger:  logger,
		db:      db,
		bus:     bus,
		node:    o.node,
		tracker: o.tracker,
	}
}

func (o *Opus) Serve(ctx context.Context) error {
	if o.node != nil {
		if err := o.node.Start(ctx); err != nil {
			o.logger.ErrorContext(ctx, "failed to start node detector", "error", err)
		}
	}
	if o.tracker != nil {
		if err := o.tracker.Start(ctx); err != nil {
			o.logger.ErrorContext(ctx, "failed to start tracker poller", "error", err)
		}
	}

	addr := net.JoinHostPort(o.cfg.Host, o.cfg.Port)
	o.logger.InfoContext(ctx, "starting server", "addr", addr)
	err := o.fiber.Listen(addr, fiber.ListenConfig{
		DisableStartupMessage: true,
		GracefulContext:       ctx,
		ShutdownTimeout:       o.cfg.ShutdownTimeout,
	})

	if o.node != nil {
		o.node.Stop()
	}
	if o.tracker != nil {
		o.tracker.Stop()
	}

	if o.bus != nil {
		timeout := o.cfg.ShutdownTimeout
		if timeout <= 0 {
			timeout = 10 * time.Second
		}
		closeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), timeout)
		if closeErr := o.bus.Close(closeCtx); closeErr != nil {
			o.logger.ErrorContext(ctx, "failed to close event bus", "error", closeErr)
		}
		cancel()
	}

	if o.db != nil {
		if closeErr := o.db.Close(); closeErr != nil {
			o.logger.ErrorContext(ctx, "failed to close database client", "error", closeErr)
		}
	}
	return err
}
