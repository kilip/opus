//go:generate go run generate.go
package config

// Config is the top-level configuration structure.
type Config struct {
	TestField string `mapstructure:"test_field" json:"test_field" jsonschema:"description=A test configuration field"`
}
