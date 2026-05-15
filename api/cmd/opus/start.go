package main

import (
	"fmt"
	"log"

	"os"
	"path/filepath"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/opus/api/internal/config"
	"github.com/kilip/opus/api/internal/handler"
	"github.com/kilip/opus/api/internal/middleware"
	"github.com/kilip/opus/api/internal/repository"
	"github.com/kilip/opus/api/internal/service"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the API server",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.GetConfig()
		db := config.GetDatabase()

		// Ensure .opus directory exists
		opusDir := filepath.Join(os.Getenv("HOME"), ".opus")
		_ = os.MkdirAll(opusDir, 0755)

		// Save PID
		pidFile := filepath.Join(opusDir, "opus.pid")
		_ = os.WriteFile(pidFile, []byte(fmt.Sprintf("%d", os.Getpid())), 0644)

		// Repositories
		userRepo := repository.NewUserRepository(db)
		sessionRepo := repository.NewSessionRepository(db)

		// Services
		authService := service.NewAuthService(userRepo, sessionRepo, cfg)
		userService := service.NewUserService(userRepo, cfg)

		// Handlers
		authHandler := handler.NewAuthHandler(authService, userService, cfg)
		userHandler := handler.NewUserHandler(userService)
		healthHandler := handler.NewHealthHandler()
		sseHandler := handler.NewSSEHandler()

		// App initialization
		app := fiber.New(fiber.Config{
			AppName:      "Opus API",
			ErrorHandler: middleware.ErrorHandler,
		})

		// Global Middleware
		app.Use(middleware.Logger())
		app.Use(middleware.Recovery())

		// Public Routes
		api := app.Group("/api")
		api.Get("/health", healthHandler.Check)
		
		auth := api.Group("/auth")
		auth.Post("/login", authHandler.Login)
		auth.Post("/refresh", authHandler.Refresh)
		auth.Post("/logout", authHandler.Logout)
		auth.Get("/google", authHandler.GoogleLogin)
		auth.Get("/google/callback", authHandler.GoogleCallback)

		// Protected Routes
		protected := api.Group("/")
		protected.Use(middleware.Auth(authService))
		
		protected.Get("/user/me", userHandler.Me)
		protected.Get("/stream", sseHandler.Stream)

		addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
		log.Printf("Starting server on %s", addr)
		if err := app.Listen(addr); err != nil {
			log.Fatalf("Server failed to start: %v", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
}
