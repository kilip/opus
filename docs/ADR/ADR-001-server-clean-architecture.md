# ADR-001: Server Clean Architecture

**Status:** Accepted  
**Date:** 2026-05-17  
**Deciders:** Chief Architect  
**Context:** Opus Server (`opus/server/`)

---

## 1. Context

Opus is a self-hosted, autonomous AI assistant built as a **modular monolith** in Go. The server must support multiple domains (auth, agent, vault, workflow, scheduler, LLM routing), remain testable in isolation, and preserve the ability to extract individual modules into independent microservices in the future.

This ADR establishes the clean architecture pattern and directory structure for all Go server-side code under `opus/server/`.

---

## 2. Decision

Opus Server adopts a **Go-idiomatic, feature-based clean architecture** with explicit layer boundaries enforced through directory structure and Go interface contracts.

### 2.1 Directory Structure

> **Note for implementors and AI agents:** The directory structure below is **illustrative**.
> Concrete layer paths are determined by their respective ADRs (e.g. `delivery/fiber/` as
> defined in ADR-005, not `delivery/http/`). This ADR defines layer responsibilities,
> dependency rules, and architectural boundaries only — not literal folder names.

```
opus/
└── server/
    ├── main.go
    ├── internal/
    │   ├── config/             # Configuration loading (opus.yaml + env vars)
    │   ├── shared/             # Cross-cutting domain entities (User, Workspace)
    │   ├── auth/               # Authentication & authorization domain
    │   │   ├── model.go        # Domain models (Token, Claims, Session)
    │   │   ├── repository.go   # Repository interface (port)
    │   │   └── service.go      # Business logic (use cases)
    │   ├── agent/              # Agent lifecycle domain
    │   │   ├── model.go
    │   │   ├── repository.go
    │   │   └── service.go
    │   ├── vault/              # Vault read/write domain
    │   │   ├── model.go
    │   │   ├── repository.go
    │   │   └── service.go
    │   ├── workflow/           # Workflow execution domain
    │   │   ├── model.go
    │   │   ├── repository.go
    │   │   └── service.go
    │   ├── scheduler/          # Background job scheduling domain
    │   │   ├── model.go
    │   │   └── service.go
    │   └── llm/                # LLM abstraction domain
    │       ├── model.go        # CompletionRequest, CompletionResponse
    │       └── router.go       # LLM Router interface + provider resolution
    ├── adapter/
    │   └── entgo/              # Concrete repository implementations (ent ORM)
    │       ├── client.go       # Ent client setup + migration
    │       ├── auth_repo.go    # Implements auth.Repository
    │       ├── agent_repo.go   # Implements agent.Repository
    │       ├── vault_repo.go   # Implements vault.Repository
    │       └── workflow_repo.go
    └── delivery/
        ├── http/               # HTTP delivery layer (REST + SSE)
        │   ├── handler/        # Route handlers per domain
        │   │   ├── auth_handler.go
        │   │   ├── agent_handler.go
        │   │   └── vault_handler.go
        │   ├── middleware/     # Cross-cutting HTTP concerns
        │   │   ├── auth.go     # JWT validation middleware
        │   │   └── logger.go
        │   ├── router/         # Route registration + app bootstrap
        │   │   └── router.go
        │   └── sse/            # SSE connection manager
        │       └── manager.go
        └── grpc/               # gRPC delivery layer (future: inter-module)
```

### 2.2 Layer Responsibilities

| Layer | Path | Responsibility |
|---|---|---|
| **Domain** | `internal/[feature]/` | Business logic, domain models, repository interfaces |
| **Infrastructure** | `adapter/entgo/` | Concrete implementations of repository interfaces |
| **Delivery** | `delivery/http/`, `delivery/grpc/` | HTTP/gRPC handlers; translates requests to service calls |
| **Config** | `internal/config/` | Configuration parsing; injected at startup |
| **Shared** | `internal/shared/` | Cross-cutting domain entities used by multiple features |

### 2.3 Dependency Rule

Dependencies flow **inward only**:

```
delivery/ → internal/[feature]/ ← adapter/
                    ↑
              internal/shared/
              internal/config/
```

- `internal/[feature]/` has **zero knowledge** of delivery or adapter implementations
- `adapter/entgo/` imports `internal/[feature]/` interfaces — never the reverse
- `delivery/` imports `internal/[feature]/` services — never adapter directly

### 2.4 Repository Pattern

Each feature domain defines its own repository interface (port). The adapter layer provides the concrete implementation (adapter).

**Interface (port) — defined in domain:**

```go
// internal/auth/repository.go
package auth

import "context"

type Repository interface {
    FindByEmail(ctx context.Context, email string) (*User, error)
    Create(ctx context.Context, user *User) error
    UpdatePassword(ctx context.Context, userID string, hash string) error
}
```

**Implementation (adapter) — defined in adapter layer:**

```go
// adapter/entgo/auth_repo.go
package entgo

import (
    "context"
    "opus/server/internal/auth"
    "opus/server/adapter/entgo/ent"
)

type AuthRepo struct {
    client *ent.Client
}

func (r *AuthRepo) FindByEmail(ctx context.Context, email string) (*auth.User, error) {
    // entgo query implementation
}
```

