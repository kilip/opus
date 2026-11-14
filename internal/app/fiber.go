package app

import (
	"github.com/gofiber/fiber/v3"
	"github.com/kilip/opus/internal/task"
)

func GetFiber() *fiber.App {
	app := fiber.New()

	// Backend API routes are registered first so they win over any SPA fallback.
	api := app.Group("/api")
	api.Get("/health", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})
	task.Register(api)

	return app
}
