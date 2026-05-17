//go:generate go run generate.go
package config

import (
	"github.com/kilip/opus/server/internal/delivery/gofiber"
)

// Config is the top-level configuration structure.
type Config struct {
	// TestField is a test configuration field.
	TestField string `mapstructure:"test_field" json:"test_field" jsonschema:"description=A test configuration field"`

	// Server is the configuration for the GoFiber HTTP server.
	Server gofiber.Config `mapstructure:"server" json:"server"`
}
