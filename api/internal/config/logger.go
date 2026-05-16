// api/internal/config/logger.go
package config

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	logger     *slog.Logger
	loggerOnce sync.Once
)

type multiHandler struct {
	handlers []slog.Handler
}

func (m *multiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, h := range m.handlers {
		if h.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (m *multiHandler) Handle(ctx context.Context, record slog.Record) error {
	for _, h := range m.handlers {
		if h.Enabled(ctx, record.Level) {
			if err := h.Handle(ctx, record); err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newHandlers := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		newHandlers[i] = h.WithAttrs(attrs)
	}
	return &multiHandler{handlers: newHandlers}
}

func (m *multiHandler) WithGroup(name string) slog.Handler {
	newHandlers := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		newHandlers[i] = h.WithGroup(name)
	}
	return &multiHandler{handlers: newHandlers}
}

func GetLogger() *slog.Logger {
	loggerOnce.Do(func() {
		cfg := GetConfig()
		opusDir := GetOpusDir()
		logDir := filepath.Join(opusDir, "logs")
		_ = os.MkdirAll(logDir, 0755)

		logFile := &lumberjack.Logger{
			Filename:   filepath.Join(logDir, "api.log"),
			MaxSize:    50, // megabytes
			MaxBackups: 30,
			MaxAge:     30, // days
			Compress:   true,
		}

		var level slog.Level
		if cfg.Server.Env == "development" {
			level = slog.LevelDebug
		} else {
			level = slog.LevelInfo
		}

		opts := &slog.HandlerOptions{Level: level}

		var handlers []slog.Handler
		// Stdout handler: Text in dev, JSON in prod
		if cfg.Server.Env == "development" {
			handlers = append(handlers, slog.NewTextHandler(os.Stdout, opts))
		} else {
			handlers = append(handlers, slog.NewJSONHandler(os.Stdout, opts))
		}

		// File handler: Always JSON
		handlers = append(handlers, slog.NewJSONHandler(logFile, opts))

		logger = slog.New(&multiHandler{handlers: handlers})
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
