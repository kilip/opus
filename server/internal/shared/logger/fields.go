// server/internal/shared/logger/fields.go
package logger

import "log/slog"

// Field is an alias to slog.Attr to provide type safety and direct optimization.
type Field = slog.Attr

// String creates a new Field of type string.
func String(key, val string) Field { return slog.String(key, val) }

// Int creates a new Field of type int.
func Int(key string, val int) Field { return slog.Int(key, val) }

// Int64 creates a new Field of type int64.
func Int64(key string, val int64) Field { return slog.Int64(key, val) }

// Bool creates a new Field of type bool.
func Bool(key string, val bool) Field { return slog.Bool(key, val) }

// Any creates a new Field of type any.
func Any(key string, val any) Field { return slog.Any(key, val) }

// Err creates a new Field for an error.
func Err(err error) Field { return slog.Any("error", err) }

// Redact creates a new Field with a redacted string value to protect sensitive information.
func Redact(key, _ string) Field {
	return slog.String(key, "[REDACTED]")
}
