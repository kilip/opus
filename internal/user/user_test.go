package user_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/kilip/opus/internal/app"
	"github.com/kilip/opus/internal/user"
)

func setupTest(t *testing.T) (*user.User, context.Context) {
	t.Helper()
	ctx := context.Background()
	dsn := "file:ent?mode=memory&cache=shared&_pragma=foreign_keys(ON)"

	client, err := app.OpenDB(ctx, dsn)
	if err != nil {
		t.Fatalf("OpenDB failed: %v", err)
	}

	// Clean up database tables before test
	if _, err := client.User.Delete().Exec(ctx); err != nil {
		t.Fatalf("clean users table: %v", err)
	}

	t.Cleanup(func() {
		_, _ = client.User.Delete().Exec(ctx)
		_ = client.Close()
	})

	return user.New(client), ctx
}

func TestUser_Create(t *testing.T) {
	u, ctx := setupTest(t)

	got, err := u.Create(ctx, "  Alice@Example.COM  ")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if got.ID == uuid.Nil {
		t.Error("expected non-nil UUID")
	}
	if got.Email != "alice@example.com" {
		t.Errorf("got email %q, want %q", got.Email, "alice@example.com")
	}
	if got.CreatedAt.IsZero() {
		t.Error("expected non-zero CreatedAt")
	}
}

func TestUser_DuplicateEmailReturnsErrEmailTaken(t *testing.T) {
	u, ctx := setupTest(t)

	if _, err := u.Create(ctx, "bob@example.com"); err != nil {
		t.Fatalf("first Create failed: %v", err)
	}

	_, err := u.Create(ctx, "  BOB@example.com  ")
	if err == nil {
		t.Fatal("expected error on duplicate email, got nil")
	}
	if !errors.Is(err, user.ErrEmailTaken) {
		t.Errorf("got error %v, want %v", err, user.ErrEmailTaken)
	}
}

func TestUser_ByEmail_Miss(t *testing.T) {
	u, ctx := setupTest(t)

	_, err := u.ByEmail(ctx, "unknown@example.com")
	if err == nil {
		t.Fatal("expected error on missing email, got nil")
	}
	if !errors.Is(err, user.ErrNotFound) {
		t.Errorf("got error %v, want %v", err, user.ErrNotFound)
	}
}

func TestUser_ByEmail_Hit(t *testing.T) {
	u, ctx := setupTest(t)

	created, err := u.Create(ctx, "carol@example.com")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := u.ByEmail(ctx, "  CAROL@example.com  ")
	if err != nil {
		t.Fatalf("ByEmail failed: %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("got ID %v, want %v", got.ID, created.ID)
	}
	if got.Email != "carol@example.com" {
		t.Errorf("got email %q, want %q", got.Email, "carol@example.com")
	}
}

func TestUser_Count(t *testing.T) {
	u, ctx := setupTest(t)

	count, err := u.Count(ctx)
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if count != 0 {
		t.Errorf("got initial count %d, want 0", count)
	}

	if _, err := u.Create(ctx, "user1@example.com"); err != nil {
		t.Fatalf("Create user1 failed: %v", err)
	}
	if _, err := u.Create(ctx, "user2@example.com"); err != nil {
		t.Fatalf("Create user2 failed: %v", err)
	}

	count, err = u.Count(ctx)
	if err != nil {
		t.Fatalf("Count after creates failed: %v", err)
	}
	if count != 2 {
		t.Errorf("got count %d, want 2", count)
	}
}
