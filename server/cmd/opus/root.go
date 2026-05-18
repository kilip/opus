package main

import (
	"github.com/spf13/cobra"
)

// defaultRunFunc represents the default action when no subcommand is provided.
// It will be populated by the start command implementation to maintain backward compatibility.
var defaultRunFunc func(cmd *cobra.Command, args []string) error

var rootCmd = &cobra.Command{
	Use:   "opus",
	Short: "Opus CLI",
	Long:  `Opus is an autonomous AI assistant that unifies knowledge, workflows, and agents.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if defaultRunFunc != nil {
			return defaultRunFunc(cmd, args)
		}
		// If no default run function is registered, show help
		return cmd.Help()
	},
}

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}

// RootCmd returns the root command.
func RootCmd() *cobra.Command {
	return rootCmd
}
