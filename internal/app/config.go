package app

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/sethvargo/go-envconfig"
)

type Config struct {
	AppName                string        `env:"APP_NAME, default=opus"`
	Host                   string        `env:"HOST, default=0.0.0.0"`
	Port                   string        `env:"PORT, default=7880"`
	LogLevel               slog.Level    `env:"LOG_LEVEL, default=info"`
	ReadTimeout            time.Duration `env:"READ_TIMEOUT, default=10s"`
	WriteTimeout           time.Duration `env:"WRITE_TIMEOUT, default=10s"`
	IdleTimeout            time.Duration `env:"IDLE_TIMEOUT, default=60s"`
	ShutdownTimeout        time.Duration `env:"SHUTDOWN_TIMEOUT, default=10s"`
	BodyLimit              int           `env:"BODY_LIMIT, default=4194304"`
	DatabaseURL            string        `env:"DATABASE_URL, default=file:opus.db?_pragma=foreign_keys(ON)&_pragma=busy_timeout(5000)"`
	AppVersion             string        `env:"APP_VERSION"`
	SentryDSN              string        `env:"SENTRY_DSN, default="`
	SentryEnvironment      string        `env:"SENTRY_ENVIRONMENT, default=development"`
	SentryTracesSampleRate float64       `env:"SENTRY_TRACES_SAMPLE_RATE, default=1.0"`
}

func loadEnvFile(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if k, v, ok := strings.Cut(line, "="); ok {
			k = strings.TrimSpace(k)
			v = strings.TrimSpace(v)
			if os.Getenv(k) == "" {
				_ = os.Setenv(k, v)
			}
		}
	}
}

func GetConfig(ctx context.Context) (Config, error) {
	loadEnvFile(".env")
	var cfg Config
	err := envconfig.Process(ctx, &cfg)
	return cfg, err
}
