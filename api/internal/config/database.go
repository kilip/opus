// api/internal/config/database.go
package config

import (
	"context"
	"log"
	"sync"

	"github.com/kilip/opus/api/ent"
	_ "github.com/mattn/go-sqlite3"
	_ "github.com/lib/pq"
)

var (
	dbClient *ent.Client
	dbOnce   sync.Once
)

func GetDatabase() *ent.Client {
	dbOnce.Do(func() {
		cfg := GetConfig()
		var err error
		switch cfg.Database.Driver {
		case "sqlite":
			dbClient, err = ent.Open("sqlite3", cfg.Database.DSN+"?_fk=1")
		case "postgres":
			dbClient, err = ent.Open("postgres", cfg.Database.DSN)
		default:
			log.Fatalf("unsupported database driver: %s", cfg.Database.Driver)
		}
		if err != nil {
			log.Fatalf("failed to connect to database: %v", err)
		}
		if err := dbClient.Schema.Create(context.Background()); err != nil {
			log.Fatalf("failed to run migrations: %v", err)
		}
	})
	return dbClient
}
