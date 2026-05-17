# ADR-006: Server Logger Architecture

**Status:** Accepted
**Date:** 2026-05-17
**Deciders:** Chief Architect
**Context:** Opus Server (`opus/server/`)

---

## 1. Context

Opus requires a robust, high-performance logging system to capture application activity, errors, and debug information. Given the modular monolith design (Clean Architecture, see ADR-001) and the use of the Fiber web framework (see ADR-005), the logging solution must not tightly couple any layer — domain, adapter, or delivery — to a specific third-party logging library.

Furthermore, debugging a production server requires tracing logs back to specific HTTP requests. This necessitates a context-aware logger that can extract and append structured tracing metadata (`request_id`, `user_id`, `trace_id`, `span_id`) to log entries seamlessly.

Finally, to maintain high performance and prevent runtime panics caused by malformed log arguments, the logging API must enforce type safety for structured fields.

This ADR establishes the interface-driven, context-aware, and strongly-typed logging architecture for all Go server-side code under `opus/server/`. The ADR defines **what** the logger must do, not **how** it is implemented; the backing engine is an implementation detail and may be replaced without an ADR amendment.

---

## 2. Decision

Opus Server adopts an **Interface-Driven, Context-Aware Logger** with strictly typed fields. All application code depends exclusively on the `logger.Logger` interface. No concrete logging engine is referenced outside of `internal/shared/logger/`.

---

### 2.1 Directory Structure

```
opus/
└── server/
    └── internal/
        └── shared/
            └── logger/
                ├── logger.go       # Logger interface definition
                ├── fields.go       # Typed field constructors and Redact helper
                ├── context.go      # Context key constants and extraction helpers
                ├── config.go       # logger.Config struct (hybrid composition — ADR-002)
                └── noop.go         # NoopLogger implementation (testing utility)
```

---

### 2.2 Logger Interface

The `Logger` interface is defined in `internal/shared/logger/logger.go`. All layers — Delivery, Service, Repository — depend solely on this interface. No concrete type is referenced outside the `logger` package.

```go
// internal/shared/logger/logger.go
package logger

import "context"

// Logger defines the standard logging contract for the entire application.
// Implementations must be safe for concurrent use.
type Logger interface {
    Debug(msg string, fields ...Field)
    Info(msg string, fields ...Field)
    Warn(msg string, fields ...Field)
    Error(msg string, err error, fields ...Field)

    // Fatal logs at error level and then calls os.Exit(1).
    // RESTRICTION: Fatal must only be called from main.go during application
    // bootstrap. It is prohibited in all domain, service, adapter, and
    // handler code. Use Error and return instead.
    Fatal(msg string, err error, fields ...Field)

    // Context-aware variants automatically extract and attach tracing metadata
    // (request_id, user_id, trace_id, span_id) from the provided context.
    DebugCtx(ctx context.Context, msg string, fields ...Field)
    InfoCtx(ctx context.Context, msg string, fields ...Field)
    WarnCtx(ctx context.Context, msg string, fields ...Field)
    ErrorCtx(ctx context.Context, msg string, err error, fields ...Field)

    // With returns a new Logger with the provided fields permanently attached
    // to every subsequent log entry. Use to bind component-level context
    // (e.g., logger.String("component", "auth_service")).
    With(fields ...Field) Logger
}
```

---

### 2.3 Strongly-Typed Fields and PII Masking

To guarantee type safety, the `Field` type is defined as an opaque alias within the `logger` package. Helper constructors are provided for all common value types. Callers must never construct `Field` values directly.

```go
// internal/shared/logger/fields.go
package logger

// Field is an opaque structured log field. Use the constructor functions below;
// never construct Field values directly.
type Field interface{}

// Constructor functions — the only permitted way to create Field values.
func String(key, val string) Field  { /* implementation */ }
func Int(key string, val int) Field { /* implementation */ }
func Int64(key string, val int64) Field { /* implementation */ }
func Bool(key string, val bool) Field   { /* implementation */ }
func Any(key string, val any) Field     { /* implementation */ }
func Err(err error) Field               { /* implementation */ }

// Redact replaces a sensitive string value with "[REDACTED]" before logging.
// Use for any field that may carry PII: passwords, tokens, SSNs, etc.
//
// Example: logger.Redact("api_key", rawKey)
func Redact(key, _ string) Field {
    return String(key, "[REDACTED]")
}
```

---

### 2.4 Context Keys and Tracing Metadata

Tracing metadata is propagated via `context.Context`. The `logger` package defines a fixed set of context keys corresponding to the OpenTelemetry semantic conventions for identity and tracing fields.

