package mailer

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
)

type captureHandler struct {
	records []slog.Record
}

func (h *captureHandler) Enabled(context.Context, slog.Level) bool { return true }
func (h *captureHandler) Handle(_ context.Context, r slog.Record) error {
	h.records = append(h.records, r)
	return nil
}
func (h *captureHandler) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h *captureHandler) WithGroup(string) slog.Handler      { return h }

func TestConsoleMailer_WritesStderrAndNeverLogsBodyToSlog(t *testing.T) {
	ch := &captureHandler{}
	logger := slog.New(ch)
	prev := slog.Default()
	slog.SetDefault(logger)
	defer slog.SetDefault(prev)

	var buf bytes.Buffer
	m := &consoleMailer{out: &buf}
	if err := m.Send(context.Background(), "user@example.com", "Your OTP code", "secret-otp-123456"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if !strings.Contains(buf.String(), "secret-otp-123456") {
		t.Fatalf("expected stderr buffer to contain OTP body, got %q", buf.String())
	}
	if len(ch.records) != 0 {
		t.Fatalf("expected 0 slog records from consoleMailer, got %d", len(ch.records))
	}
}
