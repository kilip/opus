package main

import (
	"fmt"
	"log"

	"os"
	"path/filepath"

	"github.com/kilip/opus/api/internal/config"
	"github.com/kilip/opus/api/internal/delivery/fiber"
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

		// Delivery
		srv := fiber.NewServer(cfg, authService, userService)

		log.Printf("Starting server on %s:%d", cfg.Server.Host, cfg.Server.Port)
		if err := srv.Start(); err != nil {
			log.Fatalf("Server failed to start: %v", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
}
