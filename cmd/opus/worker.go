package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/kilip/opus/internal/worker"
	"github.com/spf13/cobra"
)

func newWorkerCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "worker",
		Short: "Start worker node from saved configuration",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runWorker(cmd.Context())
		},
	}
}

func runWorker(ctx context.Context) error {
	configPath, err := defaultNodeConfigPath()
	if err != nil {
		return err
	}

	cfg, err := readNodeConfig(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) || os.IsNotExist(err) {
			return fmt.Errorf("node configuration not found at %s: run 'opus join <url>' first", configPath)
		}
		return fmt.Errorf("load node configuration: %w", err)
	}

	w := worker.New(worker.Config{
		MasterURL:  cfg.MasterURL,
		NodeID:     cfg.NodeID,
		Credential: cfg.Credential,
		Logger:     slog.Default(),
	})
	return w.Run(ctx)
}
