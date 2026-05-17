package entgo_test

import (
	"testing"

	"github.com/kilip/opus/server/internal/adapter/entgo"
	"github.com/kilip/opus/server/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestNewClient(t *testing.T) {
	cfg := config.DatabaseConfig{
		Driver: "sqlite3",
		DSN:    "file:ent?mode=memory&cache=shared&_fk=1",
	}
	client, err := entgo.NewClient(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, client)
	if client != nil {
		defer func() { _ = client.Close() }()
	}
}
