//go:build integration

package entgo_test

import (
	"context"
	"testing"
	"time"

	"github.com/kilip/opus/server/ent/enttest"
	"github.com/kilip/opus/server/internal/adapter/entgo"
	"github.com/kilip/opus/server/internal/auth"
	_ "github.com/mattn/go-sqlite3"
)

func TestAuthRepository_Integration(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	repo := entgo.NewAuthRepo(client)

	// Assure registration linkage transactional integrity
	u := &auth.User{
		ID:          "user-v7",
		Email:       "acme@acme.com",
		Provider:    "credential",
		ProviderID:  "user-v7",
		WorkspaceID: "ws-v7",
	}

	created, err := repo.CreateUserWithWorkspace(ctx, u, nil, "Acme Corp")
	if err != nil {
		t.Fatalf("failed to insert user repository transactions: %v", err)
	}

	// Fetch DB object
	fetched, err := repo.FindUserByEmail(ctx, "acme@acme.com")
	if err != nil {
		t.Fatalf("failed to fetch user: %v", err)
	}

	if fetched.ID != created.ID || fetched.WorkspaceID != "ws-v7" {
		t.Errorf("retrieved mismatching db entities")
	}

	// Verify state verification queries
	err = repo.CreateOAuthState(ctx, "my-state", "google", time.Now().Add(10*time.Minute))
	if err != nil {
		t.Fatalf("failed to create oauth state: %v", err)
	}

	val, err := repo.ValidateOAuthState(ctx, "my-state")
	if err != nil || val != "google" {
		t.Errorf("failed validation: val=%s err=%v", val, err)
	}
}
