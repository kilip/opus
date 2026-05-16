// api/internal/config/logger.go
package config

import (
	"log/slog"
	"os"
	"sync"
)

var (
	logger     *slog.Logger
	loggerOnce sync.Once
)

func GetLogger() *slog.Logger {
	loggerOnce.Do(func() {
		cfg := GetConfig()
		var handler slog.Handler
		if cfg.Server.Env == "development" {
			handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
				Level: slog.LevelDebug,
			})
		} else {
			handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
				Level: slog.LevelInfo,
			})
		}
		logger = slog.New(handler)
	})
	return logger
}

// SetLogger sets the logger singleton. This is mainly used for testing.
func SetLogger(l *slog.Logger) {
	logger = l
	loggerOnce.Do(func() {}) // Ensure loggerOnce is marked as done
}

// ResetLogger resets the logger singleton and its sync.Once.
// This is mainly used for testing to force re-initialization.
func ResetLogger() {
	logger = nil
	loggerOnce = sync.Once{}
}
