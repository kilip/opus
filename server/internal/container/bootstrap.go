package container

import (
	"context"
	"fmt"

	"github.com/kilip/opus/server/internal/adapter/entgo"
	adapterqueue "github.com/kilip/opus/server/internal/adapter/queue"
	"github.com/kilip/opus/server/internal/config"
	"github.com/kilip/opus/server/internal/shared/logger"
)

// Bootstrap orchestrates the initialization of all shared and feature domains.
// Panics if configuration is invalid or initialization fails.
func Bootstrap(cfg config.Config) {
	once.Do(func() {
		c = &container{}

		// 1. Logger
		log, err := logger.NewSlogLogger(logger.DefaultConfig())
		if err != nil {
			panic(fmt.Errorf("container: failed to initialize logger: %w", err))
		}
		c.log = log

		// 2. Database
		dbClient, err := entgo.NewClient(cfg.Database)
		if err != nil {
			panic(fmt.Errorf("container: failed to initialize database: %w", err))
		}
		c.db = dbClient

		// 3. Auto-Migrate Schema
		if err := entgo.AutoMigrate(c.db, context.Background()); err != nil {
			panic(fmt.Errorf("container: failed to run auto-migrations: %w", err))
		}

		// 4. Extract *sql.DB for shared connections
		db, err := entgo.DB(c.db)
		if err != nil {
			panic(fmt.Errorf("container: failed to extract *sql.DB: %w", err))
		}

		// 5. Queue
		q, err := adapterqueue.NewQueue(cfg.Queue, db, c.log)
		if err != nil {
			panic(fmt.Errorf("container: failed to init queue: %w", err))
		}
		c.queue = q

		// 6. EventBus
		c.bus = adapterqueue.NewEventBus()
	})
}
