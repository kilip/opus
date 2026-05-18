package handler

import (
	"github.com/gofiber/fiber/v3"
	"github.com/kilip/opus/server/internal/auth"
	"github.com/kilip/opus/server/internal/delivery/gofiber/middleware"
)

// RegisterAuthRoutes routes all registration, login, logout, and callback requests.
func RegisterAuthRoutes(app *fiber.App, h *AuthHandler, repo auth.Repository, cfg auth.Config) {
	authGroup := app.Group("/auth")

	authGroup.Post("/register", h.Register)
	authGroup.Post("/login", h.Login)
	authGroup.Post("/refresh", h.Refresh)

	// Protected OAuth callback registrations
	authGroup.Get("/oauth/:provider", h.OAuthRedirect)
	authGroup.Get("/oauth/:provider/callback", h.OAuthCallback)

	// Protected logout and profile endpoints
	authGroup.Post("/logout", h.Logout, middleware.Auth(cfg, repo))
	authGroup.Get("/me", h.Me, middleware.Auth(cfg, repo))
}
