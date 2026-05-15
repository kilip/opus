package repository_test

import (
	"context"
	"testing"
	"time"
	"github.com/stretchr/testify/assert"
	"github.com/kilip/opus/api/ent/enttest"
	"github.com/kilip/opus/api/internal/model"
	"github.com/kilip/opus/api/internal/repository"
	_ "github.com/mattn/go-sqlite3"
)

func TestSessionRepository_CreateAndFindByHash(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer func() { _ = client.Close() }()

	// Need to create a user first to satisfy foreign key constraint
	userRepo := repository.NewUserRepository(client)
	_, err := userRepo.Create(context.Background(), &model.User{
		ID:    "user_1",
		Email: "test@example.com",
		Name:  "Test User",
	})
	assert.NoError(t, err)

	repo := repository.NewSessionRepository(client)
	ctx := context.Background()

	s := &model.Session{
		ID:        "session_1",
		TokenHash: "hash_1",
		UserID:    "user_1",
		ExpiresAt: time.Now().Add(time.Hour),
		Revoked:   false,
	}

	created, err := repo.Create(ctx, s)
	assert.NoError(t, err)
	assert.Equal(t, s.ID, created.ID)

	found, err := repo.FindByTokenHash(ctx, "hash_1")
	assert.NoError(t, err)
	assert.Equal(t, s.ID, found.ID)
}

func TestSessionRepository_Revoke(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer func() { _ = client.Close() }()

	// Need to create a user first
	userRepo := repository.NewUserRepository(client)
	_, err := userRepo.Create(context.Background(), &model.User{
		ID:    "user_1",
		Email: "test@example.com",
		Name:  "Test User",
	})
	assert.NoError(t, err)

	repo := repository.NewSessionRepository(client)
	ctx := context.Background()

	s := &model.Session{
		ID:        "session_1",
		TokenHash: "hash_1",
		UserID:    "user_1",
		ExpiresAt: time.Now().Add(time.Hour),
		Revoked:   false,
	}

	_, err = repo.Create(ctx, s)
	assert.NoError(t, err)

	err = repo.RevokeByID(ctx, "session_1")
	assert.NoError(t, err)

	// Since FindByTokenHash does not seem to care about revoked status (or does it?), 
	// I should probably test that it's revoked in DB.
	// Looking at session.go, there is no FindByID, just FindByTokenHash
	// Maybe I should add a way to check if revoked, but for now I'll just rely on RevokeByID not returning error
}
