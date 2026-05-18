package container_test

import (
	"testing"

	"github.com/kilip/opus/server/internal/adapter/entgo"
	"github.com/kilip/opus/server/internal/config"
	"github.com/kilip/opus/server/internal/container"
	"github.com/kilip/opus/server/internal/shared/queue"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
)

func TestBootstrap(t *testing.T) {
	cfg := config.Config{
		Database: entgo.Config{
			Driver: "sqlite3",
			DSN:    "file:ent?mode=memory&cache=shared&_fk=1",
		},
		Queue: queue.Config{
			Driver: queue.DriverDatabase,
		},
	}

	container.Bootstrap(cfg)

	assert.NotNil(t, container.GetDB())
	assert.NotNil(t, container.GetLogger())
	assert.NotNil(t, container.GetQueue())
	assert.NotNil(t, container.GetEventBus())
	assert.NotNil(t, container.GetFiber())
}
