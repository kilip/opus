package app_test

import (
	"context"
	"testing"

	"github.com/kilip/opus/internal/app"
)

func TestOpenDB_SQLiteInMemory(t *testing.T) {
	ctx := context.Background()
	dsn := "file:ent?mode=memory&cache=shared&_pragma=foreign_keys(ON)"

	client, err := app.OpenDB(ctx, dsn)
	if err != nil {
		t.Fatalf("OpenDB failed: %v", err)
	}
	defer func() {
		_ = client.Close()
	}()
	email := "test@example.com"
	u, err := client.User.Create().
		SetEmail(email).
		Save(ctx)
	if err != nil {
		t.Fatalf("create user failed: %v", err)
	}

	got, err := client.User.Get(ctx, u.ID)
	if err != nil {
		t.Fatalf("get user failed: %v", err)
	}

	if got.Email != email {
		t.Errorf("got email %q, want %q", got.Email, email)
	}
	if got.CreatedAt.IsZero() {
		t.Error("expected non-zero CreatedAt")
	}
}

func TestOpenDB_UnsupportedScheme(t *testing.T) {
	ctx := context.Background()
	_, err := app.OpenDB(ctx, "postgres://user:pass@localhost:5432/db")
	if err == nil {
		t.Fatal("expected error for unsupported scheme postgres://, got nil")
	}
}
