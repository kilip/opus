package mailer_test

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/kilip/opus/internal/platform/mailer"
)

func TestConsoleMailer_Send(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	m := mailer.NewConsole(logger)
	err := m.Send(context.Background(), "pak.bos@example.com", "Hello", "This is the body")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "pak.bos@example.com") {
		t.Errorf("log output missing recipient; got: %s", out)
	}
	if !strings.Contains(out, "This is the body") {
		t.Errorf("log output missing body; got: %s", out)
	}
}
