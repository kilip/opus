package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

// @title Opus API
// @version 1.0.0
// @description Web-based mission control dashboard for AI coding agents
// @BasePath /api
func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := run(ctx); err != nil {
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	return newRootCmd().ExecuteContext(ctx)
}
