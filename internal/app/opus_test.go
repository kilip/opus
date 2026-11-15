package app_test

import (
	"bytes"
	"context"
	"log/slog"
	"reflect"
	"strings"
	"testing"

	"github.com/kilip/opus/internal/app"
)

func TestNewMailer_FallbackToConsole(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn}))

	cfg := app.Config{
		SMTPHost: "",
	}

	m := app.NewMailer(cfg, logger)
	if m == nil {
		t.Fatal("expected non-nil mailer")
	}

	typeName := reflect.TypeOf(m).String()
	if !strings.Contains(typeName, "consoleMailer") {
		t.Errorf("expected consoleMailer, got %s", typeName)
	}

	logOutput := buf.String()
	if !strings.Contains(logOutput, "SMTP_HOST not set, falling back to console mailer") {
		t.Errorf("expected startup warning in log, got: %q", logOutput)
	}
}

func TestNewMailer_FallbackWithNilLogger(t *testing.T) {
	cfg := app.Config{
		SMTPHost: "",
	}
	m := app.NewMailer(cfg, nil)
	if m == nil {
		t.Fatal("expected non-nil mailer even with nil logger")
	}
}

func TestNewMailer_SMTPConfigured(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn}))

	cfg := app.Config{
		SMTPHost: "smtp.example.com",
		SMTPPort: 587,
	}

	m := app.NewMailer(cfg, logger)
	if m == nil {
		t.Fatal("expected non-nil mailer")
	}

	typeName := reflect.TypeOf(m).String()
	if !strings.Contains(typeName, "smtpMailer") {
		t.Errorf("expected smtpMailer, got %s", typeName)
	}

	if buf.Len() > 0 {
		t.Errorf("expected no warning logs when SMTP is configured, got: %q", buf.String())
	}
}

func TestConfig_SMTPDefaults(t *testing.T) {
	cfg, err := app.GetConfig(context.Background())
	if err != nil {
		t.Fatalf("GetConfig returned unexpected error: %v", err)
	}

	if cfg.SMTPPort != 587 {
		t.Errorf("expected default SMTPPort 587, got %d", cfg.SMTPPort)
	}
	if cfg.SMTPHost != "" {
		t.Errorf("expected default SMTPHost to be empty, got %q", cfg.SMTPHost)
	}
	if cfg.SMTPUsername != "" {
		t.Errorf("expected default SMTPUsername to be empty, got %q", cfg.SMTPUsername)
	}
	if cfg.SMTPPassword != "" {
		t.Errorf("expected default SMTPPassword to be empty, got %q", cfg.SMTPPassword)
	}
	if cfg.SMTPFrom != "" {
		t.Errorf("expected default SMTPFrom to be empty, got %q", cfg.SMTPFrom)
	}
}

func TestConfig_OAuthEnv(t *testing.T) {
	t.Setenv("GITHUB_CLIENT_ID", "test-gh-id")
	t.Setenv("GITHUB_CLIENT_SECRET", "test-gh-secret")
	t.Setenv("GOOGLE_CLIENT_ID", "test-g-id")
	t.Setenv("GOOGLE_CLIENT_SECRET", "test-g-secret")
	t.Setenv("FACEBOOK_CLIENT_ID", "test-fb-id")
	t.Setenv("FACEBOOK_CLIENT_SECRET", "test-fb-secret")

	cfg, err := app.GetConfig(context.Background())
	if err != nil {
		t.Fatalf("GetConfig returned unexpected error: %v", err)
	}

	if cfg.GitHubClientID != "test-gh-id" || cfg.GitHubClientSecret != "test-gh-secret" {
		t.Errorf("unexpected github oauth config: id=%q secret=%q", cfg.GitHubClientID, cfg.GitHubClientSecret)
	}
	if cfg.GoogleClientID != "test-g-id" || cfg.GoogleClientSecret != "test-g-secret" {
		t.Errorf("unexpected google oauth config: id=%q secret=%q", cfg.GoogleClientID, cfg.GoogleClientSecret)
	}
	if cfg.FacebookClientID != "test-fb-id" || cfg.FacebookClientSecret != "test-fb-secret" {
		t.Errorf("unexpected facebook oauth config: id=%q secret=%q", cfg.FacebookClientID, cfg.FacebookClientSecret)
	}
}

func TestConfig_TrustedOrigins(t *testing.T) {
	t.Setenv("TRUSTED_ORIGINS", "http://localhost:5173,http://localhost:3000")
	cfg, err := app.GetConfig(context.Background())
	if err != nil {
		t.Fatalf("GetConfig returned unexpected error: %v", err)
	}
	if len(cfg.TrustedOrigins) != 2 {
		t.Fatalf("expected 2 trusted origins, got %d", len(cfg.TrustedOrigins))
	}
	if cfg.TrustedOrigins[0] != "http://localhost:5173" || cfg.TrustedOrigins[1] != "http://localhost:3000" {
		t.Errorf("unexpected trusted origins: %v", cfg.TrustedOrigins)
	}
}
