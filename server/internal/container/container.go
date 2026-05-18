package container

import (
	"sync"

	"github.com/kilip/opus/server/ent"
	"github.com/kilip/opus/server/internal/shared/logger"
	"github.com/kilip/opus/server/internal/shared/queue"
)

type container struct {
	db    *ent.Client
	log   logger.Logger
	queue queue.Queue
	bus   queue.EventBus
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
// Panics if Bootstrap has not been called.
func GetQueue() queue.Queue {
	mustInit()
	return c.queue
}

// GetEventBus returns the initialized queue.EventBus.
// Panics if Bootstrap has not been called.
func GetEventBus() queue.EventBus {
	mustInit()
	return c.bus
}

// GetLogger returns the initialized logger.Logger.
// Panics if Bootstrap has not been called.
func GetLogger() logger.Logger {
	mustInit()
	return c.log
}

// GetDB returns the initialized *ent.Client.
// Panics if Bootstrap has not been called.
func GetDB() *ent.Client {
	mustInit()
	return c.db
}
