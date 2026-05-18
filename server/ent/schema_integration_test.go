//go:build integration

package ent_test

import (
	"context"
	"testing"
	"time"

	"github.com/kilip/opus/server/ent/enttest"
	_ "github.com/mattn/go-sqlite3"
)

func TestSchemaDefinitions(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	now := time.Now()

	// Try setting fields on refactored User and Workspace
	w, err := client.Workspace.Create().
		SetID("ws-1").
		SetName("Acme Workspace").
		SetCreatedAt(now).
		SetUpdatedAt(now).
		Save(ctx)
	if err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}

	u, err := client.User.Create().
		SetID("user-1").
		SetEmail("bob@acme.com").
		SetPasswordHash("bcrypt-hash").
		SetName("Bob").
		SetAvatarURL("https://acme.com/bob").
		SetProvider("credential").
		SetProviderID("user-1").
		SetWorkspaceID(w.ID).
		Save(ctx)
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Try creating supporting tables
	_, err = client.AuthSession.Create().
		SetID("session-1").
		SetUserID(u.ID).
		SetWorkspaceID(w.ID).
		SetIPAddress("127.0.0.1").
		SetUserAgent("GoTest").
		Save(ctx)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	_, err = client.AuthToken.Create().
		SetID("token-1").
		SetSessionID("session-1").
		SetUserID(u.ID).
		SetType("access").
		SetHash("sha-hash").
		SetExpiresAt(now.Add(time.Hour)).
		Save(ctx)
	if err != nil {
		t.Fatalf("failed to create token: %v", err)
	}
}
