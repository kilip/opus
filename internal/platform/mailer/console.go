package mailer

import (
	"context"
	"log/slog"
)

type consoleMailer struct {
	log *slog.Logger
}

// NewConsole returns a Mailer that logs emails instead of sending them.
func NewConsole(logger *slog.Logger) Mailer {
	return &consoleMailer{log: logger}
}

func (c *consoleMailer) Send(ctx context.Context, to, subject, body string) error {
	c.log.InfoContext(ctx, "console mailer: send email",
		slog.String("to", to),
		slog.String("subject", subject),
		slog.String("body", body),
	)
	return nil
}
