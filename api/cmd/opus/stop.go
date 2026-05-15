package main

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the API server",
	Run: func(cmd *cobra.Command, args []string) {
		pidFile := filepath.Join(os.Getenv("HOME"), ".opus", "opus.pid")
		data, err := os.ReadFile(pidFile)
		if err != nil {
			fmt.Println("Server is not running (no PID file found)")
			return
		}

		var pid int
		if _, err := fmt.Sscanf(string(data), "%d", &pid); err != nil {
			fmt.Printf("Failed to parse PID file: %v\n", err)
			return
		}

		process, err := os.FindProcess(pid)
		if err != nil {
			fmt.Printf("Failed to find process %d: %v\n", pid, err)
			return
		}

		err = process.Signal(syscall.SIGTERM)
		if err != nil {
			fmt.Printf("Failed to stop process %d: %v\n", pid, err)
			return
		}

		_ = os.Remove(pidFile)
		fmt.Printf("Stopped process %d\n", pid)
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)
}
