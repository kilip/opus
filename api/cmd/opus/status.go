package main

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check API server status",
	Run: func(cmd *cobra.Command, args []string) {
		pidFile := filepath.Join(os.Getenv("HOME"), ".opus", "opus.pid")
		data, err := os.ReadFile(pidFile)
		if err != nil {
			fmt.Println("Status: STOPPED (no PID file)")
			return
		}

		var pid int
		if _, err := fmt.Sscanf(string(data), "%d", &pid); err != nil {
			fmt.Printf("Status: UNKNOWN (invalid PID file content: %v)\n", err)
			return
		}

		process, err := os.FindProcess(pid)
		if err != nil {
			fmt.Printf("Status: UNKNOWN (PID %d not found)\n", pid)
			return
		}

		// On Unix, FindProcess always succeeds. Need to signal 0 to check existence.
		err = process.Signal(syscall.Signal(0))
		if err != nil {
			fmt.Println("Status: STOPPED (process dead)")
			return
		}

		fmt.Printf("Status: RUNNING (PID %d)\n", pid)
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
