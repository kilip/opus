package config

import (
	"context"
	"fmt"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"
)

// GetWhatsAppStore returns a new whatsmeow sqlstore.Container based on the application configuration.
// It uses a dedicated DSN for WhatsApp to isolate session data from domain data.
func GetWhatsAppStore(cfg *Config) (*sqlstore.Container, error) {
	var driver, dsn string
	switch cfg.Database.Driver {
	case "sqlite":
		driver = "sqlite3"
		// Ensure foreign keys are enabled for SQLite
		dsn = cfg.Database.DSN + "?_fk=1"
	case "postgres":
		driver = "postgres"
		dsn = cfg.Database.DSN
	default:
		return nil, fmt.Errorf("unsupported database driver for whatsapp store: %s", cfg.Database.Driver)
	}

	return sqlstore.New(context.Background(), driver, dsn, waLog.Stdout("WA-Store", "WARN", true))
}
