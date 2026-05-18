package container

import (
	"sync"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/opus/server/ent"
	"github.com/kilip/opus/server/internal/shared/logger"
	"github.com/kilip/opus/server/internal/shared/queue"
)

type container struct {
	db    *ent.Client
	log   logger.Logger
	queue queue.Queue
	bus   queue.EventBus
	fiber *fiber.App
}

var (
	c    *container
	once sync.Once
)

func mustInit() {
	if c == nil {
		panic("container: Bootstrap has not been called")
	}
}

// GetQueue returns the initialized queue.Queue.
func GetQueue() queue.Queue {
	mustInit()
	return c.queue
}

// GetEventBus returns the initialized queue.EventBus.
func GetEventBus() queue.EventBus {
	mustInit()
	return c.bus
}

// GetLogger returns the initialized logger.Logger.
func GetLogger() logger.Logger {
	mustInit()
	return c.log
}

// GetDB returns the initialized *ent.Client.
func GetDB() *ent.Client {
	mustInit()
	return c.db
}

// GetFiber returns the initialized *fiber.App.
func GetFiber() *fiber.App {
	mustInit()
	return c.fiber
}
