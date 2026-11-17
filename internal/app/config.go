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
	AppHome                string        `env:"APP_HOME, default="`
	Host                   string        `env:"APP_HOST, default=0.0.0.0"`
	Port                   string        `env:"APP_PORT, default=7880"`
	LogLevel               slog.Level    `env:"APP_LOG_LEVEL, default=info"`
	ReadTimeout            time.Duration `env:"APP_READ_TIMEOUT, default=10s"`
	WriteTimeout           time.Duration `env:"APP_WRITE_TIMEOUT, default=10s"`
	IdleTimeout            time.Duration `env:"APP_IDLE_TIMEOUT, default=60s"`
	ShutdownTimeout        time.Duration `env:"APP_SHUTDOWN_TIMEOUT, default=10s"`
	BodyLimit              int           `env:"APP_BODY_LIMIT, default=4194304"`
	DatabaseURL            string        `env:"APP_DATABASE_URL, default=file:opus.db?_pragma=foreign_keys(ON)&_pragma=busy_timeout(5000)"`
	BaseURL                string        `env:"APP_BASE_URL, default=http://localhost:7880"`
	TrustedOrigins         []string      `env:"APP_TRUSTED_ORIGINS"`
	SessionSecret          string        `env:"AUTH_SECRET, default="`
	SessionTTL             time.Duration `env:"AUTH_SESSION_TTL, default=720h"`
	AppVersion             string        `env:"APP_VERSION"`
	SentryDSN              string        `env:"APP_SENTRY_DSN, default="`
	SentryEnvironment      string        `env:"APP_SENTRY_ENVIRONMENT, default=development"`
	SentryTracesSampleRate float64       `env:"APP_SENTRY_TRACES_SAMPLE_RATE, default=1.0"`
	SMTPHost               string        `env:"APP_SMTP_HOST, default="`
	SMTPPort               int           `env:"APP_SMTP_PORT, default=587"`
	SMTPUsername           string        `env:"APP_SMTP_USERNAME, default="`
	SMTPPassword           string        `env:"APP_SMTP_PASSWORD, default="`
	SMTPFrom               string        `env:"APP_SMTP_FROM, default="`
	GitHubClientID         string        `env:"AUTH_GITHUB_ID, default="`
	GitHubClientSecret     string        `env:"AUTH_GITHUB_SECRET, default="`
	GoogleClientID         string        `env:"AUTH_GOOGLE_ID, default="`
	GoogleClientSecret     string        `env:"AUTH_GOOGLE_SECRET, default="`
	FacebookClientID       string        `env:"AUTH_FACEBOOK_ID, default="`
	FacebookClientSecret   string        `env:"AUTH_FACEBOOK_SECRET, default="`
	TrackerPollInterval    time.Duration `env:"TRACKER_POLL_INTERVAL, default=60s"`
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
