// Package middleware provides HTTP middleware for the GoFiber server.
package middleware

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/kilip/opus/server/internal/shared/logger"
)

// RequestLogger returns a Fiber middleware that logs HTTP requests and injects a request ID.
func RequestLogger(log logger.Logger) fiber.Handler {
	return func(c fiber.Ctx) error {
		requestID := uuid.NewString()
		ctx := logger.WithRequestID(c.Context(), requestID)
		c.SetContext(ctx)

		err := c.Next()

		log.InfoCtx(ctx, "request completed",
			logger.String("method", c.Method()),
			logger.String("path", c.Path()),
			logger.Int("status", c.Response().StatusCode()),
			logger.String("request_id", requestID),
		)
		return err
	}
}
