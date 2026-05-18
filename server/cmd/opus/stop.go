package main

import (
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the Opus server",
	Long:  `Stops the Opus server by sending a SIGTERM signal to the process identified in the PID file.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		pid, err := readPID()
		if err != nil {
			cmd.Println("Opus server is not running (no PID file found).")
			return nil
		}
		if !isProcessRunning(pid) {
			cmd.Printf("Opus server is not running (PID %d is dead). Cleaning up PID file.\n", pid)
			_ = removePID()
			return nil
		}

		process, err := os.FindProcess(pid)
		if err != nil {
			return fmt.Errorf("failed to find process %d: %w", pid, err)
		}

		cmd.Printf("Stopping Opus server (PID %d)...\n", pid)
		if err := process.Signal(syscall.SIGTERM); err != nil {
			return fmt.Errorf("failed to send SIGTERM to process %d: %w", pid, err)
		}

		// Wait up to 5 seconds for the process to exit gracefully
		for i := 0; i < 50; i++ {
			if !isProcessRunning(pid) {
				cmd.Println("Opus server stopped successfully.")
				_ = removePID()
				return nil
			}
			time.Sleep(100 * time.Millisecond)
		}

		// Force kill if it fails to exit gracefully
		cmd.Println("Server did not stop within 5 seconds. Force killing...")
		if err := process.Signal(syscall.SIGKILL); err != nil {
			return fmt.Errorf("failed to send SIGKILL to process %d: %w", pid, err)
		}
		_ = removePID()
		cmd.Println("Opus server force killed successfully.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)
}
