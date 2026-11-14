package main

import (
	"github.com/kilip/opus/internal/app"
	"github.com/spf13/cobra"
)

func newServeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Start HTTP server",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			cfg, err := app.GetConfig(ctx)
			if err != nil {
				return err
			}
			logger := app.GetLogger(cfg)
			fiberApp := app.GetFiber(cfg, logger)
			return app.New(cfg, fiberApp, logger).Serve(ctx)
		},
	}
}
