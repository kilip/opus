package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

var Version = "0.1.0"

func newRootCmd() *cobra.Command {
	serveCmd := newServeCmd()
	rootCmd := &cobra.Command{
		Use:          "opus",
		Short:        "Opus mission control dashboard for AI coding agents",
		SilenceUsage: true,
		Version:      Version,
		RunE:         serveCmd.RunE,
	}
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(newJoinCmd())
	rootCmd.AddCommand(newWorkerCmd())
	rootCmd.AddCommand(newServiceCmd())
	rootCmd.AddCommand(newSelfUpdateCmd())
	rootCmd.AddCommand(newUninstallCmd())
	return rootCmd
}

// @title Opus API
// @version 0.1.0
// @description Opus API specification.
// @BasePath /api
func main() {
	cleanupOldBinary()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := newRootCmd().ExecuteContext(ctx); err != nil {
		var exitErr exitCodeError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.code)
		}
		os.Exit(1)
	}
}
