package app

import (
	"log/slog"
	"time"

	"github.com/getsentry/sentry-go"
)

// InitSentry initializes Sentry client if SentryDSN is configured.
// It returns a flush function to be called on shutdown.
func InitSentry(cfg Config, logger *slog.Logger) func() {
	if cfg.SentryDSN == "" {
		return func() {}
	}
	err := sentry.Init(sentry.ClientOptions{
		Dsn:              cfg.SentryDSN,
		Environment:      cfg.SentryEnvironment,
		TracesSampleRate: cfg.SentryTracesSampleRate,
		EnableTracing:    true,
		AttachStacktrace: true,
	})
	if err != nil {
		logger.Error("failed to initialize sentry", "error", err)
		return func() {}
	}
	logger.Info("sentry initialized", "environment", cfg.SentryEnvironment)
	return func() {
		sentry.Flush(2 * time.Second)
	}
}
