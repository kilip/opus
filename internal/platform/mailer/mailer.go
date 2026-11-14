package mailer

import "context"

// Mailer sends emails.
type Mailer interface {
	Send(ctx context.Context, to, subject, body string) error
}
