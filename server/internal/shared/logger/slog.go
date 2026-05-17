// server/internal/shared/logger/slog.go
package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"gopkg.in/natefinch/lumberjack.v2"
)

// SlogLogger is a structured logger implementation backed by log/slog.
type SlogLogger struct {
	logger *slog.Logger
}

// NewSlogLogger creates a new instance of SlogLogger with the given configuration.
func NewSlogLogger(cfg Config) (*SlogLogger, error) {
	var level slog.Level
	switch cfg.Level {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	var handlers []slog.Handler

	var consoleWriter io.Writer = os.Stdout
	if cfg.Format == "text" {
		handlers = append(handlers, slog.NewTextHandler(consoleWriter, opts))
	} else {
		handlers = append(handlers, slog.NewJSONHandler(consoleWriter, opts))
	}

	if cfg.FileEnabled {
		if err := os.MkdirAll(filepath.Dir(cfg.Filename), 0755); err != nil {
			return nil, err
		}

		fileWriter := &lumberjack.Logger{
			Filename:   cfg.Filename,
			MaxSize:    cfg.MaxSize,
			MaxBackups: cfg.MaxBackups,
			MaxAge:     cfg.MaxAge,
			Compress:   cfg.Compress,
		}

		handlers = append(handlers, slog.NewJSONHandler(fileWriter, opts))
	}

	multiplexer := NewMultiHandler(handlers...)
	return &SlogLogger{
		logger: slog.New(multiplexer),
	}, nil
}

var _ Logger = (*SlogLogger)(nil)

// Debug logs records at the Debug severity level.
func (l *SlogLogger) Debug(msg string, fields ...Field) {
	l.logger.LogAttrs(context.Background(), slog.LevelDebug, msg, fields...)
}

// Info logs records at the Info severity level.
func (l *SlogLogger) Info(msg string, fields ...Field) {
	l.logger.LogAttrs(context.Background(), slog.LevelInfo, msg, fields...)
}

// Warn logs records at the Warn severity level.
func (l *SlogLogger) Warn(msg string, fields ...Field) {
	l.logger.LogAttrs(context.Background(), slog.LevelWarn, msg, fields...)
}

// Error logs records at the Error severity level along with an error context.
func (l *SlogLogger) Error(msg string, err error, fields ...Field) {
	all := append([]Field{Err(err)}, fields...)
	l.logger.LogAttrs(context.Background(), slog.LevelError, msg, all...)
}

// Fatal logs records at the Error severity level and exits the runtime immediately.
func (l *SlogLogger) Fatal(msg string, err error, fields ...Field) {
	all := append([]Field{Err(err)}, fields...)
	l.logger.LogAttrs(context.Background(), slog.LevelError, msg, all...)
	os.Exit(1)
}

// DebugCtx extracts tracing details and logs records at the Debug level.
func (l *SlogLogger) DebugCtx(ctx context.Context, msg string, fields ...Field) {
	ctxFields := ExtractFields(ctx)
	all := make([]Field, 0, len(ctxFields)+len(fields))
	all = append(all, ctxFields...)
	all = append(all, fields...)
	l.logger.LogAttrs(ctx, slog.LevelDebug, msg, all...)
}

// InfoCtx extracts tracing details and logs records at the Info level.
func (l *SlogLogger) InfoCtx(ctx context.Context, msg string, fields ...Field) {
	ctxFields := ExtractFields(ctx)
	all := make([]Field, 0, len(ctxFields)+len(fields))
	all = append(all, ctxFields...)
	all = append(all, fields...)
	l.logger.LogAttrs(ctx, slog.LevelInfo, msg, all...)
}

// WarnCtx extracts tracing details and logs records at the Warn level.
func (l *SlogLogger) WarnCtx(ctx context.Context, msg string, fields ...Field) {
	ctxFields := ExtractFields(ctx)
	all := make([]Field, 0, len(ctxFields)+len(fields))
	all = append(all, ctxFields...)
	all = append(all, fields...)
	l.logger.LogAttrs(ctx, slog.LevelWarn, msg, all...)
}

// ErrorCtx extracts tracing details and logs records at the Error level.
func (l *SlogLogger) ErrorCtx(ctx context.Context, msg string, err error, fields ...Field) {
	ctxFields := ExtractFields(ctx)
	all := make([]Field, 0, len(ctxFields)+len(fields)+1)
	all = append(all, Err(err))
	all = append(all, ctxFields...)
	all = append(all, fields...)
	l.logger.LogAttrs(ctx, slog.LevelError, msg, all...)
}

// With binds the given fields into a child structured logger instances.
func (l *SlogLogger) With(fields ...Field) Logger {
	args := make([]any, len(fields))
	for i, f := range fields {
		args[i] = f
	}
	return &SlogLogger{
		logger: l.logger.With(args...),
	}
}
