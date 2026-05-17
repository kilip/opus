package entgo

import (
	"context"
	"database/sql"
	"fmt"

	entsql "entgo.io/ent/dialect/sql"
	_ "github.com/lib/pq"
	_ "modernc.org/sqlite"

	"github.com/kilip/opus/server/ent"
	"github.com/kilip/opus/server/internal/config"
)

// NewClient creates a new Ent database client using the provided database configuration.
func NewClient(cfg config.DatabaseConfig) (*ent.Client, error) {
	var dialectName string
	var driverName string

	switch cfg.Driver {
	case "sqlite3", "sqlite":
		driverName = "sqlite"
		dialectName = "sqlite3"
	case "postgres":
		driverName = "postgres"
		dialectName = "postgres"
	default:
		return nil, fmt.Errorf("entgo: unsupported driver: %s", cfg.Driver)
	}

	db, err := sql.Open(driverName, cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("entgo: open database: %w", err)
	}

	if driverName == "sqlite" {
		if _, err := db.Exec("PRAGMA journal_mode=WAL;"); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("entgo: enable WAL mode: %w", err)
		}
		if _, err := db.Exec("PRAGMA foreign_keys=ON;"); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("entgo: enable foreign keys: %w", err)
		}
	}

	drv := entsql.OpenDB(dialectName, db)
	return ent.NewClient(ent.Driver(drv)), nil
}

// AutoMigrate runs the auto-migration for the database schema.
func AutoMigrate(client *ent.Client, ctx context.Context) error {
	return client.Schema.Create(ctx)
}
