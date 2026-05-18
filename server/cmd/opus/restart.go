package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var restartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart the Opus server",
	Long:  `Restarts the running Opus server by stopping it first and then starting it again.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		pid, err := readPID()
		if err == nil && isProcessRunning(pid) {
			cmd.Printf("Restarting Opus server (stopping PID %d first)...\n", pid)
			if err := stopCmd.RunE(cmd, args); err != nil {
				return fmt.Errorf("restart failed during stop: %w", err)
			}
		} else {
			cmd.Println("Opus server is not running. Starting fresh...")
		}

		return runStart(cmd, args)
	},
}

func init() {
	// Share the daemon flag with the start command
	restartCmd.Flags().BoolVarP(&daemonBool, "daemon", "d", false, "Run Opus server in background")
	rootCmd.AddCommand(restartCmd)
}
