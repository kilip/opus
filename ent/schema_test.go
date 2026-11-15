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

func TestSchema_Account(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", "file:ent_account?mode=memory&cache=shared&_pragma=foreign_keys(ON)")
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
		SetEmail("oauth_user@example.com").
		Save(ctx)
	if err != nil {
		t.Fatalf("create user failed: %v", err)
	}

	// 2. Create an account for that user
	acc, err := client.Account.Create().
		SetUserID(u.ID).
		SetProvider("github").
		SetProviderUserID("gh_9999").
		SetEmail("oauth_user@example.com").
		SetEmailVerified(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("create account failed: %v", err)
	}

	// 3. Read it back
	gotAcc, err := client.Account.Get(ctx, acc.ID)
	if err != nil {
		t.Fatalf("get account failed: %v", err)
	}
	if gotAcc.Provider != "github" {
		t.Errorf("got provider %q, want %q", gotAcc.Provider, "github")
	}
	if gotAcc.ProviderUserID != "gh_9999" {
		t.Errorf("got provider_user_id %q, want %q", gotAcc.ProviderUserID, "gh_9999")
	}
	if gotAcc.Email != "oauth_user@example.com" {
		t.Errorf("got email %q, want %q", gotAcc.Email, "oauth_user@example.com")
	}
	if !gotAcc.EmailVerified {
		t.Errorf("got email_verified false, want true")
	}
	if gotAcc.UserID != u.ID {
		t.Errorf("got user_id %v, want %v", gotAcc.UserID, u.ID)
	}

	// 4. Test unique constraint on (provider, provider_user_id)
	u2, err := client.User.Create().
		SetEmail("oauth_user2@example.com").
		Save(ctx)
	if err != nil {
		t.Fatalf("create user 2 failed: %v", err)
	}
	_, err = client.Account.Create().
		SetUserID(u2.ID).
		SetProvider("github").
		SetProviderUserID("gh_9999").
		SetEmail("oauth_user2@example.com").
		Save(ctx)
	if err == nil {
		t.Fatal("expected error due to unique (provider, provider_user_id) index violation, got nil")
	}

	// 5. Different provider with same provider_user_id should succeed
	_, err = client.Account.Create().
		SetUserID(u2.ID).
		SetProvider("google").
		SetProviderUserID("gh_9999").
		SetEmail("oauth_user2@example.com").
		Save(ctx)
	if err != nil {
		t.Fatalf("create account with different provider failed: %v", err)
	}

	// 6. Cascade delete: deleting user u should delete its account acc
	if err := client.User.DeleteOneID(u.ID).Exec(ctx); err != nil {
		t.Fatalf("delete user failed: %v", err)
	}
	stillAcc, err := client.Account.Get(ctx, acc.ID)
	if err == nil {
		t.Fatalf("expected account to be deleted via cascade, but found: %+v", stillAcc)
	}
}
