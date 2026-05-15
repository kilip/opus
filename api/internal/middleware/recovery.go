package middleware

import (
	"log/slog"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/opus/api/internal/config"
)

func Recovery() fiber.Handler {
	logger := config.GetLogger()
	return func(c fiber.Ctx) error {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("panic recovered",
					slog.Any("panic", r),
					slog.String("method", c.Method()),
					slog.String("path", c.Path()),
				)

				_ = c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"success": false,
					"data":    nil,
					"error": fiber.Map{
						"code":    "INTERNAL_SERVER_ERROR",
						"message": "An unexpected error occurred",
					},
				})
			}
		}()
		return c.Next()
	}
}
