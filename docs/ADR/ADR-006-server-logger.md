# ADR-006: Server Logger Architecture

**Status:** Accepted  
**Date:** 2026-05-17  
**Deciders:** Chief Architect  
**Context:** Opus Server (`opus/server/`)

---

## 1. Context

Opus requires a robust, high-performance logging system to capture application activity, errors, and debug information. Given the modular monolith design (Clean Architecture, see ADR-001) and the use of the Fiber web framework (see ADR-005), the logging solution must not tightly couple the domain logic to a specific third-party logging library.

Furthermore, debugging a production server requires tracing logs back to specific HTTP requests. This necessitates a context-aware logger that can extract and append a `Request ID` (or other tracing metadata) to log entries seamlessly.

Finally, to maintain high performance (zero-allocation where possible) and prevent runtime panics caused by malformed log arguments, the logging API must enforce type safety for structured fields.

This ADR establishes the interface-driven, context-aware, and strongly-typed logging architecture for all Go server-side code under `opus/server/`.

---

## 2. Decision

Opus Server adopts an **Interface-Driven, Context-Aware Logger** with strictly typed fields, abstracting the underlying logging engine (initially `log/slog`).

### 2.1 Interface Definition

We define a core `Logger` interface in `internal/shared/logger/`. All layers (Delivery, Service, Repository) depend solely on this interface.

```go
package logger

import "context"

// Logger defines the standard logging contract for the entire application.
type Logger interface {
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, err error, fields ...Field)
	Fatal(msg string, err error, fields ...Field) // Logs and then calls os.Exit(1)

	// Context-aware methods for extracting Request IDs, User IDs, or tracing info
	DebugCtx(ctx context.Context, msg string, fields ...Field)
	InfoCtx(ctx context.Context, msg string, fields ...Field)
	WarnCtx(ctx context.Context, msg string, fields ...Field)
	ErrorCtx(ctx context.Context, msg string, err error, fields ...Field)
	
	// With returns a new Logger instance with the provided fields attached
	With(fields ...Field) Logger
}
```

### 2.2 Strongly-Typed Fields and PII Masking

To guarantee type safety and performance, we avoid the variadic `args ...any` pattern. Instead, we alias the underlying logging engine's field type (currently `slog.Attr`) and provide helper functions.

To prevent PII (Personally Identifiable Information) leakage, we implement a `Redact` helper for sensitive strings.

```go
package logger

import "log/slog"

type Field = slog.Attr

var (
	String = slog.String
	Int    = slog.Int
	Int64  = slog.Int64
	Bool   = slog.Bool
	Any    = slog.Any
	
	// Redact masks sensitive data before logging.
	Redact = func(key, value string) Field {
		return slog.String(key, "[REDACTED]")
	}
)
```

### 2.3 Hybrid Configuration and Stack Traces

Following the **Hybrid Config Composition Pattern** established in [ADR-002](./ADR-002-server-configuration.md), the logger's configuration is defined within its own package (`internal/shared/logger/config.go`) and composed into the root `Config` struct.

The `logger.Config` struct supports:
- **Level (`OPUS_LOG_LEVEL`):** Supports `debug`, `info`, `warn`, and `error`. Defaults to `info`.
- **Format (`OPUS_LOG_FORMAT`):** 
    - `text`: Human-readable with color support. Default for development.
    - `json`: Machine-parseable JSON. Default for production.

This approach ensures the logger remains self-contained while still being globally configurable via Viper and environment variables.

For `Error` and `Fatal` levels, the logger implementation will automatically capture and attach the stack trace (using a custom `slog.Handler` wrapper) to aid in debugging, regardless of the selected format.

### 2.4 Context-Aware Tracing (Extensible)

The logger leverages `context.Context` to automatically attach metadata. While initially focused on `request_id`, the extraction logic is designed to be extensible to:
- `user_id`: Captured from auth middleware.
- `trace_id` / `span_id`: For future OpenTelemetry integration.

1. **Context Key:** A standard key (e.g., `logger.RequestIDKey`) is defined for storing the ID in the context.
2. **Middleware:** A Fiber middleware in `delivery/http/middleware/logger.go` generates a unique Request ID, injects it into the `context.Context`, and passes it down the chain.
3. **Extraction:** The implementation of the `*Ctx` methods (e.g., `InfoCtx`) retrieves the ID from the context and appends it to the outgoing log payload.

### 2.5 Dependency Injection

The `Logger` instance is injected into Services and Repositories during application bootstrap in `main.go`. There is no global logger instance accessible across packages.

```go
// internal/auth/service.go
package auth

import (
    "context"
    "opus/server/internal/shared/logger"
)

type Service struct {
    repo   Repository
    logger logger.Logger
}

func NewService(r Repository, l logger.Logger) *Service {
    return &Service{
        repo:   r,
        logger: l.With(logger.String("component", "auth_service")),
    }
}

func (s *Service) Login(ctx context.Context, email string) error {
    s.logger.InfoCtx(ctx, "login attempt started", logger.String("email", email))
    // ...
}
```

### 2.6 Implementation Engine

The initial implementation backing this interface will be the standard library's `log/slog` (introduced in Go 1.21). `slog` provides high-performance structured logging (JSON and Text formats) natively without external dependencies.

---

## 3. Consequences

### Positive
- **Decoupled Architecture:** The domain layer has zero dependency on any concrete logging library.
- **Future-Proof:** Migrating to a different logging engine only requires modifying the `internal/shared/logger/` implementation.
- **Security by Design:** PII masking is built into the logger API via the `Redact` helper.
- **Production Ready:** JSON formatting and automatic stack traces ensure logs are actionable in production.
- **Enhanced Observability:** Enforcing `*Ctx` methods ensures tracing identifiers are consistently logged.
- **Type Safety & Performance:** The use of `logger.Field` eliminates runtime errors and reduces allocations.
- **Testability:** The interface makes it trivial to inject a mock logger during unit testing.

### Negative / Trade-offs
- **Verbosity:** Writing logs requires slightly more boilerplate (e.g., `logger.String()`).
- **Context Passing Burden:** Developers must diligently pass `context.Context` throughout the call stack.
- **Implementation Complexity:** Custom `slog.Handler` is required to handle stack traces and automatic context extraction.
