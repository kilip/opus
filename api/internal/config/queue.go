// api/internal/config/queue.go
package config

import (
	"sync"

	"github.com/kilip/opus/api/internal/repository/queue"
	"github.com/redis/go-redis/v9"
)

var (
	queueDriver queue.QueueDriver
	queueOnce   sync.Once
)

// GetQueueDriver returns a singleton instance of the queue driver based on configuration.
func GetQueueDriver() queue.QueueDriver {
	queueOnce.Do(func() {
		cfg := GetConfig()
		switch cfg.Queue.Driver {
		case "redis":
			rdb := redis.NewClient(&redis.Options{
				Addr:     cfg.Queue.Redis.Addr,
				Password: cfg.Queue.Redis.Password,
				DB:       cfg.Queue.Redis.DB,
			})
			queueDriver = queue.NewRedisDriver(rdb, "opus")
		default: // "entgo" or fallback
			queueDriver = queue.NewEntGoDriver(GetDatabase(), cfg.Database.Driver)
		}
	})
	return queueDriver
}
