// server/internal/config/loader.go
package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

func resolveConfigDir() string {
	if opusHome := os.Getenv("OPUS_HOME"); opusHome != "" {
		return opusHome
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return filepath.Join(home, ".opus")
}

func LoadWithViper() (*Config, *viper.Viper, error) {
	configDir := resolveConfigDir()

	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, nil, err
	}

	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("json")

	// Resolution order: highest to lowest priority (Viper checks in order of addition)

	// 1. Explicit override via env var (Highest file priority)
	if opusHome := os.Getenv("OPUS_HOME"); opusHome != "" {
		v.AddConfigPath(opusHome)
	}

	// 2. User home directory
	home, _ := os.UserHomeDir()
	if home != "" {
		v.AddConfigPath(filepath.Join(home, ".opus"))
	}

	// 3. Project-local directories (Development fallbacks)
	v.AddConfigPath(".opus")    // when run from project root
	v.AddConfigPath("../.opus") // when run from server/ directory

	v.SetEnvPrefix("OPUS")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))

	// Register nested default values for CORS to enable correct environment variable unmarshaling.
	// Without explicit default values, Viper will ignore env-vars for nested structures (like server.cors)
	// when no config.json file exists.
	v.SetDefault("server.cors.allowed_origins", []string{"http://localhost:5173"})
	v.SetDefault("server.cors.allowed_methods", []string{})
	v.SetDefault("server.cors.allowed_headers", []string{})
	v.SetDefault("server.cors.expose_headers", []string{})
	v.SetDefault("server.cors.allow_credentials", true)
	v.SetDefault("server.cors.max_age", 3600)

	v.SetDefault("database.driver", "sqlite3")
	v.SetDefault("database.dsn", ":memory:")
	v.SetDefault("queue.driver", "database")

	v.AutomaticEnv()

	// It's okay if config file doesn't exist, we might just use env vars
	_ = v.ReadInConfig()

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, nil, err
	}

	return &cfg, v, nil
}

func Load() (*Config, error) {
	cfg, _, err := LoadWithViper()
	return cfg, err
}

func Watch(v *viper.Viper, onChange func(cfg *Config)) {
	v.WatchConfig()
	v.OnConfigChange(func(e fsnotify.Event) {
		var cfg Config
		if err := v.Unmarshal(&cfg); err != nil {
			// log error, do not apply partial config
			return
		}
		onChange(&cfg)
	})
}
