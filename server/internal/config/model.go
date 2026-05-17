// server/internal/config/model.go
package config

// Config is the top-level configuration structure.
type Config struct {
	TestField string `mapstructure:"test_field" json:"test_field"`
}
