package ent_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/kilip/opus/ent"
	"github.com/kilip/opus/ent/enttest"
	_ "modernc.org/sqlite"
)

func TestSchema_SessionAndOTP(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", "file:ent?mode=memory&cache=shared&_pragma=foreign_keys(ON)")
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	drv := entsql.OpenDB(dialect.SQLite, db)
	client := enttest.NewClient(t, enttest.WithOptions(ent.Driver(drv)))
	defer func() {
		_ = client.Close()
	}()

	// 1. Create a user
	u, err := client.User.Create().
		SetEmail("tester@example.com").
		Save(ctx)
	if err != nil {
		t.Fatalf("create user failed: %v", err)
	}

	// 2. Create a session for that user
	expiresAt := time.Now().Add(24 * time.Hour)
	tokenHash := "hash123456"
	sess, err := client.Session.Create().
		SetUserID(u.ID).
		SetTokenHash(tokenHash).
		SetExpiresAt(expiresAt).
		Save(ctx)
	if err != nil {
		t.Fatalf("create session failed: %v", err)
	}

	// 3. Create an OTP
	otpExpiresAt := time.Now().Add(10 * time.Minute)
	otp, err := client.OTP.Create().
		SetEmail("tester@example.com").
		SetCodeHash("otphash123").
		SetExpiresAt(otpExpiresAt).
		Save(ctx)
	if err != nil {
		t.Fatalf("create otp failed: %v", err)
	}

	// 4. Read them back
	// Read user
	gotUser, err := client.User.Get(ctx, u.ID)
	if err != nil {
		t.Fatalf("get user failed: %v", err)
	}
	if gotUser.Email != "tester@example.com" {
		t.Errorf("got user email %q, want %q", gotUser.Email, "tester@example.com")
	}

	// Read session
	gotSess, err := client.Session.Get(ctx, sess.ID)
	if err != nil {
		t.Fatalf("get session failed: %v", err)
	}
	if gotSess.TokenHash != tokenHash {
		t.Errorf("got token hash %q, want %q", gotSess.TokenHash, tokenHash)
	}
	if gotSess.UserID != u.ID {
		t.Errorf("got session user_id %v, want %v", gotSess.UserID, u.ID)
	}

	// Read OTP
	gotOTP, err := client.OTP.Get(ctx, otp.ID)
	if err != nil {
		t.Fatalf("get otp failed: %v", err)
	}
	if gotOTP.Email != "tester@example.com" {
		t.Errorf("got otp email %q, want %q", gotOTP.Email, "tester@example.com")
	}
	if gotOTP.CodeHash != "otphash123" {
		t.Errorf("got otp code_hash %q, want %q", gotOTP.CodeHash, "otphash123")
	}
	if gotOTP.Attempts != 0 {
		t.Errorf("got otp attempts %d, want 0", gotOTP.Attempts)
	}

	// Verify cascade delete: deleting user should delete session
	if err := client.User.DeleteOneID(u.ID).Exec(ctx); err != nil {
		t.Fatalf("delete user failed: %v", err)
	}
	stillSess, err := client.Session.Get(ctx, sess.ID)
	if err == nil {
		t.Fatalf("expected session to be deleted via cascade, but found: %+v", stillSess)
	}
}
