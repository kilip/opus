# ADR-005: Server Delivery Layer with GoFiber v3

**Status:** Accepted
**Date:** 2026-05-17
**Deciders:** Chief Architect
**Context:** Opus Server (`opus/server/`)

---

## 1. Context

Opus Server requires a robust, high-performance HTTP framework for its Delivery Layer. The framework must align cleanly with our strict architectural boundaries (Clean Architecture, ADR-001) and our strict API response contracts (ADR-004). We are evaluating GoFiber v3 to fulfil this role.

---

## 2. Decision

We will adopt **GoFiber v3** as the exclusive framework for the HTTP Delivery Layer in Opus Server. The delivery layer is structured explicitly under **`internal/delivery/fiber/`**.

> **Note for AI agents and automated tooling:** The canonical delivery layer path is `internal/delivery/fiber/`. Any reference to `delivery/http/` or `delivery/fiber` without the `internal` prefix in other ADRs (e.g. ADR-001) is illustrative only and predates this decision. All concrete implementation files must reside under `internal/delivery/fiber/`.

---

### 2.1 Directory Structure

```
opus/
└── server/
    └── internal/
        └── delivery/
            └── fiber/              # GoFiber v3 delivery layer (canonical path)
                ├── handler/        # Route handlers per domain
                │   ├── auth_handler.go
                │   └── agent_handler.go
                ├── middleware/     # Cross-cutting Fiber middleware
                │   ├── auth.go     # JWT validation middleware
                │   └── logger.go   # Request logger middleware (uses logger.Logger — see §2.5)
                ├── router/         # Route registration + app bootstrap
                │   └── router.go
                └── response/       # Fiber-specific response wrappers (ADR-004)
                    └── response.go
```

---

### 2.2 Global Error Handling and Response Contract

GoFiber v3 allows setting a custom `ErrorHandler`. We will implement a global Fiber error handler that automatically catches any `fiber.Error` or unhandled `error`, and formats it into the strict RFC 7807 Problem Details envelope mandated by ADR-004.

This ensures that the response contract is enforced at the framework level, preventing handlers from accidentally returning non-compliant structures.

```go
// internal/delivery/fiber/router/router.go
package router

import (
    "github.com/gofiber/fiber/v3"
    "opus/server/internal/delivery/fiber/response"
)

func New(...) *fiber.App {
    app := fiber.New(fiber.Config{
        ErrorHandler: func(c fiber.Ctx, err error) error {
            code := fiber.StatusInternalServerError
            if e, ok := err.(*fiber.Error); ok {
                code = e.Code
            }
            return response.Error(c, code, slugFromStatus(code), titleFromStatus(code), err.Error())
        },
    })
    return app
}
```

---

### 2.3 Handler Responsibilities

Handlers in `internal/delivery/fiber/handler/` are strictly responsible for:

1. Parsing incoming Fiber requests (`c.BodyParser`, `c.Params`, `c.Query`).
2. Calling the appropriate `internal/[feature]/` Service method.
3. Returning the result using the standardised `internal/delivery/fiber/response` helpers.

**No business logic** will exist within Fiber handlers. They remain thin translation layers between HTTP/Fiber constructs and the pure Go Service layer.

```go
// internal/delivery/fiber/handler/agent_handler.go
package handler

import (
    "fmt"
    "github.com/gofiber/fiber/v3"
    "opus/server/internal/delivery/fiber/response"
    "opus/server/internal/agent"
)

type AgentHandler struct {
    service *agent.Service
}

func NewAgentHandler(svc *agent.Service) *AgentHandler {
    return &AgentHandler{service: svc}
}

func (h *AgentHandler) GetAgent(c fiber.Ctx) error {
    id := c.Params("id")
    a, err := h.service.FindByID(c.Context(), id)
    if err != nil {
        return response.Error(c, fiber.StatusNotFound, "not-found", "Resource Not Found",
            fmt.Sprintf("Agent with ID %s does not exist.", id))
    }
    return response.OK(c, a)
}
```

---

### 2.4 Configuration Integration (ADR-002)

The Fiber application instance is configured using the centralised `internal/config` system established in ADR-002. The `cfg.Server` slice (`internal/config.ServerConfig`) is injected into the router at startup. No Fiber-specific configuration struct is introduced; the existing `ServerConfig` covers all required properties.

