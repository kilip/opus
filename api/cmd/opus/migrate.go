package main

import (
	"context"
	"log"

	"github.com/kilip/opus/api/internal/config"
	"github.com/spf13/cobra"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Run database migrations",
	Run: func(cmd *cobra.Command, args []string) {
		db := config.GetDatabase()
		if err := db.Schema.Create(context.Background()); err != nil {
			log.Fatalf("failed creating schema resources: %v", err)
		}
		log.Println("Database migration completed successfully.")
	},
}

func init() {
	rootCmd.AddCommand(migrateCmd)
}
