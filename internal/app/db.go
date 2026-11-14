package app

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/kilip/opus/ent"
	_ "modernc.org/sqlite"
)

// OpenDB opens an Ent client matching DATABASE_URL scheme.
// ponytail: sqlite-only driver dispatch; add postgres dialect when switching to remote db
func OpenDB(ctx context.Context, databaseURL string) (*ent.Client, error) {
	switch {
	case strings.HasPrefix(databaseURL, "file:"), strings.HasPrefix(databaseURL, "sqlite:"):
		dsn := strings.TrimPrefix(databaseURL, "sqlite:")
		db, err := sql.Open("sqlite", dsn)
		if err != nil {
			return nil, fmt.Errorf("open sqlite db: %w", err)
		}

		drv := entsql.OpenDB(dialect.SQLite, db)
		client := ent.NewClient(ent.Driver(drv))

		// ponytail: auto-migration on startup; replace with versioned migrations (atlas/golang-migrate) when schema changes need rollbacks
		if err := client.Schema.Create(ctx); err != nil {
			_ = client.Close()
			return nil, fmt.Errorf("auto-migrate schema: %w", err)
		}
		return client, nil
	default:
		return nil, fmt.Errorf("unsupported database scheme in %q: only sqlite is supported", databaseURL)
	}
}
