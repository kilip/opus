package container

import (
	"context"
	"fmt"

	"github.com/kilip/opus/server/internal/adapter/entgo"
	adapterqueue "github.com/kilip/opus/server/internal/adapter/queue"
	"github.com/kilip/opus/server/internal/auth"
	"github.com/kilip/opus/server/internal/config"
	"github.com/kilip/opus/server/internal/delivery/gofiber"
	"github.com/kilip/opus/server/internal/shared/logger"
)

// Bootstrap orchestrates the initialization of all shared and feature domains.
// Panics if configuration is invalid or initialization fails.
func Bootstrap(cfg config.Config) {
	once.Do(func() {
		c = &container{}

		// 1. Shared Infrastructure
		log, err := logger.NewSlogLogger(logger.DefaultConfig())
		if err != nil {
			panic(fmt.Errorf("container: failed to initialize logger: %w", err))
		}
		c.log = log

		dbClient, err := entgo.NewClient(cfg.Database)
		if err != nil {
			panic(fmt.Errorf("container: failed to initialize database: %w", err))
		}
		c.db = dbClient

		if err := entgo.AutoMigrate(c.db, context.Background()); err != nil {
			panic(fmt.Errorf("container: failed to run auto-migrations: %w", err))
		}

		q, err := adapterqueue.NewQueue(cfg.Queue, c.db, c.log)
		if err != nil {
			panic(fmt.Errorf("container: failed to init queue: %w", err))
		}
		c.queue = q

		c.bus = adapterqueue.NewEventBus()

		c.fiber = gofiber.New(cfg.Server, c.log)

		// 2. Domain Bootstraps
		auth.Bootstrap(entgo.NewAuthRepo(c.db), c.bus, c.queue, c.log, cfg.Auth)

		// 3. Delivery Bootstrap
		gofiber.Bootstrap(c.fiber, c.log, cfg.Server)
	})
}
