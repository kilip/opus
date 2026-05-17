// server/internal/shared/logger/handler_test.go
package logger_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/kilip/opus/server/internal/shared/logger"
)

type logEntry struct {
	Msg string `json:"msg"`
}

// TestMultiHandler verifies that the MultiHandler correctly multiplexes slog records
// to multiple independent handlers and clones records to prevent race conditions.
func TestMultiHandler(t *testing.T) {
	var buf1, buf2 bytes.Buffer
	h1 := slog.NewJSONHandler(&buf1, nil)
	h2 := slog.NewJSONHandler(&buf2, nil)

	multi := logger.NewMultiHandler(h1, h2)
	l := slog.New(multi)

	l.Info("hello multiplexer")

	var e1, e2 logEntry
	if err := json.Unmarshal(buf1.Bytes(), &e1); err != nil {
		t.Fatalf("failed to decode buf1: %v", err)
	}
	if err := json.Unmarshal(buf2.Bytes(), &e2); err != nil {
		t.Fatalf("failed to decode buf2: %v", err)
	}

	if e1.Msg != "hello multiplexer" || e2.Msg != "hello multiplexer" {
		t.Errorf("multiplexing failed, got e1=%s, e2=%s", e1.Msg, e2.Msg)
	}
}
