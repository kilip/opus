package main

import (
	"fmt"
	"os/exec"

	"github.com/spf13/cobra"
)

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Tail API server logs",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Tailing logs from journalctl (if running as service) or log file...")
		// Assuming we log to a file in production, or just use tail
		c := exec.Command("tail", "-f", "/tmp/opus.log") // placeholder
		c.Stdout = cmd.OutOrStdout()
		c.Stderr = cmd.ErrOrStderr()
		_ = c.Run()
	},
}

func init() {
	rootCmd.AddCommand(logsCmd)
}
