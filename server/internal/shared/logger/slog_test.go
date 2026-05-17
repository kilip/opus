// server/internal/shared/logger/slog_test.go
package logger_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/kilip/opus/server/internal/shared/logger"
)

func TestNewSlogLogger(t *testing.T) {
	cfg := logger.DefaultConfig()
	l, err := logger.NewSlogLogger(cfg)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	l.Info("testing concrete slog info logger output")
	l.Error("testing error output", errors.New("sentinel err"))
}

func TestSlogLogger_ContextTracing(t *testing.T) {
	ctx := context.Background()
	ctx = logger.WithRequestID(ctx, "test-req")
	ctx = logger.WithUserID(ctx, "test-user")
	ctx = logger.WithTraceID(ctx, "test-trace")
	ctx = logger.WithSpanID(ctx, "test-span")

	cfg := logger.DefaultConfig()
	l, err := logger.NewSlogLogger(cfg)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	l.InfoCtx(ctx, "testing context info output")
	l.DebugCtx(ctx, "testing context debug output")
	l.WarnCtx(ctx, "testing context warn output")
	l.ErrorCtx(ctx, "testing context error output", errors.New("ctx err"))
}

func TestSlogLogger_FileLogging(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "opus-logger-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(tmpDir)
	})

	logFile := filepath.Join(tmpDir, "sub", "opus.log")
	cfg := logger.Config{
		Level:       "debug",
		Format:      "json",
		FileEnabled: true,
		Filename:    logFile,
		MaxSize:     1,
		MaxBackups:  1,
		MaxAge:      1,
		Compress:    false,
	}

	l, err := logger.NewSlogLogger(cfg)
	if err != nil {
		t.Fatalf("failed to create file logger: %v", err)
	}

	l.Debug("debug file log")
	l.Info("info file log")

	// Verify file was created and contains logs
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Errorf("expected log file to be created at %s, but it does not exist", logFile)
	}
}

func TestSlogLogger_With(t *testing.T) {
	cfg := logger.DefaultConfig()
	l, err := logger.NewSlogLogger(cfg)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	child := l.With(logger.String("child_key", "child_val"))
	if child == nil {
		t.Fatal("expected child logger to be non-nil")
	}
	child.Info("testing child logger output")
}

func TestSlogLogger_Formats(t *testing.T) {
	formats := []string{"text", "json"}
	levels := []string{"debug", "info", "warn", "error"}

	for _, format := range formats {
		for _, level := range levels {
			cfg := logger.Config{
				Level:       level,
				Format:      format,
				FileEnabled: false,
			}
			l, err := logger.NewSlogLogger(cfg)
			if err != nil {
				t.Fatalf("failed to create logger with format %s level %s: %v", format, level, err)
			}
			l.Info("testing levels and formats")
		}
	}
}
