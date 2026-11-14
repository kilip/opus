package app

import (
	"context"
	"log/slog"
	"net"

	"github.com/gofiber/fiber/v3"
)

type Opus struct {
	cfg    Config
	fiber  *fiber.App
	logger *slog.Logger
}

func New(cfg Config, fiberApp *fiber.App, logger *slog.Logger) *Opus {
	return &Opus{
		cfg:    cfg,
		fiber:  fiberApp,
		logger: logger,
	}
}

func (o *Opus) Serve(ctx context.Context) error {
	addr := net.JoinHostPort(o.cfg.Host, o.cfg.Port)
	o.logger.InfoContext(ctx, "starting server", "addr", addr)
	return o.fiber.Listen(addr, fiber.ListenConfig{
		DisableStartupMessage: true,
		GracefulContext:       ctx,
		ShutdownTimeout:       o.cfg.ShutdownTimeout,
	})
}
