package entgo_test

import (
	"github.com/kilip/opus/server/internal/adapter/entgo"
	"testing"
)

func TestConfig(t *testing.T) {
	cfg := entgo.Config{
		Driver: "sqlite3",
		DSN:    ":memory:",
	}
	if cfg.Driver != "sqlite3" {
		t.Errorf("expected driver sqlite3, got %s", cfg.Driver)
	}
}
