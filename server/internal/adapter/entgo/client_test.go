package entgo_test

import (
	"testing"

	"github.com/kilip/opus/server/internal/adapter/entgo"
	"github.com/kilip/opus/server/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestNewClient(t *testing.T) {
	cfg := entgo.Config{
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

func TestDBExposure(t *testing.T) {
	client := testutil.NewTestEntClient(t)
	db, err := entgo.DB(client)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if db == nil {
		t.Fatalf("expected non-nil db")
	}
	if err := db.Ping(); err != nil {
		t.Fatalf("expected ping success, got %v", err)
	}
}
