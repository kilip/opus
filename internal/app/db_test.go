package app

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestOpenDB_PragmasAndPool(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	dsn, err := normalizeSQLiteDSN("file:" + dbPath)
	if err != nil {
		t.Fatalf("normalizeSQLiteDSN: %v", err)
	}

	rawDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer func() { _ = rawDB.Close() }()

	var journalMode string
	if err := rawDB.QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&journalMode); err != nil {
		t.Fatalf("PRAGMA journal_mode: %v", err)
	}
	if !strings.EqualFold(journalMode, "wal") {
		t.Fatalf("journal_mode = %q, want wal", journalMode)
	}

	var syncMode int
	if err := rawDB.QueryRowContext(ctx, "PRAGMA synchronous").Scan(&syncMode); err != nil {
		t.Fatalf("PRAGMA synchronous: %v", err)
	}
	if syncMode != 1 {
		t.Fatalf("synchronous = %d, want 1 (NORMAL)", syncMode)
	}

	var busyTimeout int
	if err := rawDB.QueryRowContext(ctx, "PRAGMA busy_timeout").Scan(&busyTimeout); err != nil {
		t.Fatalf("PRAGMA busy_timeout: %v", err)
	}
	if busyTimeout != 5000 {
		t.Fatalf("busy_timeout = %d, want 5000", busyTimeout)
	}

	var foreignKeys int
	if err := rawDB.QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&foreignKeys); err != nil {
		t.Fatalf("PRAGMA foreign_keys: %v", err)
	}
	if foreignKeys != 1 {
		t.Fatalf("foreign_keys = %d, want 1", foreignKeys)
	}

	memDSN, err := normalizeSQLiteDSN("file:memdb1?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("normalizeSQLiteDSN memory: %v", err)
	}
	if strings.Contains(strings.ToLower(memDSN), "wal") {
		t.Fatalf("expected memory DSN to skip WAL, got %q", memDSN)
	}

	customDSN, err := normalizeSQLiteDSN("file:" + dbPath + "?_pragma=busy_timeout(9000)")
	if err != nil {
		t.Fatalf("normalizeSQLiteDSN custom: %v", err)
	}
	if !strings.Contains(customDSN, "busy_timeout%289000%29") && !strings.Contains(customDSN, "busy_timeout(9000)") {
		t.Fatalf("expected user-set pragma preserved, got %q", customDSN)
	}
	if strings.Contains(customDSN, "5000") {
		t.Fatalf("expected default busy_timeout not added when user set it, got %q", customDSN)
	}
}

func TestOpenDB_VersionGuard(t *testing.T) {
	ctx := context.Background()
	dbURL := "file:" + filepath.Join(t.TempDir(), "version.db")

	c1, err := OpenDBWithVersion(ctx, dbURL, "0.1.0", false)
	if err != nil {
		t.Fatalf("initial OpenDBWithVersion 0.1.0: %v", err)
	}
	_ = c1.Close()

	// Upgrade 0.1.0 -> 0.2.0 succeeds.
	c2, err := OpenDBWithVersion(ctx, dbURL, "0.2.0", false)
	if err != nil {
		t.Fatalf("upgrade OpenDBWithVersion 0.2.0: %v", err)
	}
	_ = c2.Close()

	// Downgrade 0.2.0 -> 0.1.0 is refused by default.
	if _, err := OpenDBWithVersion(ctx, dbURL, "0.1.0", false); err == nil {
		t.Fatal("expected downgrade 0.2.0 -> 0.1.0 to fail")
	}

	// Downgrade with APP_ALLOW_DOWNGRADE=true succeeds.
	c3, err := OpenDBWithVersion(ctx, dbURL, "0.1.0", true)
	if err != nil {
		t.Fatalf("downgrade with allowDowngrade=true failed: %v", err)
	}
	_ = c3.Close()
}

func TestOpenDB_ConcurrentStress(t *testing.T) {
	ctx := context.Background()
	dbURL := "file:" + filepath.Join(t.TempDir(), "stress.db")
	client, err := OpenDBWithVersion(ctx, dbURL, "0.1.0", false)
	if err != nil {
		t.Fatalf("OpenDBWithVersion: %v", err)
	}
	defer func() { _ = client.Close() }()

	deadline := time.Now().Add(2 * time.Second)
	var wg sync.WaitGroup
	errs := make(chan error, 50)

	for i := range 50 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			n := 0
			for time.Now().Before(deadline) {
				key := fmt.Sprintf("stress-%d-%d", id, n)
				if err := client.Meta.Create().SetKey(key).SetValue("ok").Exec(ctx); err != nil {
					errs <- err
					return
				}
				if _, err := client.Meta.Query().Limit(5).Count(ctx); err != nil {
					errs <- err
					return
				}
				n++
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatalf("concurrent read/write error: %v", err)
	}
}
