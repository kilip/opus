// server/internal/shared/logger/noop.go
package logger

import "context"

// NoopLogger is a Logger implementation that discards all output.
type NoopLogger struct{}

var _ Logger = (*NoopLogger)(nil)

// Debug discards log records at the debug level.
func (n *NoopLogger) Debug(_ string, _ ...Field) {}

// Info discards log records at the info level.
func (n *NoopLogger) Info(_ string, _ ...Field) {}

// Warn discards log records at the warn level.
func (n *NoopLogger) Warn(_ string, _ ...Field) {}

// Error discards log records at the error level.
func (n *NoopLogger) Error(_ string, _ error, _ ...Field) {}

// Fatal discards log records at the error level and does nothing.
func (n *NoopLogger) Fatal(_ string, _ error, _ ...Field) {}

// DebugCtx discards log records with context at the debug level.
func (n *NoopLogger) DebugCtx(_ context.Context, _ string, _ ...Field) {}

// InfoCtx discards log records with context at the info level.
func (n *NoopLogger) InfoCtx(_ context.Context, _ string, _ ...Field) {}

// WarnCtx discards log records with context at the warn level.
func (n *NoopLogger) WarnCtx(_ context.Context, _ string, _ ...Field) {}

// ErrorCtx discards log records with context at the error level.
func (n *NoopLogger) ErrorCtx(_ context.Context, _ string, _ error, _ ...Field) {}

// With returns the same NoopLogger receiver instance.
func (n *NoopLogger) With(_ ...Field) Logger { return n }
