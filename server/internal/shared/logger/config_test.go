// server/internal/shared/logger/config_test.go
package logger_test

import (
	"testing"

	"github.com/kilip/opus/server/internal/shared/logger"
)

// TestDefaultConfig verifies that the default configuration populated
// by logger.DefaultConfig matches the architectural specification.
func TestDefaultConfig(t *testing.T) {
	cfg := logger.DefaultConfig()
	if cfg.Level != "info" {
		t.Errorf("expected level info, got %s", cfg.Level)
	}
	if cfg.Format != "json" {
		t.Errorf("expected format json, got %s", cfg.Format)
	}
	if cfg.FileEnabled {
		t.Error("expected file logging disabled by default")
	}
	if cfg.Filename != "logs/opus.log" {
		t.Errorf("expected logs/opus.log, got %s", cfg.Filename)
	}
}
