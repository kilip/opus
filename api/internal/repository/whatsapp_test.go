//go:build integration

package repository

import (
	"context"
	"testing"
	
	"github.com/kilip/opus/api/ent/enttest"
	"github.com/stretchr/testify/assert"
	_ "github.com/mattn/go-sqlite3"
)

func TestWhatsAppRepository(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&_fk=1")
	defer client.Close()
	ctx := context.Background()

	// Create user
	user := client.User.Create().
		SetID("user-123").
		SetEmail("test@test.com").
		SetName("Test User").
		SetProvider("email").
		SaveX(ctx)
	
	repo := NewWhatsAppRepository(client)
	
	// Test UpsertSession
	session, err := repo.UpsertSession(ctx, user.ID, "UNAUTHENTICATED", "")
	if err != nil {
		t.Fatalf("failed to upsert session: %v", err)
	}
	if session.Status != "UNAUTHENTICATED" {
		t.Errorf("expected UNAUTHENTICATED, got %s", session.Status)
	}

	// Test UpdateStatus
	err = repo.UpdateStatus(ctx, user.ID, "CONNECTED", "jid-123")
	assert.NoError(t, err)

	// Test GetAllActiveSessions
	sessions, err := repo.GetAllActiveSessions(ctx)
	assert.NoError(t, err)
	assert.Len(t, sessions, 1)
	assert.Equal(t, "jid-123", sessions[0].Jid)
}
