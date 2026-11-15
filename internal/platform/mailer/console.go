package mailer

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
)

type consoleMailer struct {
	log *slog.Logger
}

// NewConsole returns a Mailer that logs emails instead of sending them.
func NewConsole(logger *slog.Logger) Mailer {
	return &consoleMailer{log: logger}
}

func (c *consoleMailer) Send(ctx context.Context, to, subject, body string) error {
	if strings.ContainsAny(to, "\r\n") {
		return fmt.Errorf("%w: recipient address", ErrHeaderInjection)
	}
	if strings.ContainsAny(subject, "\r\n") {
		return fmt.Errorf("%w: subject", ErrHeaderInjection)
	}
	if c.log != nil {
		c.log.InfoContext(ctx, "console mailer: send email",
			slog.String("to", to),
			slog.String("subject", subject),
			slog.String("body", body),
		)
	}
	return nil
}