```go
// internal/config/model.go (relevant excerpt — defined in ADR-002)
type ServerConfig struct {
    Address   string `mapstructure:"address"    json:"address"    jsonschema:"default=:8080"`
    Debug     bool   `mapstructure:"debug"      json:"debug"`
    BodyLimit int    `mapstructure:"body_limit" json:"body_limit" jsonschema:"default=4194304,description=Max request body size in bytes"`
    Prefork   bool   `mapstructure:"prefork"    json:"prefork"    jsonschema:"description=Enable Fiber Prefork mode (production only)"`
}
```

```go
// internal/delivery/fiber/router/router.go
func New(cfg config.ServerConfig, ...) *fiber.App {
    app := fiber.New(fiber.Config{
        Prefork:     cfg.Prefork,
        BodyLimit:   cfg.BodyLimit,
        // ...
    })
    // register routes
    return app
}
```

```go
// main.go
app := router.New(cfg.Server, authHandler, agentHandler)
app.Listen(cfg.Server.Address)
```

---

### 2.5 Logger Middleware (ADR-006)

The request logger middleware in `internal/delivery/fiber/middleware/logger.go` **must** use the `logger.Logger` interface defined in ADR-006.

> **Constraint:** Direct use of `log/slog`, `fmt.Println`, or any concrete logging library is **prohibited** within the delivery layer. All logging must go through the injected `logger.Logger` interface.

The middleware is responsible for:

1. Generating a unique `request_id` per incoming request.
2. Injecting the `request_id` into the `context.Context` using `logger.RequestIDKey` (defined in ADR-006).
3. Logging the request and response summary via `logger.InfoCtx`.

```go
// internal/delivery/fiber/middleware/logger.go
package middleware

import (
    "github.com/gofiber/fiber/v3"
    "github.com/google/uuid"
    "opus/server/internal/shared/logger"
)

func RequestLogger(log logger.Logger) fiber.Handler {
    return func(c fiber.Ctx) error {
        requestID := uuid.NewString()
        ctx := context.WithValue(c.Context(), logger.RequestIDKey, requestID)
        c.SetUserContext(ctx)

        err := c.Next()

        log.InfoCtx(ctx, "request completed",
            logger.String("method", c.Method()),
            logger.String("path", c.Path()),
            logger.Int("status", c.Response().StatusCode()),
            logger.String("request_id", requestID),
        )
        return err
    }
}
```

---

## 3. Alternatives Considered

### 3.1 Standard Library `net/http` (Go 1.22+)

Go 1.22 introduced improved routing in the standard library. While excellent for minimal dependencies, it lacks the rich middleware ecosystem, zero-allocation optimisations, and built-in body parsing conveniences that GoFiber v3 provides natively.

---

## 4. Consequences

### 4.1 Positive

- **Explicit Boundaries:** The `internal/delivery/fiber` name correctly signals that everything inside is coupled to GoFiber v3, preventing accidental bleeding of Fiber context into the generic `internal` domains.
- **Performance:** GoFiber v3 provides excellent performance and modern Go generics support.
- **Contract Enforcement:** Fiber's global error handler uniformly enforces the ADR-004 response contract.
- **Observability:** Logger middleware enforces ADR-006 compliance at the framework boundary; all requests are traced with a `request_id`.
- **Config Cohesion:** Reusing `cfg.Server` from ADR-002 avoids a parallel config struct and keeps the schema generator accurate.

### 4.2 Negative / Trade-offs

- **Framework Lock-in:** The delivery layer is fully coupled to GoFiber v3. However, because business logic is isolated in the `internal/` service layer (ADR-001), this lock-in is contained entirely within the `internal/delivery/fiber/` boundary.

---

## 5. References

- [ADR-001: Server Clean Architecture](./ADR-001-server-clean-architecture.md)
- [ADR-002: Server Configuration](./ADR-002-server-configuration.md)
- [ADR-004: API Response Contract](./ADR-004-api-response-contract.md)
- [ADR-006: Server Logger Architecture](./ADR-006-server-logger.md)
- [GoFiber v3 Documentation](https://docs.gofiber.io/v3)