package entgo_test

import (
	"context"
	"testing"

	"github.com/kilip/opus/server/internal/adapter/entgo"
	"github.com/kilip/opus/server/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthRepo_FindByEmail(t *testing.T) {
	cfg := config.DatabaseConfig{
		Driver: "sqlite3",
		DSN:    "file:ent_auth?mode=memory&cache=shared&_fk=1",
	}
	client, err := entgo.NewClient(cfg)
	require.NoError(t, err)
	defer func() { _ = client.Close() }()

	ctx := context.Background()
	require.NoError(t, entgo.AutoMigrate(client, ctx))

	_, err = client.User.Create().
		SetID("test-id").
		SetEmail("test@example.com").
		SetPasswordHash("hash").
		SetProvider("credential").
		SetProviderID("test-id").
		SetWorkspaceID("test-ws").
		Save(ctx)
	require.NoError(t, err)

	repo := entgo.NewAuthRepo(client)
	u, err := repo.FindUserByEmail(ctx, "test@example.com")
	assert.NoError(t, err)
	assert.Equal(t, "test@example.com", u.Email)
}
