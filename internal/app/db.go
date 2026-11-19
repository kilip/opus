package app

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/kilip/opus/ent"
	"github.com/kilip/opus/ent/meta"
	_ "modernc.org/sqlite"
)

const metaKeyAppVersion = "app_version"

// OpenDB opens an Ent client matching APP_DATABASE_URL scheme.
// ponytail: sqlite-only driver dispatch; add postgres dialect when switching to remote db
func OpenDB(ctx context.Context, databaseURL string) (*ent.Client, error) {
	return OpenDBWithVersion(ctx, databaseURL, "", false)
}

// OpenDBWithVersion opens the database, configures SQLite pragmas/pooling, runs schema
// auto-migration, and enforces the stored schema version guard when binaryVersion is non-empty.
func OpenDBWithVersion(ctx context.Context, databaseURL, binaryVersion string, allowDowngrade bool) (*ent.Client, error) {
	switch {
	case strings.HasPrefix(databaseURL, "file:"), strings.HasPrefix(databaseURL, "sqlite:"):
		dsn, err := normalizeSQLiteDSN(databaseURL)
		if err != nil {
			return nil, fmt.Errorf("invalid sqlite dsn: %w", err)
		}
		db, err := sql.Open("sqlite", dsn)
		if err != nil {
			return nil, fmt.Errorf("open sqlite db: %w", err)
		}
		db.SetMaxOpenConns(8)
		db.SetMaxIdleConns(4)

		drv := entsql.OpenDB(dialect.SQLite, db)
		client := ent.NewClient(ent.Driver(drv))

		// ponytail: auto-migration on startup; replace with versioned migrations (atlas/golang-migrate) when schema changes need rollbacks
		if err := client.Schema.Create(ctx); err != nil {
			_ = client.Close()
			return nil, fmt.Errorf("auto-migrate schema: %w", err)
		}
		if binaryVersion != "" {
			if err := checkAndStoreVersion(ctx, client, binaryVersion, allowDowngrade); err != nil {
				_ = client.Close()
				return nil, err
			}
		}
		return client, nil
	default:
		return nil, fmt.Errorf("unsupported database scheme in %q: only sqlite is supported", databaseURL)
	}
}

func normalizeSQLiteDSN(databaseURL string) (string, error) {
	dsn := strings.TrimPrefix(databaseURL, "sqlite:")
	base, rawQuery, _ := strings.Cut(dsn, "?")
	q, err := url.ParseQuery(rawQuery)
	if err != nil {
		return "", err
	}

	isMemory := strings.Contains(strings.ToLower(base), ":memory:") ||
		strings.EqualFold(q.Get("mode"), "memory")

	existingPragmas := make(map[string]struct{})
	for _, p := range q["_pragma"] {
		name := strings.ToLower(strings.TrimSpace(p))
		if idx := strings.IndexAny(name, "(="); idx > 0 {
			name = strings.TrimSpace(name[:idx])
		}
		existingPragmas[name] = struct{}{}
	}
	if q.Has("_fk") {
		existingPragmas["foreign_keys"] = struct{}{}
	}

	addPragma := func(name, expr string) {
		if _, ok := existingPragmas[name]; !ok {
			q.Add("_pragma", expr)
		}
	}

	if !isMemory {
		addPragma("journal_mode", "journal_mode(WAL)")
		addPragma("synchronous", "synchronous(NORMAL)")
	}
	addPragma("busy_timeout", "busy_timeout(5000)")
	addPragma("foreign_keys", "foreign_keys(ON)")

	if !isMemory && !q.Has("_txlock") {
		q.Set("_txlock", "immediate")
	}

	encoded := q.Encode()
	if encoded == "" {
		return base, nil
	}
	return base + "?" + encoded, nil
}

func checkAndStoreVersion(ctx context.Context, client *ent.Client, binaryVersion string, allowDowngrade bool) error {
	row, err := client.Meta.Query().Where(meta.KeyEQ(metaKeyAppVersion)).Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return client.Meta.Create().
				SetKey(metaKeyAppVersion).
				SetValue(binaryVersion).
				Exec(ctx)
		}
		return fmt.Errorf("read stored schema version: %w", err)
	}

	if cmp, ok := compareSemver(row.Value, binaryVersion); ok && cmp > 0 && !allowDowngrade {
		return fmt.Errorf(
			"stored schema version %s is newer than binary version %s (set APP_ALLOW_DOWNGRADE=true to override)",
			row.Value,
			binaryVersion,
		)
	}

	if row.Value != binaryVersion {
		return client.Meta.UpdateOneID(row.ID).SetValue(binaryVersion).Exec(ctx)
	}
	return nil
}

// compareSemver compares two v?MAJOR.MINOR.PATCH strings.
// Returns (cmp, true) when both parse as numeric semver triples, or (0, false) otherwise.
func compareSemver(a, b string) (int, bool) {
	va, okA := parseSemver(a)
	vb, okB := parseSemver(b)
	if !okA || !okB {
		return 0, false
	}
	for i := range 3 {
		if va[i] < vb[i] {
			return -1, true
		}
		if va[i] > vb[i] {
			return 1, true
		}
	}
	return 0, true
}

func parseSemver(raw string) ([3]int, bool) {
	var out [3]int
	s := strings.TrimPrefix(strings.TrimSpace(raw), "v")
	if idx := strings.IndexAny(s, "-+"); idx >= 0 {
		s = s[:idx]
	}
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return out, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return out, false
		}
		out[i] = n
	}
	return out, true
}
