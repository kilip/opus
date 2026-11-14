package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

var Version = "0.1.0"

// @title Opus API
// @version 0.1.0
// @description Opus API specification.
// @BasePath /api
func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	rootCmd := &cobra.Command{
		Use:          "opus",
		Short:        "Opus mission control dashboard for AI coding agents",
		SilenceUsage: true,
		Version:      Version,
	}
	rootCmd.AddCommand(newServeCmd())

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}