```go
// internal/shared/logger/context.go
package logger

import "context"

// contextKey is an unexported type for context keys defined in this package.
// Using a package-scoped type prevents collision with keys from other packages.
type contextKey int

const (
    // RequestIDKey identifies the unique ID assigned to an HTTP request.
    RequestIDKey contextKey = iota
    // UserIDKey identifies the authenticated user associated with a request.
    UserIDKey
    // TraceIDKey holds the OpenTelemetry trace ID for distributed tracing.
    TraceIDKey
    // SpanIDKey holds the OpenTelemetry span ID for distributed tracing.
    SpanIDKey
)

// Standard field names — aligned with OpenTelemetry semantic conventions.
const (
    FieldRequestID = "request_id"
    FieldUserID    = "user_id"
    FieldTraceID   = "trace_id"
    FieldSpanID    = "span_id"
)

// WithRequestID returns a new context carrying the given request ID.
func WithRequestID(ctx context.Context, id string) context.Context {
    return context.WithValue(ctx, RequestIDKey, id)
}

// WithUserID returns a new context carrying the given user ID.
func WithUserID(ctx context.Context, id string) context.Context {
    return context.WithValue(ctx, UserIDKey, id)
}

// WithTraceID returns a new context carrying the given trace ID.
func WithTraceID(ctx context.Context, id string) context.Context {
    return context.WithValue(ctx, TraceIDKey, id)
}

// WithSpanID returns a new context carrying the given span ID.
func WithSpanID(ctx context.Context, id string) context.Context {
    return context.WithValue(ctx, SpanIDKey, id)
}

// ExtractFields extracts all known tracing fields from the context and returns
// them as a slice of Field values ready to be appended to a log entry.
// Missing fields are silently omitted.
func ExtractFields(ctx context.Context) []Field {
    var fields []Field
    if v, ok := ctx.Value(RequestIDKey).(string); ok && v != "" {
        fields = append(fields, String(FieldRequestID, v))
    }
    if v, ok := ctx.Value(UserIDKey).(string); ok && v != "" {
        fields = append(fields, String(FieldUserID, v))
    }
    if v, ok := ctx.Value(TraceIDKey).(string); ok && v != "" {
        fields = append(fields, String(FieldTraceID, v))
    }
    if v, ok := ctx.Value(SpanIDKey).(string); ok && v != "" {
        fields = append(fields, String(FieldSpanID, v))
    }
    return fields
}
```

**Extensibility note:** When OpenTelemetry integration is introduced, `TraceIDKey` and `SpanIDKey` will be populated by the OTel SDK middleware. No changes to the logger interface or context extraction logic will be required.

---

### 2.5 Configuration — Hybrid Composition (ADR-002)

Following the **Hybrid Config Composition Pattern** from ADR-002, the logger owns its configuration struct. This struct is composed into the root `Config` in `internal/config/model.go`.

```go
// internal/shared/logger/config.go
package logger

// Config holds all logger configuration. It is owned by the logger package
// and composed into the root config.Config by internal/config/model.go.
//
// Environment variable overrides follow the OPUS_ prefix convention:
//   OPUS_LOG_LEVEL  — sets Level
//   OPUS_LOG_FORMAT — sets Format
type Config struct {
    // Level controls the minimum severity of emitted log entries.
    // Valid values: "debug", "info", "warn", "error". Default: "info".
    Level string `mapstructure:"level" json:"level" jsonschema:"enum=debug,enum=info,enum=warn,enum=error,default=info"`

    // Format controls the output encoding of log entries.
    // "json" produces machine-parseable output suitable for production log aggregators.
    // "text" produces human-readable, optionally colourised output for development.
    // Default: "json".
    Format string `mapstructure:"format" json:"format" jsonschema:"enum=json,enum=text,default=json"`
}
```

Root config composition (in `internal/config/model.go`):

```go
// internal/config/model.go (excerpt)
import "opus/server/internal/shared/logger"

type Config struct {
    // ... other fields
    Log logger.Config `mapstructure:"log" json:"log"`
}
```

---

### 2.6 Dependency Injection

The `Logger` instance is constructed once in `main.go` and injected into every Service, Repository, and Handler that requires it. There is no global logger variable; no `init()` function sets a package-level logger.

```go
// internal/auth/service.go
package auth

import (
    "context"
    "opus/server/internal/shared/logger"
)

// Service handles authentication business logic.
type Service struct {
    repo   Repository
    logger logger.Logger
}

// NewService constructs an AuthService.
// The logger is scoped with a "component" field so every entry from this
// service is identifiable without additional caller context.
func NewService(r Repository, l logger.Logger) *Service {
    return &Service{
        repo:   r,
        logger: l.With(logger.String("component", "auth_service")),
    }
}

func (s *Service) Login(ctx context.Context, email string) error {
    s.logger.InfoCtx(ctx, "login attempt started",
        logger.Redact("email", email),
    )
    // ...
}
```

---

### 2.7 Usage in the Delivery Layer

The request logger middleware in `delivery/fiber/middleware/logger.go` is the canonical entry point for populating tracing metadata into the context. It generates a `request_id`, injects it using `logger.WithRequestID`, and logs the request summary via `InfoCtx`.

