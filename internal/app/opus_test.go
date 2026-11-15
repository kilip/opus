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
