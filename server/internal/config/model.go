//go:generate go run generate.go
package config

import (
	"github.com/kilip/opus/server/internal/auth"
	"github.com/kilip/opus/server/internal/delivery/gofiber"
	"github.com/kilip/opus/server/internal/shared/queue"
)

// DatabaseConfig holds the configuration details for the database connection.
type DatabaseConfig struct {
	// Driver defines the database engine/driver to use.
	Driver string `mapstructure:"driver" json:"driver" jsonschema:"enum=sqlite3,enum=postgres,default=sqlite3,description=Database driver"`

	// DSN is the data source name containing connection options.
	DSN string `mapstructure:"dsn" json:"dsn" jsonschema:"description=Data source name. Inject via OPUS_DATABASE_DSN for production secrets"`
}

// Config is the top-level configuration structure.
type Config struct {
	// TestField is a test configuration field.
	TestField string `mapstructure:"test_field" json:"test_field" jsonschema:"description=A test configuration field"`

	// Server is the configuration for the GoFiber HTTP server.
	Server gofiber.Config `mapstructure:"server" json:"server"`

	// Database is the database configuration section.
	Database DatabaseConfig `mapstructure:"database" json:"database"`

	// Auth is the authentication configuration section.
	Auth auth.Config `mapstructure:"auth" json:"auth"`

	// Queue is the queue configuration section.
	Queue queue.Config `mapstructure:"queue" json:"queue"`
}
