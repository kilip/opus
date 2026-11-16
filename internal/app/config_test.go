package app

import (
	"context"
	"log/slog"
	"reflect"
	"testing"
	"time"
)

func TestGetConfig_AppEnvPrefix(t *testing.T) {
	t.Setenv("APP_HOST", "127.0.0.1")
	t.Setenv("APP_PORT", "9000")
	t.Setenv("APP_LOG_LEVEL", "warn")
	t.Setenv("APP_READ_TIMEOUT", "15s")
	t.Setenv("APP_WRITE_TIMEOUT", "25s")
	t.Setenv("APP_IDLE_TIMEOUT", "45s")
	t.Setenv("APP_SHUTDOWN_TIMEOUT", "20s")
	t.Setenv("APP_BODY_LIMIT", "1048576")
	t.Setenv("APP_DATABASE_URL", "file:custom.db")
	t.Setenv("APP_BASE_URL", "https://example.com")
	t.Setenv("APP_TRUSTED_ORIGINS", "https://a.com,https://b.com")
	t.Setenv("APP_SENTRY_DSN", "https://key@sentry.io/123")
	t.Setenv("APP_SENTRY_ENVIRONMENT", "production")
	t.Setenv("APP_SENTRY_TRACES_SAMPLE_RATE", "0.75")
	t.Setenv("APP_SMTP_HOST", "smtp.example.com")
	t.Setenv("APP_SMTP_PORT", "465")
	t.Setenv("APP_SMTP_USERNAME", "user@example.com")
	t.Setenv("APP_SMTP_PASSWORD", "supersecret")
	t.Setenv("APP_SMTP_FROM", "noreply@example.com")

	cfg, err := GetConfig(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Host != "127.0.0.1" {
		t.Errorf("expected Host %q, got %q", "127.0.0.1", cfg.Host)
	}
	if cfg.Port != "9000" {
		t.Errorf("expected Port %q, got %q", "9000", cfg.Port)
	}
	if cfg.LogLevel != slog.LevelWarn {
		t.Errorf("expected LogLevel %v, got %v", slog.LevelWarn, cfg.LogLevel)
	}
	if cfg.ReadTimeout != 15*time.Second {
		t.Errorf("expected ReadTimeout 15s, got %v", cfg.ReadTimeout)
	}
	if cfg.WriteTimeout != 25*time.Second {
		t.Errorf("expected WriteTimeout 25s, got %v", cfg.WriteTimeout)
	}
	if cfg.IdleTimeout != 45*time.Second {
		t.Errorf("expected IdleTimeout 45s, got %v", cfg.IdleTimeout)
	}
	if cfg.ShutdownTimeout != 20*time.Second {
		t.Errorf("expected ShutdownTimeout 20s, got %v", cfg.ShutdownTimeout)
	}
	if cfg.BodyLimit != 1048576 {
		t.Errorf("expected BodyLimit 1048576, got %d", cfg.BodyLimit)
	}
	if cfg.DatabaseURL != "file:custom.db" {
		t.Errorf("expected DatabaseURL %q, got %q", "file:custom.db", cfg.DatabaseURL)
	}
	if cfg.BaseURL != "https://example.com" {
		t.Errorf("expected BaseURL %q, got %q", "https://example.com", cfg.BaseURL)
	}
	expectedOrigins := []string{"https://a.com", "https://b.com"}
	if !reflect.DeepEqual(cfg.TrustedOrigins, expectedOrigins) {
		t.Errorf("expected TrustedOrigins %v, got %v", expectedOrigins, cfg.TrustedOrigins)
	}
	if cfg.SentryDSN != "https://key@sentry.io/123" {
		t.Errorf("expected SentryDSN %q, got %q", "https://key@sentry.io/123", cfg.SentryDSN)
	}
	if cfg.SentryEnvironment != "production" {
		t.Errorf("expected SentryEnvironment %q, got %q", "production", cfg.SentryEnvironment)
	}
	if cfg.SentryTracesSampleRate != 0.75 {
		t.Errorf("expected SentryTracesSampleRate 0.75, got %v", cfg.SentryTracesSampleRate)
	}
	if cfg.SMTPHost != "smtp.example.com" {
		t.Errorf("expected SMTPHost %q, got %q", "smtp.example.com", cfg.SMTPHost)
	}
	if cfg.SMTPPort != 465 {
		t.Errorf("expected SMTPPort 465, got %d", cfg.SMTPPort)
	}
	if cfg.SMTPUsername != "user@example.com" {
		t.Errorf("expected SMTPUsername %q, got %q", "user@example.com", cfg.SMTPUsername)
	}
	if cfg.SMTPPassword != "supersecret" {
		t.Errorf("expected SMTPPassword %q, got %q", "supersecret", cfg.SMTPPassword)
	}
	if cfg.SMTPFrom != "noreply@example.com" {
		t.Errorf("expected SMTPFrom %q, got %q", "noreply@example.com", cfg.SMTPFrom)
	}
}
