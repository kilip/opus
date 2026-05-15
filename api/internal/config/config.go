// api/internal/config/config.go
package config

import (
	"sync"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Auth     AuthConfig
}

type ServerConfig struct {
	Host string
	Port int
	Env  string
}

type DatabaseConfig struct {
	Driver string
	DSN    string
}

type AuthConfig struct {
	Secret          string
	AccessTokenTTL  int
	RefreshTokenTTL int
	Google          OAuthConfig
	GitHub          OAuthConfig
}

type OAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

var (
	cfg     *Config
	cfgOnce sync.Once
)

func GetConfig() *Config {
	cfgOnce.Do(func() {
		viper.SetConfigName("config")
		viper.SetConfigType("toml")
		viper.AddConfigPath("$HOME/.opus")
		viper.SetEnvPrefix("OPUS")
		viper.AutomaticEnv()

		// Defaults
		viper.SetDefault("server.host", "0.0.0.0")
		viper.SetDefault("server.port", 8080)
		viper.SetDefault("server.env", "production")
		viper.SetDefault("database.driver", "sqlite")
		viper.SetDefault("database.dsn", "$HOME/.opus/opus.db")
		viper.SetDefault("auth.access_token_ttl", 15)
		viper.SetDefault("auth.refresh_token_ttl", 10080)

		_ = viper.ReadInConfig()

		cfg = &Config{}
		_ = viper.Unmarshal(cfg)
	})
	return cfg
}
