package middleware

import (
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/opus/api/internal/config"
)

func Logger() fiber.Handler {
	logger := config.GetLogger()
	return func(c fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		latency := time.Since(start)

		logger.Info("request processed",
			slog.String("method", c.Method()),
			slog.String("path", c.Path()),
			slog.Int("status", c.Response().StatusCode()),
			slog.Duration("latency", latency),
			slog.String("ip", c.IP()),
		)

		return err
	}
}
