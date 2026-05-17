// Package gofiber implements the HTTP delivery layer for the Opus server using GoFiber v3.
package gofiber

// Config defines the configuration for the GoFiber HTTP server.
type Config struct {
	// Address is the TCP address the HTTP server listens on (e.g., ":8080").
	Address string `mapstructure:"address" json:"address" jsonschema:"default=:8080,description=TCP address the HTTP server listens on"`

	// Debug enables debug mode and verbose request logging.
	Debug bool `mapstructure:"debug" json:"debug" jsonschema:"description=Enable debug mode and verbose request logging"`

	// BodyLimit specifies the maximum request body size in bytes.
	BodyLimit int `mapstructure:"body_limit" json:"body_limit" jsonschema:"default=4194304,description=Max request body size in bytes"`

	// Prefork enables Fiber Prefork mode (production only).
	Prefork bool `mapstructure:"prefork" json:"prefork" jsonschema:"description=Enable Fiber Prefork mode (production only)"`
}
