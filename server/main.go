// Package main is the entry point for the Opus server.
package main

import (
	"fmt"
	"os"

	"github.com/kilip/opus/server/internal/delivery/gofiber"
	"github.com/kilip/opus/server/internal/shared/logger"
)

func main() {
	// Simple manual init for bootstrap verification
	log := &logger.NoopLogger{}
	cfg := gofiber.Config{
		Address: ":8080",
	}

	app := gofiber.New(cfg, log)
	fmt.Println("Opus Server starting on", cfg.Address)
	if err := app.Listen(cfg.Address); err != nil {
		fmt.Printf("failed to start server: %v\n", err)
		os.Exit(1)
	}
}
