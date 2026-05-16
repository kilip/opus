// api/internal/config/config.go
package config

import (
	"fmt"
	"os"
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
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
	Env  string `mapstructure:"env"`
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

func GetConfig() *Config {
	cfgOnce.Do(func() {
		viper.SetConfigName("config")
		viper.SetConfigType("toml")
		viper.AddConfigPath("$HOME/.opus")
		viper.SetEnvPrefix("OPUS")
		viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
		viper.AutomaticEnv()

		// Defaults
		viper.SetDefault("server.host", "0.0.0.0")
		viper.SetDefault("server.port", 8080)
		viper.SetDefault("server.env", "production")
		viper.SetDefault("database.driver", "sqlite")
		viper.SetDefault("database.dsn", "$HOME/.opus/opus.db")
		viper.SetDefault("auth.access_token_ttl", 15)
		viper.SetDefault("auth.refresh_token_ttl", 10080)
		viper.SetDefault("auth.secret", "")

		_ = viper.ReadInConfig()

		cfg = &Config{}
		_ = viper.Unmarshal(cfg)
		cfg.Database.DSN = os.ExpandEnv(cfg.Database.DSN)
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
