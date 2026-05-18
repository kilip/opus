// Package queue provides the factory functions to instantiate the Queue and EventBus.
package queue

import (
	"database/sql"
	"fmt"

	"github.com/kilip/opus/server/internal/adapter/queue/memory"
	"github.com/kilip/opus/server/internal/adapter/queue/sqlite"
	"github.com/kilip/opus/server/internal/shared/logger"
	"github.com/kilip/opus/server/internal/shared/queue"
)

// NewQueue constructs and returns the Queue implementation specified by cfg.Driver.
func NewQueue(cfg queue.Config, db *sql.DB, log logger.Logger) (queue.Queue, error) {
	switch cfg.Driver {
	case queue.DriverSQLite:
		return sqlite.NewSQLiteQueue(db, cfg.Concurrency, log)
	case queue.DriverPostgres:
		return nil, fmt.Errorf("postgres queue driver is not yet implemented")
	case queue.DriverRedis:
		return nil, fmt.Errorf("redis queue driver is not yet implemented")
	default:
		return nil, fmt.Errorf("unknown queue driver: %s", cfg.Driver)
	}
}

// NewEventBus always returns the in-process EventBus implementation.
func NewEventBus() queue.EventBus {
	return memory.NewInMemoryEventBus()
}