### 2.5 Service Layer

Services contain pure business logic with no infrastructure dependencies. All infrastructure access goes through repository interfaces, injected at startup.

```go
// internal/auth/service.go
package auth

import (
    "context"
    "opus/server/internal/config"
)

type Service struct {
    repo   Repository
    config *config.Config
}

func NewService(repo Repository, cfg *config.Config) *Service {
    return &Service{repo: repo, config: cfg}
}

func (s *Service) Login(ctx context.Context, email, password string) (*TokenPair, error) {
    user, err := s.repo.FindByEmail(ctx, email)
    if err != nil {
        return nil, ErrInvalidCredentials
    }
    if !verifyPassword(password, user.PasswordHash) {
        return nil, ErrInvalidCredentials
    }
    return s.issueTokens(user)
}
```

### 2.6 Delivery Layer

Handlers translate HTTP requests into service calls. No business logic lives in handlers.

```go
// delivery/http/handler/auth_handler.go
package handler

import (
    "opus/server/internal/auth"
    "github.com/gofiber/fiber/v2"
)

type AuthHandler struct {
    service *auth.Service
}

func NewAuthHandler(svc *auth.Service) *AuthHandler {
    return &AuthHandler{service: svc}
}

func (h *AuthHandler) Login(c *fiber.Ctx) error {
    var req LoginRequest
    if err := c.BodyParser(&req); err != nil {
        return fiber.ErrBadRequest
    }
    tokens, err := h.service.Login(c.Context(), req.Email, req.Password)
    if err != nil {
        return fiber.NewError(fiber.StatusUnauthorized, err.Error())
    }
    return c.JSON(tokens)
}
```

### 2.7 Dependency Injection at Startup

All wiring happens in `main.go`. No global state, no service locators.

```go
// main.go
package main

import (
    "opus/server/adapter/entgo"
    "opus/server/delivery/http/handler"
    "opus/server/delivery/http/router"
    "opus/server/internal/auth"
    "opus/server/internal/config"
)

func main() {
    cfg := config.Load()

    // Adapter layer
    db := entgo.NewClient(cfg)
    authRepo := entgo.NewAuthRepo(db)

    // Service layer
    authService := auth.NewService(authRepo, cfg)

    // Delivery layer
    authHandler := handler.NewAuthHandler(authService)

    // Bootstrap
    app := router.New(authHandler)
    app.Listen(cfg.Server.Address)
}
```

### 2.8 Shared Models

`internal/shared/` contains domain entities that are genuinely cross-cutting — used by more than one feature domain. Discipline is required: entities must not be added here speculatively.

```go
// internal/shared/model.go
package shared

import "time"

type User struct {
    ID        string
    Username  string
    Email     string
    Role      string
    CreatedAt time.Time
}

type Workspace struct {
    ID        string
    Name      string
    VaultPath string
    CreatedAt time.Time
}
```

Feature-specific models (e.g. `auth.TokenPair`, `agent.Run`) remain in their respective feature packages.

---

## 3. Alternatives Considered

### 3.1 Layer-Based Structure (`internal/service/`, `internal/model/`, `internal/adapter/`)

Familiar to developers from Java/Spring backgrounds. Rejected because:

- Not idiomatic Go — Go community prefers package-by-feature over package-by-layer
- Cross-feature dependencies become implicit and hard to trace
- Does not naturally map to microservice extraction boundaries

### 3.2 Flat Package Structure

All code in a single `internal/` level without sub-packages. Rejected because:

- Does not scale beyond a small codebase
- No clear microservice extraction path
- Insufficient separation of concerns for a multi-domain system like Opus

---

## 4. Consequences

### 4.1 Positive

- **Testability** — Service layer has zero infrastructure dependencies; unit tests require only mock repositories
- **Swappable infrastructure** — Replacing entgo with another ORM, or SQLite with PostgreSQL, requires changes only in `adapter/entgo/`
- **Microservice extraction** — Each `internal/[feature]/` folder is a natural microservice boundary; extraction requires lifting the folder, replacing repository calls with gRPC clients, and promoting the adapter to a standalone DB
- **Go-idiomatic** — Package-by-feature aligns with standard Go project layout conventions
- **Clear onboarding** — Contributors navigate to a single feature folder to understand the full domain

### 4.2 Negative / Trade-offs

- **`internal/shared/` discipline required** — Without governance, `shared/` becomes a dumping ground for entities that should remain in their feature domain
- **Boilerplate at startup** — Explicit DI in `main.go` grows as modules are added; a DI container (e.g. `uber/fx`) may be considered in a future ADR if wiring complexity becomes unmanageable
- **gRPC delivery layer is a stub** — `delivery/grpc/` is reserved for future inter-module communication; it adds directory noise in v1 but is intentional

---

## 5. References
- [Standard Go Project Layout](https://github.com/golang-standards/project-layout)
- [Clean Architecture — Robert C. Martin](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)
- [Hexagonal Architecture (Ports & Adapters) — Alistair Cockburn](https://alistair.cockburn.us/hexagonal-architecture/)