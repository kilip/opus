// api/internal/config/database.go
package config

import (
	"context"
	"database/sql"
	"log"
	"path/filepath"
	"sync"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/kilip/opus/api/ent"
	_ "github.com/lib/pq"
	_ "modernc.org/sqlite"
)

var (
	dbClient *ent.Client
	dbOnce   sync.Once
)

func GetDatabase() *ent.Client {
	dbOnce.Do(func() {
		cfg := GetConfig()
		logger := GetLogger()

		absDSN, _ := filepath.Abs(cfg.Database.DSN)
		logger.Info("Connecting to database", "driver", cfg.Database.Driver, "dsn", cfg.Database.DSN, "abs_dsn", absDSN)
		var err error
		switch cfg.Database.Driver {
		case "sqlite":
			// Kembalikan ke kondisi semula tanpa WAL mode
			db, err := sql.Open("sqlite", cfg.Database.DSN+"?_pragma=foreign_keys(1)")
			if err != nil {
				log.Fatalf("failed to open sqlite database: %v", err)
			}
			drv := entsql.OpenDB(dialect.SQLite, db)
			dbClient = ent.NewClient(ent.Driver(drv))
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
