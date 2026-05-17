# ADR-005: Delivery Layer using GoFiber v3

**Status:** Proposed  
**Date:** 2026-05-17  
**Deciders:** Chief Architect  
**Context:** Opus Server (`opus/server/`)

---

## 1. Context

Opus Server requires a robust, high-performance HTTP framework for its Delivery Layer. The framework must align cleanly with our strict architectural boundaries (Clean Architecture, ADR-001) and our strict API response contracts (ADR-004). We are evaluating GoFiber v3 to fulfill this role.

## 2. Decision

We will adopt **GoFiber v3** as the exclusive framework for the HTTP Delivery Layer in Opus Server. We will structure the delivery layer explicitly under **`delivery/fiber/`**.

### 2.1 Directory Structure

The delivery layer will be explicitly named `fiber`:

```
opus/
└── server/
    └── delivery/
        └── fiber/              # GoFiber v3 delivery layer
            ├── handler/        # Route handlers per domain
            │   ├── auth_handler.go
            │   └── user_handler.go
            ├── middleware/     # Cross-cutting Fiber middleware
            │   ├── auth.go
            │   └── logger.go
            ├── router/         # Route registration + app bootstrap
            │   └── router.go
            └── response/       # Fiber-specific response wrappers (ADR-004)
                └── response.go
```

### 2.2 Global Error Handling and Response Contract

GoFiber v3 allows setting a custom `ErrorHandler`. We will implement a global Fiber error handler that automatically catches any `fiber.Error` or unhandled `error`, and formats it into the strict RFC 7807 Problem Details envelope mandated by ADR-004.

This ensures that our response contract is enforced at the framework level, preventing handlers from accidentally returning non-compliant structures.

### 2.3 Handler Responsibilities

Handlers in `delivery/fiber/handler/` are strictly responsible for:
1. Parsing incoming Fiber requests (`c.BodyParser`, `c.Params`, `c.Query`).
2. Calling the appropriate `internal/[feature]/` Service method.
3. Returning the result using the standardized `delivery/fiber/response` helpers.

**No business logic** will exist within Fiber handlers. They remain thin translation layers between HTTP/Fiber constructs and our pure Go Service layer.

### 2.4 Configuration (Integration with ADR-002)

The Fiber application instance (`fiber.New(fiber.Config{...})`) will be configured using our centralized configuration system (`internal/config` via Viper), as outlined in ADR-002. Properties such as server port, read/write timeouts, body limits, and environment-specific settings (e.g., enabling/disabling Prefork) must be injected from this configuration layer rather than hardcoded in the delivery layer.

## 3. Alternatives Considered

### 3.1 Standard Library `net/http` (Go 1.22+)
Go 1.22 introduced improved routing in the standard library. While excellent for minimal dependencies, it lacks the rich middleware ecosystem, zero-allocation optimisations, and built-in body parsing conveniences that GoFiber v3 provides natively.

## 4. Consequences

### 4.1 Positive
- **Explicit Boundaries:** The `delivery/fiber` name correctly signals that everything inside is coupled to GoFiber v3, preventing accidental bleeding of Fiber context into the generic `internal` domains.
- **Performance:** GoFiber v3 provides excellent performance and modern Go generics support, improving developer experience.
- **Contract Enforcement:** Fiber's global error handler perfectly aligns with our need to uniformly enforce ADR-004.

### 4.2 Negative / Trade-offs
- **Framework Lock-in:** The delivery layer is fully coupled to GoFiber v3. However, because our business logic is isolated in the `internal/` service layer (ADR-001), this lock-in is isolated entirely to the `delivery/fiber/` boundary.

## 5. References
- [ADR-001: Server Clean Architecture](./ADR-001-server-clean-architecture.md)
- [ADR-002: Server Configuration](./ADR-002-server-configuration.md)
- [ADR-004: API Response Contract](./ADR-004-api-response-contract.md)
- [GoFiber v3 Documentation](https://docs.gofiber.io/v3)
