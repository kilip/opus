//go:generate go run generate.go
package config

import (
	"github.com/kilip/opus/server/internal/adapter/entgo"
	"github.com/kilip/opus/server/internal/auth"
	"github.com/kilip/opus/server/internal/delivery/gofiber"
	"github.com/kilip/opus/server/internal/shared/queue"
)

// Config is the top-level configuration structure.
type Config struct {
	// TestField is a test configuration field.
	TestField string `mapstructure:"test_field" json:"test_field" jsonschema:"description=A test configuration field"`

	// Server is the configuration for the GoFiber HTTP server.
	Server gofiber.Config `mapstructure:"server" json:"server"`

	// Database is the database configuration section.
	Database entgo.Config `mapstructure:"database" json:"database"`

	// Auth is the authentication configuration section.
	Auth auth.Config `mapstructure:"auth" json:"auth"`

	// Queue is the queue configuration section.
	Queue queue.Config `mapstructure:"queue" json:"queue"`
}
