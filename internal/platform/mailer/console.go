package mailer

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
)

type consoleMailer struct {
	out io.Writer
}

// NewConsole returns a Mailer that writes emails to stderr instead of sending them.
func NewConsole(_ *slog.Logger) Mailer {
	return &consoleMailer{out: os.Stderr}
}

func (c *consoleMailer) Send(_ context.Context, to, subject, body string) error {
	if strings.ContainsAny(to, "\r\n") {
		return fmt.Errorf("%w: recipient address", ErrHeaderInjection)
	}
	if strings.ContainsAny(subject, "\r\n") {
		return fmt.Errorf("%w: subject", ErrHeaderInjection)
	}
	_, _ = fmt.Fprintf(c.out, "[console-mailer] to=%s subject=%q\n%s\n", to, subject, body)
	return nil
}
