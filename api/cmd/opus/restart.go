package main

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

var restartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart the API server",
	Run: func(cmd *cobra.Command, args []string) {
		stopCmd.Run(cmd, args)
		time.Sleep(2 * time.Second)
		startCmd.Run(cmd, args)
		fmt.Println("Server restarted")
	},
}

func init() {
	rootCmd.AddCommand(restartCmd)
}
