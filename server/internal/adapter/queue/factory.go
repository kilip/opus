// Package queue provides the factory functions to instantiate the Queue and EventBus.
package queue

import (
	"fmt"

	"github.com/kilip/opus/server/ent"
	"github.com/kilip/opus/server/internal/adapter/queue/database"
	"github.com/kilip/opus/server/internal/adapter/queue/memory"
	"github.com/kilip/opus/server/internal/adapter/queue/redis"
	"github.com/kilip/opus/server/internal/shared/logger"
	"github.com/kilip/opus/server/internal/shared/queue"
)

// NewQueue constructs and returns the Queue implementation specified by cfg.Driver.
func NewQueue(cfg queue.Config, db *ent.Client, log logger.Logger) (queue.Queue, error) {
	switch cfg.Driver {
	case queue.DriverDatabase:
		return database.NewDatabaseQueue(db, cfg.Concurrency, log), nil
	case queue.DriverRedis:
		return redis.NewRedisQueue(cfg.DSN, cfg.Concurrency, log)
	default:
		return nil, fmt.Errorf("queue: unsupported driver %q", cfg.Driver)
	}
}

// NewEventBus always returns the in-process EventBus implementation.
// The EventBus does not require a persistent backend.
func NewEventBus() queue.EventBus {
	return memory.NewInMemoryEventBus()
}
