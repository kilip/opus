// server/internal/config/loader.go
package config

import (
	"os"
	"path/filepath"
)

func resolveConfigDir() string {
	if opusHome := os.Getenv("OPUS_HOME"); opusHome != "" {
		return opusHome
	}
	// Fallback to user home
	home, err := os.UserHomeDir()
	if err != nil {
		return "." // fallback if home cannot be determined
	}
	return filepath.Join(home, ".opus")
}

func Load() (*Config, error) {
	configDir := resolveConfigDir()

	// Auto-create directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, err
	}

	return &Config{}, nil
}
