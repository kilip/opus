package app

import (
	"context"
	"log/slog"
	"net"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/opus/ent"
)

type Opus struct {
	cfg    Config
	fiber  *fiber.App
	logger *slog.Logger
	db     *ent.Client
}

func New(cfg Config, fiberApp *fiber.App, logger *slog.Logger, db *ent.Client) *Opus {
	return &Opus{
		cfg:    cfg,
		fiber:  fiberApp,
		logger: logger,
		db:     db,
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
	if o.db != nil {
		if closeErr := o.db.Close(); closeErr != nil {
			o.logger.ErrorContext(ctx, "failed to close database client", "error", closeErr)
		}
	}
	return err
}
