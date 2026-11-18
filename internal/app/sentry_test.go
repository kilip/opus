package app

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
)

type spySentryTransport struct {
	mu     sync.Mutex
	events []*sentry.Event
}

func (s *spySentryTransport) Configure(_ sentry.ClientOptions) {}

func (s *spySentryTransport) SendEvent(event *sentry.Event) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, event)
}

func (s *spySentryTransport) Flush(_ time.Duration) bool { return true }

func (s *spySentryTransport) FlushWithContext(_ context.Context) bool { return true }

func (s *spySentryTransport) Close() {}

func (s *spySentryTransport) snapshot() []*sentry.Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*sentry.Event, len(s.events))
	copy(out, s.events)
	return out
}

func TestGetLogger_NoSentryHandlerWhenDSNEmpty(t *testing.T) {
	logger := GetLogger(Config{LogLevel: slog.LevelInfo})
	if _, ok := logger.Handler().(*slog.JSONHandler); !ok {
		t.Fatalf("expected *slog.JSONHandler when SentryDSN is empty, got %T", logger.Handler())
	}
}

func TestSentryErrorHandler_ForwardsErrorOnlyAndPreservesOutput(t *testing.T) {
	transport := &spySentryTransport{}
	if err := sentry.Init(sentry.ClientOptions{
		Dsn:       "https://public@example.com/1",
		Transport: transport,
	}); err != nil {
		t.Fatalf("sentry.Init failed: %v", err)
	}
	defer sentry.Flush(time.Second)

	var out bytes.Buffer
	base := slog.NewJSONHandler(&out, &slog.HandlerOptions{Level: slog.LevelDebug})
	logger := slog.New(newSentryErrorHandler(base)).With("service", "tracker").WithGroup("sync")

	logger.Debug("debug message", "k", "v")
	logger.Info("info message", "k", "v")
	logger.Warn("warn message", "k", "v")
	logger.Error("github auth status failed", "error", errors.New("gh not found"), "project_id", "p-1")

	events := transport.snapshot()
	if len(events) != 1 {
		t.Fatalf("expected 1 sentry event for Error level only, got %d", len(events))
	}
	ev := events[0]
	if ev.Level != sentry.LevelError {
		t.Fatalf("expected sentry LevelError, got %q", ev.Level)
	}
	if ev.Message != "github auth status failed" {
		t.Fatalf("expected message %q, got %q", "github auth status failed", ev.Message)
	}
	slogCtx := ev.Contexts["slog"]
	if got := slogCtx["service"]; got != "tracker" {
		t.Fatalf("expected context slog.service=tracker, got %v", got)
	}
	if got := slogCtx["sync.error"]; got != "gh not found" {
		t.Fatalf("expected context slog.sync.error='gh not found', got %v", got)
	}
	if got := slogCtx["sync.project_id"]; got != "p-1" {
		t.Fatalf("expected context slog.sync.project_id=p-1, got %v", got)
	}
	if len(ev.Exception) == 0 || ev.Exception[0].Value != "gh not found" {
		t.Fatalf("expected exception 'gh not found', got %+v", ev.Exception)
	}

	logged := out.String()
	for _, want := range []string{"debug message", "info message", "warn message", "github auth status failed"} {
		if !strings.Contains(logged, want) {
			t.Fatalf("expected underlying stdout handler to contain %q, got %q", want, logged)
		}
	}
}

func TestGetLogger_AttachesSentryHandlerWhenDSNConfigured(t *testing.T) {
	logger := GetLogger(Config{
		LogLevel:  slog.LevelInfo,
		SentryDSN: "https://public@example.com/1",
	})
	if _, ok := logger.Handler().(*sentryErrorHandler); !ok {
		t.Fatalf("expected *sentryErrorHandler when SentryDSN is set, got %T", logger.Handler())
	}
}
