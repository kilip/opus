//go:build integration

package container_test

import (
	"testing"

	"github.com/kilip/opus/server/internal/adapter/entgo"
	"github.com/kilip/opus/server/internal/config"
	"github.com/kilip/opus/server/internal/container"
	"github.com/kilip/opus/server/internal/dash"
	"github.com/kilip/opus/server/internal/shared/queue"
)

func TestDashBootstrap_Integration(t *testing.T) {
	cfg := config.Config{
		Database: entgo.Config{
			Driver: "sqlite3",
			DSN:    "file:test?mode=memory&cache=shared&_fk=1",
		},
		Queue: queue.Config{
			Driver: queue.DriverDatabase,
		},
		Dash: dash.Config{Address: ":8081"},
	}
	container.Bootstrap(cfg)

	dashApp := container.GetDash()
	if dashApp == nil {
		t.Fatal("expected Dash server to be initialized, got nil")
	}
}
