// server/internal/shared/logger/handler.go
package logger

import (
	"context"
	"log/slog"
)

// MultiHandler multiplexes a single log entry into multiple sub-handlers.
type MultiHandler struct {
	handlers []slog.Handler
}

// NewMultiHandler creates a new MultiHandler using standard sub-handlers.
func NewMultiHandler(handlers ...slog.Handler) slog.Handler {
	return &MultiHandler{handlers: handlers}
}

// Enabled checks if any sub-handler is enabled for the specified log level.
func (h *MultiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, sub := range h.handlers {
		if sub.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

// Handle clones the log record and forwards it to each sub-handler.
func (h *MultiHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, sub := range h.handlers {
		if sub.Enabled(ctx, r.Level) {
			if err := sub.Handle(ctx, r.Clone()); err != nil {
				return err
			}
		}
	}
	return nil
}

// WithAttrs returns a new MultiHandler with attributes bound to all sub-handlers.
func (h *MultiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	nextHandlers := make([]slog.Handler, len(h.handlers))
	for i, sub := range h.handlers {
		nextHandlers[i] = sub.WithAttrs(attrs)
	}
	return &MultiHandler{handlers: nextHandlers}
}

// WithGroup returns a new MultiHandler with log fields scoped to the given group name.
func (h *MultiHandler) WithGroup(name string) slog.Handler {
	nextHandlers := make([]slog.Handler, len(h.handlers))
	for i, sub := range h.handlers {
		nextHandlers[i] = sub.WithGroup(name)
	}
	return &MultiHandler{handlers: nextHandlers}
}
