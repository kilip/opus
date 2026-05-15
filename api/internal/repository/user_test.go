package repository_test

import (
	"context"
	"testing"

	"github.com/kilip/opus/api/ent/enttest"
	"github.com/kilip/opus/api/internal/model"
	"github.com/kilip/opus/api/internal/repository"
	"github.com/stretchr/testify/assert"

	_ "github.com/mattn/go-sqlite3"
)

func TestUserRepository_CreateAndFind(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer func() { _ = client.Close() }()

	repo := repository.NewUserRepository(client)
	ctx := context.Background()

	u := &model.User{
		ID:    "user_1",
		Email: "test@example.com",
		Name:  "Test User",
	}

	created, err := repo.Create(ctx, u)
	assert.NoError(t, err)
	assert.Equal(t, u.Email, created.Email)

	found, err := repo.FindByID(ctx, "user_1")
	assert.NoError(t, err)
	assert.Equal(t, u.Email, found.Email)
}

func TestUserRepository_FindByEmail(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer func() { _ = client.Close() }()

	repo := repository.NewUserRepository(client)
	ctx := context.Background()

	u := &model.User{
		ID:    "user_2",
		Email: "find@example.com",
		Name:  "Find User",
	}

	_, _ = repo.Create(ctx, u)

	found, err := repo.FindByEmail(ctx, "find@example.com")
	assert.NoError(t, err)
	assert.Equal(t, u.ID, found.ID)

	notFound, err := repo.FindByEmail(ctx, "nonexistent@example.com")
	assert.Error(t, err)
	assert.Nil(t, notFound)
}
