package main

import (
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Get status of the Opus server",
	Long:  `Get execution status of the server by checking the PID file.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		pid, err := readPID()
		if err != nil {
			cmd.Println("Opus server is not running (no PID file found).")
			return nil
		}
		if !isProcessRunning(pid) {
			cmd.Printf("Opus server is not running (PID file exists with PID %d, but process is dead).\n", pid)
			return nil
		}
		cmd.Printf("Opus server is running with PID %d.\n", pid)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
