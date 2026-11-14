package main

import (
	"github.com/spf13/cobra"
)

var Version = "dev"

type rootOptions struct {
	addr      string
	home      string
	env       string
	logLevel  string
	logFormat string
	ompBin    string
}

func newRootCmd() *cobra.Command {
	opts := &rootOptions{}

	cmd := &cobra.Command{
		Use:     "opus",
		Short:   "Opus web-based mission control dashboard for AI coding agents",
		Version: Version,
	}

	cmd.PersistentFlags().StringVarP(&opts.addr, "addr", "a", "", "HTTP listen address (default 0.0.0.0:7878)")
	cmd.PersistentFlags().StringVar(&opts.home, "home", "", "Opus data directory (default ~/.opus)")
	cmd.PersistentFlags().StringVarP(&opts.env, "env", "e", "", "Environment: development or production (default production)")
	cmd.PersistentFlags().StringVar(&opts.logLevel, "log-level", "", "Log level: debug, info, warn, error (default info, debug in development)")
	cmd.PersistentFlags().StringVar(&opts.logFormat, "log-format", "", "Log format: json or text (default json, text in development)")
	cmd.PersistentFlags().StringVar(&opts.ompBin, "omp-bin", "", "Path to omp binary (default omp)")
	cmd.AddCommand(
		newServeCmd(opts),
	)
	return cmd
}