```go
// delivery/fiber/middleware/logger.go
package middleware

import (
    "context"

    "github.com/gofiber/fiber/v3"
    "github.com/google/uuid"
    "opus/server/internal/shared/logger"
)

// RequestLogger returns a Fiber middleware that assigns a unique request_id
// to every incoming request, injects it into the context, and logs the
// request/response summary using the provided Logger.
func RequestLogger(log logger.Logger) fiber.Handler {
    return func(c fiber.Ctx) error {
        requestID := uuid.NewString()
        ctx := logger.WithRequestID(context.Background(), requestID)
        c.SetUserContext(ctx)

        err := c.Next()

        log.InfoCtx(ctx, "request completed",
            logger.String("method", c.Method()),
            logger.String("path", c.Path()),
            logger.Int("status", c.Response().StatusCode()),
        )
        return err
    }
}
```

Auth middleware populates `user_id` after token validation:

```go
// delivery/fiber/middleware/auth.go (excerpt)
ctx := logger.WithUserID(c.UserContext(), claims.UserID)
c.SetUserContext(ctx)
```

---

### 2.8 Testing — NoopLogger

A `NoopLogger` is provided in `internal/shared/logger/noop.go` for use in unit tests. It discards all log output while satisfying the `Logger` interface, eliminating the need to mock the logger in tests that are not explicitly testing log output.

```go
// internal/shared/logger/noop.go
package logger

import "context"

// NoopLogger is a Logger implementation that discards all output.
// Use it in unit tests to satisfy Logger dependencies without noise.
type NoopLogger struct{}

func (n *NoopLogger) Debug(_ string, _ ...Field)                          {}
func (n *NoopLogger) Info(_ string, _ ...Field)                           {}
func (n *NoopLogger) Warn(_ string, _ ...Field)                           {}
func (n *NoopLogger) Error(_ string, _ error, _ ...Field)                 {}
func (n *NoopLogger) Fatal(_ string, _ error, _ ...Field)                 {}
func (n *NoopLogger) DebugCtx(_ context.Context, _ string, _ ...Field)    {}
func (n *NoopLogger) InfoCtx(_ context.Context, _ string, _ ...Field)     {}
func (n *NoopLogger) WarnCtx(_ context.Context, _ string, _ ...Field)     {}
func (n *NoopLogger) ErrorCtx(_ context.Context, _ string, _ error, _ ...Field) {}
func (n *NoopLogger) With(_ ...Field) Logger                              { return n }
```

For tests that assert specific log output, generate a mock using `go.uber.org/mock`:

```go
//go:generate mockgen -destination=mock_logger.go -package=logger . Logger
```

Usage in a test:

```go
func TestLogin_LogsAttempt(t *testing.T) {
    ctrl := gomock.NewController(t)
    mockLog := logger.NewMockLogger(ctrl)
    mockLog.EXPECT().
        InfoCtx(gomock.Any(), "login attempt started", gomock.Any()).
        Times(1)

    svc := auth.NewService(repo, mockLog)
    svc.Login(context.Background(), "user@example.com")
}
```

---

## 3. Consequences

### 3.1 Positive

- **Decoupled Architecture** — The domain layer has zero dependency on any concrete logging library; the backing engine can be replaced without modifying application code.
- **Implementation Agnostic** — This ADR describes the interface contract only. The choice of underlying logging engine (standard library, zerolog, zap, etc.) is an implementation detail confined to `internal/shared/logger/`.
- **Security by Design** — PII masking is built into the logger API via the `Redact` constructor. Developers are guided to the correct pattern at the call site.
- **Structured Tracing** — Context keys and field name constants align with OpenTelemetry semantic conventions, enabling future OTel integration with minimal migration effort.
- **Testability** — `NoopLogger` eliminates logger setup boilerplate in unit tests; `MockLogger` enables precise assertion of log calls in tests that require it.
- **Type Safety** — The opaque `Field` type and typed constructors eliminate runtime errors from malformed variadic `any` arguments.
- **Config Cohesion** — `logger.Config` follows the Hybrid Composition Pattern (ADR-002); configuration is co-located with the package it governs.

### 3.2 Negative / Trade-offs

- **`Fatal` Discipline Required** — The prohibition on `Fatal` outside `main.go` is enforced by convention, not by the compiler. A linter rule (e.g., via `go-critic` or a custom `golangci-lint` plugin) is recommended to automate enforcement.
- **Context Passing Burden** — The `*Ctx` methods require `context.Context` to be propagated through every call chain. This is idiomatic Go but increases parameter verbosity.
- **NoopLogger vs Mock** — Two testing utilities exist (`NoopLogger` and `MockLogger`). Teams must apply consistent judgment: use `NoopLogger` when log output is irrelevant to the test; use `MockLogger` only when asserting specific log calls.

---

## 4. References

- [ADR-001: Server Clean Architecture](./ADR-001-server-clean-architecture.md)
- [ADR-002: Configuration Management](./ADR-002-server-configuration.md)
- [ADR-005: Server Delivery Layer with GoFiber v3](./ADR-005-server-delivery-layer-with-gofiber-v3.md)
- [OpenTelemetry Semantic Conventions — Trace Fields](https://opentelemetry.io/docs/specs/semconv/general/trace/)
- [go.uber.org/mock](https://github.com/uber-go/mock)