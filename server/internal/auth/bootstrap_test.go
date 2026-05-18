package auth_test

import (
	"github.com/kilip/opus/server/ent/enttest"
	"github.com/kilip/opus/server/internal/adapter/entgo"
	"github.com/kilip/opus/server/internal/auth"
	"github.com/kilip/opus/server/internal/shared/logger"
	"github.com/kilip/opus/server/internal/shared/queue"
	_ "github.com/mattn/go-sqlite3"
	"testing"
)

func TestBootstrap(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer func() { _ = client.Close() }()

	repo := entgo.NewAuthRepo(client)
	bus := queue.NewNoopEventBus()
	q := queue.NewNoopQueue()
	log := &logger.NoopLogger{}
	cfg := auth.Config{
		AccessTokenTTL:  "15m",
		RefreshTokenTTL: "168h",
	}

	auth.Bootstrap(repo, bus, q, log, cfg)

	svc := auth.GetService()
	if svc == nil {
		t.Fatal("expected service to be initialized")
	}

	r := auth.GetRepository()
	if r == nil {
		t.Fatal("expected repository to be initialized")
	}
}
