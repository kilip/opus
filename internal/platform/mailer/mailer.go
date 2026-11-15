package mailer

import (
	"context"
	"errors"
)

var (
	// ErrHeaderInjection indicates a header field contains illegal CR or LF characters.
	ErrHeaderInjection = errors.New("mailer: header contains CR or LF")
	// ErrUnencryptedAuth indicates an attempt to authenticate over an unencrypted connection.
	ErrUnencryptedAuth = errors.New("mailer: refusing to send credentials over unencrypted connection")
)

// Mailer sends emails.
type Mailer interface {
	Send(ctx context.Context, to, subject, body string) error
}
