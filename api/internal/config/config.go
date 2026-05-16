// api/internal/config/config.go
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Auth     AuthConfig     `mapstructure:"auth"`
}

type ServerConfig struct {
	Host    string `mapstructure:"host"`
	Port    int    `mapstructure:"port"`
	Env     string `mapstructure:"env"`
	ApiURL  string `mapstructure:"api_url"`
	DashURL string `mapstructure:"dash_url"`
}

type DatabaseConfig struct {
	Driver string `mapstructure:"driver"`
	DSN    string `mapstructure:"dsn"`
}

type AuthConfig struct {
	Secret          string      `mapstructure:"secret"`
	AccessTokenTTL  int         `mapstructure:"access_token_ttl"`
	RefreshTokenTTL int         `mapstructure:"refresh_token_ttl"`
	Google          OAuthConfig `mapstructure:"google"`
	GitHub          OAuthConfig `mapstructure:"github"`
}

type OAuthConfig struct {
	ClientID     string `mapstructure:"client_id"`
	ClientSecret string `mapstructure:"client_secret"`
	RedirectURL  string `mapstructure:"redirect_url"`
}

var (
	cfg     *Config
	cfgOnce sync.Once
)

func GetOpusDir() string {
	if dir := os.Getenv("OPUS_HOME"); dir != "" {
		return dir
	}
	// Default to production if OPUS_SERVER_ENV is not set
	env := os.Getenv("OPUS_SERVER_ENV")
	if env == "" {
		env = "production"
	}

	// Always use local .opus in development mode
	if env == "development" {
		if _, err := os.Stat(".opus"); err == nil {
			if abs, err := filepath.Abs(".opus"); err == nil {
				return abs
			}
			return ".opus"
		}
		// Fallback to parent directory if in dev mode
		if abs, err := filepath.Abs(".."); err == nil {
			return filepath.Join(abs, ".opus")
		}
		return "../.opus"
	}

	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".opus")
}

func GetConfig() *Config {
	cfgOnce.Do(func() {
		opusDir := GetOpusDir()
		viper.SetConfigName("config")
		viper.SetConfigType("toml")
		viper.AddConfigPath(opusDir)
		viper.AddConfigPath(".")
		viper.SetEnvPrefix("OPUS")
		viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
		viper.AutomaticEnv()

		// Defaults
		viper.SetDefault("server.host", "0.0.0.0")
		viper.SetDefault("server.port", 8080)
		viper.SetDefault("server.env", "production")

		port := viper.GetInt("server.port")
		viper.SetDefault("server.api_url", fmt.Sprintf("http://localhost:%d", port))
		viper.SetDefault("server.dash_url", "http://localhost:3000")
		viper.SetDefault("database.driver", "sqlite")
		viper.SetDefault("database.dsn", filepath.Join(opusDir, "opus.db"))
		viper.SetDefault("auth.access_token_ttl", 15)
		viper.SetDefault("auth.refresh_token_ttl", 10080)
		viper.SetDefault("auth.secret", "")

		_ = viper.ReadInConfig()

		cfg = &Config{}
		_ = viper.Unmarshal(cfg)
		cfg.Database.DSN = os.ExpandEnv(cfg.Database.DSN)

		// Auto-config OAuth redirect URLs if empty
		if cfg.Auth.Google.RedirectURL == "" {
			cfg.Auth.Google.RedirectURL = fmt.Sprintf("%s/auth/google/callback", cfg.Server.ApiURL)
		}
		if cfg.Auth.GitHub.RedirectURL == "" {
			cfg.Auth.GitHub.RedirectURL = fmt.Sprintf("%s/auth/github/callback", cfg.Server.ApiURL)
		}
		// Mask secret for logging
		maskedSecret := ""
		if len(cfg.Auth.Secret) > 4 {
			maskedSecret = cfg.Auth.Secret[:2] + "****" + cfg.Auth.Secret[len(cfg.Auth.Secret)-2:]
		} else if cfg.Auth.Secret != "" {
			maskedSecret = "****"
		}
		fmt.Printf("Config loaded: Env=%s, DB=%s, AuthSecret=%s\n", cfg.Server.Env, cfg.Database.Driver, maskedSecret)
	})
	return cfg
}
