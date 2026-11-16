package app

import (
	"context"
	"log/slog"
	"net"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/opus/ent"
	"github.com/kilip/opus/internal/platform/eventbus"
	"github.com/kilip/opus/internal/platform/mailer"
)

// NewMailer creates a mailer.Mailer based on configuration.
// If SMTPHost is empty, it returns a console mailer and logs a startup warning.
// Otherwise, it returns an SMTP mailer configured with SMTP settings.
func NewMailer(cfg Config, logger *slog.Logger) mailer.Mailer {
	if cfg.SMTPHost == "" {
		if logger != nil {
			logger.Warn("SMTP_HOST not set, falling back to console mailer")
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

type Opus struct {
	cfg    Config
	fiber  *fiber.App
	logger *slog.Logger
	db     *ent.Client
	bus    *eventbus.Bus
}

func New(cfg Config, fiberApp *fiber.App, logger *slog.Logger, db *ent.Client, bus *eventbus.Bus) *Opus {
	return &Opus{
		cfg:    cfg,
		fiber:  fiberApp,
		logger: logger,
		db:     db,
		bus:    bus,
	}
}

func (o *Opus) Serve(ctx context.Context) error {
	addr := net.JoinHostPort(o.cfg.Host, o.cfg.Port)
	o.logger.InfoContext(ctx, "starting server", "addr", addr)
	err := o.fiber.Listen(addr, fiber.ListenConfig{
		DisableStartupMessage: true,
		GracefulContext:       ctx,
		ShutdownTimeout:       o.cfg.ShutdownTimeout,
	})

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
