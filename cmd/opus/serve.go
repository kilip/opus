package main

import (
	"github.com/kilip/opus/internal/app"
	"github.com/spf13/cobra"
)

func newServeCmd(rootOpts *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Run the Opus HTTP server in the foreground",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := app.Config{}
			app := app.New(cfg)
			return app.Serve(cmd.Context())
		},
	}
}
