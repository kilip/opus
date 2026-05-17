// server/internal/shared/logger/fields_test.go
package logger_test

import (
	"testing"

	"github.com/kilip/opus/server/internal/shared/logger"
)

func TestFields(t *testing.T) {
	s := logger.String("k1", "v1")
	if s.Key != "k1" || s.Value.String() != "v1" {
		t.Errorf("expected k1=v1, got %s=%v", s.Key, s.Value)
	}

	i := logger.Int("k2", 42)
	if i.Key != "k2" || i.Value.Int64() != 42 {
		t.Errorf("expected k2=42, got %s=%v", i.Key, i.Value)
	}

	i64 := logger.Int64("k3", int64(100))
	if i64.Key != "k3" || i64.Value.Int64() != 100 {
		t.Errorf("expected k3=100, got %s=%v", i64.Key, i64.Value)
	}

	b := logger.Bool("k4", true)
	if b.Key != "k4" || !b.Value.Bool() {
		t.Errorf("expected k4=true, got %s=%v", b.Key, b.Value)
	}

	redacted := logger.Redact("k5", "pii-data")
	if redacted.Key != "k5" || redacted.Value.String() != "[REDACTED]" {
		t.Errorf("expected k5=[REDACTED], got %s=%v", redacted.Key, redacted.Value)
	}
}
