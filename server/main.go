// Package main is the entry point for the Opus server.
package main

import (
	"context"
	"fmt"

	"github.com/kilip/opus/server/internal/config"
	"github.com/kilip/opus/server/internal/container"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(fmt.Errorf("main: failed to load config: %w", err))
	}

	container.Bootstrap(*cfg)

	ctx := context.Background()
	log := container.GetLogger()

	log.Info("starting opus queue worker")
	if err := container.GetQueue().Start(ctx); err != nil {
		panic(fmt.Errorf("main: failed to start queue: %w", err))
	}

	address := cfg.Server.Address
	if address == "" {
		address = ":8080" // Fallback
	}

	log.Info(fmt.Sprintf("starting opus server on %s", address))
	if err := container.GetFiber().Listen(address); err != nil {
		panic(fmt.Errorf("main: failed to start server: %w", err))
	}
}
