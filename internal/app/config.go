package app

import (
	"context"
	"log/slog"
	"time"

	"github.com/sethvargo/go-envconfig"
)

type Config struct {
	AppName         string        `env:"APP_NAME, default=opus"`
	Host            string        `env:"HOST, default=0.0.0.0"`
	Port            string        `env:"PORT, default=7880"`
	LogLevel        slog.Level    `env:"LOG_LEVEL, default=info"`
	ReadTimeout     time.Duration `env:"READ_TIMEOUT, default=10s"`
	WriteTimeout    time.Duration `env:"WRITE_TIMEOUT, default=10s"`
	IdleTimeout     time.Duration `env:"IDLE_TIMEOUT, default=60s"`
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT, default=10s"`
	BodyLimit       int           `env:"BODY_LIMIT, default=4194304"`
	DatabaseURL     string        `env:"DATABASE_URL, default=file:opus.db?_pragma=foreign_keys(ON)&_pragma=busy_timeout(5000)"`
}

func GetConfig(ctx context.Context) (Config, error) {
	var cfg Config
	err := envconfig.Process(ctx, &cfg)
	return cfg, err
}
