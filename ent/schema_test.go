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

func TestSchema_AuditLog(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", "file:ent_audit_log?mode=memory&cache=shared&_pragma=foreign_keys(ON)")
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
		SetEmail("audit_user@example.com").
		Save(ctx)
	if err != nil {
		t.Fatalf("create user failed: %v", err)
	}

	// 2. Create audit log with user
	al1, err := client.AuditLog.Create().
		SetUserID(u.ID).
		SetEvent("login_otp").
		SetIP("127.0.0.1").
		SetUserAgent("Mozilla/5.0").
		Save(ctx)
	if err != nil {
		t.Fatalf("create audit log with user failed: %v", err)
	}

	// 3. Create audit log without user (user_id is nil)
	al2, err := client.AuditLog.Create().
		SetEvent("otp_failed").
		SetIP("192.168.1.1").
		SetUserAgent("curl/7.68.0").
		Save(ctx)
	if err != nil {
		t.Fatalf("create audit log without user failed: %v", err)
	}

	// 4. Read back and verify
	got1, err := client.AuditLog.Get(ctx, al1.ID)
	if err != nil {
		t.Fatalf("get audit log 1 failed: %v", err)
	}
	if got1.Event != "login_otp" || got1.UserID == nil || *got1.UserID != u.ID || got1.IP != "127.0.0.1" || got1.UserAgent != "Mozilla/5.0" {
		t.Errorf("unexpected got1: %+v", got1)
	}

	got2, err := client.AuditLog.Get(ctx, al2.ID)
	if err != nil {
		t.Fatalf("get audit log 2 failed: %v", err)
	}
	if got2.Event != "otp_failed" || got2.UserID != nil || got2.IP != "192.168.1.1" || got2.UserAgent != "curl/7.68.0" {
		t.Errorf("unexpected got2: %+v", got2)
	}

	// 5. Delete user: audit log 1 should have user_id set to null (SetNull) and still exist
	if err := client.User.DeleteOneID(u.ID).Exec(ctx); err != nil {
		t.Fatalf("delete user failed: %v", err)
	}
	got1AfterDelete, err := client.AuditLog.Get(ctx, al1.ID)
	if err != nil {
		t.Fatalf("expected audit log 1 to still exist after user deletion: %v", err)
	}
	if got1AfterDelete.UserID != nil {
		t.Errorf("expected user_id to be nil after user deletion, got: %v", got1AfterDelete.UserID)
	}
}

func TestSchema_UUIDv7(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", "file:ent_uuidv7?mode=memory&cache=shared&_pragma=foreign_keys(ON)")
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	drv := entsql.OpenDB(dialect.SQLite, db)
	client := enttest.NewClient(t, enttest.WithOptions(ent.Driver(drv)))
	defer func() {
		_ = client.Close()
	}()

	// 1. User
	u, err := client.User.Create().
		SetEmail("uuidv7_user@example.com").
		Save(ctx)
	if err != nil {
		t.Fatalf("create user failed: %v", err)
	}
	if v := u.ID.Version(); v != 7 {
		t.Errorf("User ID version = %d, want 7", v)
	}

	// 2. Session
	sess, err := client.Session.Create().
		SetUserID(u.ID).
		SetTokenHash("token_hash_uuidv7").
		SetExpiresAt(time.Now().Add(time.Hour)).
		Save(ctx)
	if err != nil {
		t.Fatalf("create session failed: %v", err)
	}
	if v := sess.ID.Version(); v != 7 {
		t.Errorf("Session ID version = %d, want 7", v)
	}

	// 3. Account
	acc, err := client.Account.Create().
		SetUserID(u.ID).
		SetProvider("github").
		SetProviderUserID("uuidv7_gh").
		SetEmail("uuidv7_user@example.com").
		SetEmailVerified(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("create account failed: %v", err)
	}
	if v := acc.ID.Version(); v != 7 {
		t.Errorf("Account ID version = %d, want 7", v)
	}

	// 4. OTP
	otp, err := client.OTP.Create().
		SetEmail("uuidv7_user@example.com").
		SetCodeHash("code_hash_uuidv7").
		SetExpiresAt(time.Now().Add(10 * time.Minute)).
		Save(ctx)
	if err != nil {
		t.Fatalf("create otp failed: %v", err)
	}
	if v := otp.ID.Version(); v != 7 {
		t.Errorf("OTP ID version = %d, want 7", v)
	}

	// 5. AuditLog
	al, err := client.AuditLog.Create().
		SetEvent("uuidv7_test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create audit log failed: %v", err)
	}
	if v := al.ID.Version(); v != 7 {
		t.Errorf("AuditLog ID version = %d, want 7", v)
	}
}
