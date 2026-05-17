// server/internal/config/loader.go
package config

import (
	"os"
	"path/filepath"

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

	// Resolution order: lowest to highest priority
	v.AddConfigPath(filepath.Join("opus", ".opus")) // development
	home, _ := os.UserHomeDir()
	if home != "" {
		v.AddConfigPath(filepath.Join(home, ".opus")) // user home
	}

	// Explicit override via env var
	if opusHome := os.Getenv("OPUS_HOME"); opusHome != "" {
		v.AddConfigPath(opusHome)
	}

	v.SetEnvPrefix("OPUS")
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
