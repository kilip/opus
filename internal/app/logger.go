package app

import (
	"log/slog"
	"os"
)

// ponytail: writes to os.Stdout only; add io.Writer parameter or LOG_OUTPUT when file/multi-sink routing needed.
func GetLogger(cfg Config) *slog.Logger {
	var handler slog.Handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     cfg.LogLevel,
		AddSource: cfg.LogLevel == slog.LevelDebug,
	})
	if cfg.SentryDSN != "" {
		handler = newSentryErrorHandler(handler)
	}
	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger
}
