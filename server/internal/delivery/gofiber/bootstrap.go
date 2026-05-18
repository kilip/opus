package gofiber

import (
	"github.com/gofiber/fiber/v3"
	"github.com/kilip/opus/server/internal/auth"
	"github.com/kilip/opus/server/internal/delivery/gofiber/handler"
	"github.com/kilip/opus/server/internal/delivery/gofiber/response"
	"github.com/kilip/opus/server/internal/shared/logger"
)

// Bootstrap initializes the HTTP delivery layer and registers routes.
func Bootstrap(app *fiber.App, log logger.Logger, cfg Config) {
	// Ensure domain services are ready
	authSvc := auth.GetService()
	authRepo := auth.GetRepository()

	// Register Auth routes
	authHandler := handler.NewAuthHandler(authSvc, authRepo)

	authGroup := app.Group("/auth")
	authGroup.Post("/register", authHandler.Register)
	authGroup.Post("/login", authHandler.Login)
	authGroup.Post("/refresh", authHandler.Refresh)
	authGroup.Post("/logout", authHandler.Logout)
	authGroup.Get("/me", authHandler.Me)
	authGroup.Get("/oauth/:provider", authHandler.OAuthRedirect)
	authGroup.Get("/oauth/:provider/callback", authHandler.OAuthCallback)

	// Register system routes
	app.Get("/health", func(c fiber.Ctx) error {
		return response.OK(c, fiber.Map{"status": "ok"})
	})
}
