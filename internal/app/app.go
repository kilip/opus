package app

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v3"
)

// App encapsulates the Opus HTTP server, task supervisor, and background services.
type App struct {
	cfg   Config
	fiber *fiber.App
}

func New(cfg Config) *App {
	return &App{
		cfg:   cfg,
		fiber: GetFiber(),
	}
}

// Serve starts the server and blocks until ctx is canceled or the server encounters a fatal error.
func (a *App) Serve(ctx context.Context) error {
	// Listen for OS interrupt signals for graceful shutdown
	// ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	// defer stop()

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- a.fiber.Listen(":7880")
	}()

	select {
	case err := <-serverErr:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return a.fiber.ShutdownWithContext(shutdownCtx)
	}
}
