package main

import (
	"context"
	"fmt"
	"log"

	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/kilip/opus/api/internal/config"
	"github.com/kilip/opus/api/internal/delivery/fiber"
	"github.com/kilip/opus/api/internal/repository"
	"github.com/kilip/opus/api/internal/service"
	"github.com/kilip/opus/api/internal/worker"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the API server",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.GetConfig()
		db := config.GetDatabase()

		// Ensure .opus directory exists
		opusDir := config.GetOpusDir()
		_ = os.MkdirAll(opusDir, 0755)

		// Save PID
		pidFile := filepath.Join(opusDir, "opus.pid")
		_ = os.WriteFile(pidFile, []byte(fmt.Sprintf("%d", os.Getpid())), 0644)

		// Repositories
		userRepo := repository.NewUserRepository(db)
		sessionRepo := repository.NewSessionRepository(db)

		// Queue System
		logger := config.GetLogger()
		queueDriver := config.GetQueueDriver()
		workerEngine := worker.NewWorkerEngine(queueDriver, cfg.Queue.Concurrency, logger)
		scheduler := worker.NewScheduler(queueDriver, logger)

		// Services
		authService := service.NewAuthService(userRepo, sessionRepo, cfg)
		userService := service.NewUserService(userRepo, cfg)

		// Start Queue System
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		if err := workerEngine.Start(ctx); err != nil {
			log.Fatalf("Failed to start worker engine: %v", err)
		}
		if err := scheduler.Start(ctx); err != nil {
			log.Fatalf("Failed to start scheduler: %v", err)
		}

		// Delivery
		srv := fiber.NewServer(cfg, authService, userService, queueDriver)

		// Graceful Shutdown
		go func() {
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
			<-sigChan
			log.Println("Shutting down gracefully...")
			cancel()
			_ = workerEngine.Stop()
			_ = scheduler.Stop()
			_ = srv.Stop()
		}()

		log.Printf("Starting server on %s:%d", cfg.Server.Host, cfg.Server.Port)
		if err := srv.Start(); err != nil {
			log.Fatalf("Server failed to start: %v", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
}
