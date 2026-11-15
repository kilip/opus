package repo_test

import (
	"context"
	"database/sql"
	"testing"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/kilip/opus/ent"
	"github.com/kilip/opus/ent/enttest"
	"github.com/kilip/opus/internal/auth/domain"
	"github.com/kilip/opus/internal/auth/repo"
	_ "modernc.org/sqlite"
)

func TestEntRepo_RecordAuditLog(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", "file:ent_audit_repo_test?mode=memory&cache=shared&_pragma=foreign_keys(ON)")
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	drv := entsql.OpenDB(dialect.SQLite, db)
	client := enttest.NewClient(t, enttest.WithOptions(ent.Driver(drv)))
	defer func() {
		_ = client.Close()
	}()

	r := repo.New(client)

	// Create user
	u, err := client.User.Create().
		SetEmail("repo_audit@example.com").
		Save(ctx)
	if err != nil {
		t.Fatalf("create user failed: %v", err)
	}

	meta := domain.AuditMeta{
		IP:        "10.0.0.1",
		UserAgent: "test-agent",
	}

	// 1. Record with valid user ID
	if err := r.RecordAuditLog(ctx, &u.ID, domain.AuditEventLoginOTP, meta); err != nil {
		t.Fatalf("record audit log with user failed: %v", err)
	}

	// 2. Record with nil user ID
	if err := r.RecordAuditLog(ctx, nil, domain.AuditEventOTPFailed, meta); err != nil {
		t.Fatalf("record audit log with nil user failed: %v", err)
	}

	// 3. Record with uuid.Nil pointer (should store as nil user)
	nilUUID := uuid.Nil
	if err := r.RecordAuditLog(ctx, &nilUUID, domain.AuditEventLogout, meta); err != nil {
		t.Fatalf("record audit log with uuid.Nil failed: %v", err)
	}

	// Check count and contents in db
	logs, err := client.AuditLog.Query().All(ctx)
	if err != nil {
		t.Fatalf("query audit logs failed: %v", err)
	}
	if len(logs) != 3 {
		t.Fatalf("expected 3 audit logs, got %d", len(logs))
	}
}
