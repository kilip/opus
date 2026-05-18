// server/internal/shared/logger/logger.go
package logger

import "context"

// Mocks live in server/mocks/ — regenerate with task mocks.

// Logger defines the standard logging contract for the entire application.
type Logger interface {
	// Debug logs a message at the debug level.
	Debug(msg string, fields ...Field)
	// Info logs a message at the info level.
	Info(msg string, fields ...Field)
	// Warn logs a message at the warn level.
	Warn(msg string, fields ...Field)
	// Error logs a message at the error level with an associated error context.
	Error(msg string, err error, fields ...Field)
	// Fatal logs a message at the error level and terminates the application process.
	Fatal(msg string, err error, fields ...Field)

	// DebugCtx logs a debug message and extracts tracking details from the context.
	DebugCtx(ctx context.Context, msg string, fields ...Field)
	// InfoCtx logs an info message and extracts tracking details from the context.
	InfoCtx(ctx context.Context, msg string, fields ...Field)
	// WarnCtx logs a warning message and extracts tracking details from the context.
	WarnCtx(ctx context.Context, msg string, fields ...Field)
	// ErrorCtx logs an error message and extracts tracking details from the context.
	ErrorCtx(ctx context.Context, msg string, err error, fields ...Field)

	// With returns a child Logger pre-configured with the given log fields.
	With(fields ...Field) Logger
}
