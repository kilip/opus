// Package gofiber implements the HTTP delivery layer using the GoFiber framework.
package gofiber

import (
	"github.com/gofiber/fiber/v3"
	"github.com/kilip/opus/server/internal/delivery/gofiber/middleware"
	"github.com/kilip/opus/server/internal/delivery/gofiber/response"
	"github.com/kilip/opus/server/internal/shared/logger"
)

// New creates and configures a new GoFiber application instance.
func New(cfg Config, log logger.Logger) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:   "Opus Server",
		BodyLimit: cfg.BodyLimit,
		ErrorHandler: func(c fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return response.Error(c, code, slugFromStatus(code), titleFromStatus(code), err.Error())
		},
	})

	// Global middleware - order is critical
	app.Use(middleware.CORS(cfg.CORS))
	app.Use(middleware.RequestLogger(log))

	app.Get("/health", func(c fiber.Ctx) error {
		return response.OK(c, fiber.Map{"status": "ok"})
	})

	return app
}

func slugFromStatus(code int) string {
	switch code {
	case 400:
		return "bad-request"
	case 401:
		return "unauthorized"
	case 403:
		return "forbidden"
	case 404:
		return "not-found"
	case 422:
		return "unprocessable-entity"
	default:
		return "internal-server-error"
	}
}

func titleFromStatus(code int) string {
	switch code {
	case 400:
		return "Bad Request"
	case 401:
		return "Unauthorized"
	case 403:
		return "Forbidden"
	case 404:
		return "Resource Not Found"
	case 422:
		return "Validation Failed"
	default:
		return "Internal Server Error"
	}
}
