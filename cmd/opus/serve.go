package main

import (
	"github.com/kilip/opus/internal/app"
	"github.com/kilip/opus/internal/node"
	"github.com/kilip/opus/internal/platform/eventbus"
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
			flushSentry := app.InitSentry(cfg, logger)
			defer flushSentry()

			db, err := app.OpenDB(ctx, cfg.DatabaseURL)
			if err != nil {
				return err
			}
			bus := eventbus.New(eventbus.WithLogger(logger))
			nodeFeature := node.New(db, node.Config{BaseURL: cfg.BaseURL, Bus: bus})
			fiberApp := app.GetFiber(cfg, logger, db, bus, nodeFeature)
			return app.New(cfg, fiberApp, logger, db, bus, nodeFeature).Serve(ctx)
		},
	}
}
