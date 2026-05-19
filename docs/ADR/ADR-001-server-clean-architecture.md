# ADR-001: Server Clean Architecture

**Status:** Accepted  
**Date:** 2026-05-17  
**Last Revised:** 2026-05-18  
**Deciders:** Chief Architect  
**Context:** Opus Server (`opus/server/`)

---

## 1. Context

Opus is a self-hosted, autonomous AI assistant built as a **modular monolith** in Go. The server
must support multiple first-class feature domains (mulai dengan `auth`, dan direncanakan untuk `agent`, `vault`, `workflow`) serta
seterusnya untuk integrasi (`gmail`, `gdrive`, `gcalendar`, `whatsapp`, `telegram`,
`gitsync`), tetap dapat diuji secara terisolasi, dan mempertahankan kemampuan untuk mengekstrak modul
individu menjadi microservices independen di masa depan.

This ADR establishes the clean architecture pattern and directory structure for all Go server-side
code under `opus/server/`. Dependency injection and bootstrap conventions are governed by
**ADR-012**, which supersedes the `main.go` wiring pattern previously described in Section 2.7
of this document.

---

## 2. Decision

Opus Server adopts a **Go-idiomatic, feature-based clean architecture** with explicit layer
boundaries enforced through directory structure and Go interface contracts. All dependency
construction and domain initialisation is delegated to the module system defined in ADR-012.

---

### 2.1 Directory Structure

> **Note for implementors and AI agents:** The directory structure below is **definitive** for sample purpose only.
> Concrete layer paths are determined by their respective ADRs (e.g. `internal/delivery/gofiber/`
> as defined in ADR-005). This ADR defines layer responsibilities, dependency rules, and
> architectural boundaries. Dependency injection and bootstrap wiring are defined in ADR-012.

```
opus/
└── server/
    ├── main.go                             # Calls container.Bootstrap(cfg) only — see ADR-012
    ├── ent/                                # Entgo generated code (never edit except ent/schema/)
    │   └── schema/                         # Hand-authored Ent schema definitions
    │
    └── internal/
        ├── container/
        │   ├── container.go                # Container struct + typed getter functions
        │   └── bootstrap.go                # Bootstrap() — orchestrates all domain init
        │
        ├── config/                         # Configuration loading (ADR-002)
        │
        ├── shared/
        │   ├── logger/                     # Logger interface + NoopLogger + MockLogger (ADR-006)
        │   └── queue/                      # Queue + EventBus interfaces + Noop* + Mock* (ADR-008)
        │
        ├── adapter/
        │   ├── entgo/                      # Concrete repository implementations (Ent ORM)
        │   │   ├── client.go               # Ent client setup, driver selection, migration
        │   │   └── auth.go                 # Implements internal/auth.Repository
        │   └── queue/                      # Queue backend implementations (ADR-008)
        │       ├── sqlite/
        │       ├── postgres/
        │       ├── redis/
        │       ├── memory/                 # In-process EventBus
        │       └── factory.go
        │
        ├── auth/
        │   ├── bootstrap.go                # Domain bootstrap: repo, service, handlers, events
        │   ├── model.go
        │   ├── repository.go               # Repository interface (port)
        │   ├── service.go
        │   ├── config.go
        │   ├── errors.go
        │   └── mock_repository.go          # Generated — DO NOT EDIT
        │
        |── delivery/
            └── gofiber/                    # HTTP delivery layer (REST + SSE) — ADR-005
                ├── bootstrap.go            # Registers all routes; bootstrapped last
                ├── handler/                # Route handlers per domain
                │   └── auth.go
                ├── middleware/             # Cross-cutting HTTP concerns
                │   ├── auth.go             # JWT validation middleware
                │   ├── rbac.go             # Casbin enforcement middleware
                │   └── logger.go
                ├── router.go               # Route registration
                ├── response.go             # ADR-004 envelope helpers
                └── config.go
```

---

### 2.2 Layer Responsibilities

| Layer | Path | Responsibility |
|---|---|---|
| **Domain** | `internal/[feature]/` | Business logic, domain models, repository interfaces, sentinel errors, feature config, bootstrap |
| **Container** | `internal/container/` | Shared infrastructure construction; typed service accessors; bootstrap orchestration |
| **Infrastructure** | `internal/adapter/entgo/` | Concrete implementations of repository interfaces |
| **Queue Adapters** | `internal/adapter/queue/` | Queue backend implementations (SQLite, PostgreSQL, Redis) |
| **Delivery** | `internal/delivery/gofiber/` | HTTP/SSE handlers; translates requests to service calls |
| **Config** | `internal/config/` | Configuration parsing; injected at startup |
| **Shared** | `internal/shared/` | Cross-cutting infrastructure interfaces (Logger, Queue, EventBus) |

---

### 2.3 Dependency Rule

Dependencies flow **inward only**:

```
internal/delivery/gofiber/ → internal/[feature]/ ← internal/adapter/
                    ↑                  ↑
              internal/shared/   internal/container/
              internal/config/
```

- `internal/[feature]/` has **zero knowledge** of delivery, adapter, or container implementations.
- `internal/adapter/` imports `internal/[feature]/` interfaces — never the reverse.
- `internal/delivery/gofiber/` imports `internal/[feature]/` services via `GetService()` — never adapter directly.
- `internal/container/` is the only package permitted to import all domain packages simultaneously.
- **Feature domains never import each other.** All cross-domain communication is exclusively via `queue.EventBus` (ADR-008).

```
internal/[feature_a]/  →  internal/[feature_b]/   ❌  PROHIBITED
internal/[feature_a]/  →  internal/shared/queue/  ✅  via EventBus only
```

---

### 2.4 Repository Pattern

Each feature domain defines its own repository interface (port) in `internal/[feature]/repository.go`. The `internal/adapter/entgo/` package provides the concrete implementation (adapter). This boundary is identical across all first-class and integration domains.

**Interface (port) — defined in domain:**

```go
// internal/auth/repository.go
package auth

import "context"

//go:generate mockgen -destination=mock_repository.go -package=auth . Repository

// Repository defines the persistence contract for the Auth domain.
type Repository interface {
    FindUserByID(ctx context.Context, id string) (*User, error)
    FindUserByEmail(ctx context.Context, email string) (*User, error)
    CreateUserWithOrganization(ctx context.Context, user *User, account *Account, organizationName string) (*User, error)
    // ...
}
```

**Implementation (adapter) — defined in adapter layer:**

```go
// internal/adapter/entgo/auth.go
package entgo

import (
    "context"

    "github.com/kilip/opus/server/ent"
    "github.com/kilip/opus/server/internal/auth"
)

// AuthRepo implements auth.Repository using Ent.
type AuthRepo struct {
    client *ent.Client
}

// NewAuthRepo constructs an AuthRepo.
func NewAuthRepo(client *ent.Client) *AuthRepo {
    return &AuthRepo{client: client}
}

// FindUserByID retrieves a user by its unique identifier.
// Returns auth.ErrUserNotFound if no user with the given ID exists.
func (r *AuthRepo) FindUserByID(ctx context.Context, id string) (*auth.User, error) {
    row, err := r.client.User.Get(ctx, id)
    if err != nil {
        if ent.IsNotFound(err) {
            return nil, auth.ErrUserNotFound
        }
        return nil, fmt.Errorf("entgo.AuthRepo.FindUserByID: %w", err)
    }
    return mapUserFromEnt(row), nil
}
```

**Dependency rule:**

```
internal/[feature]/repository.go     →  defines interface (port)
internal/adapter/entgo/[feature].go  →  implements interface (adapter)
internal/adapter/entgo imports internal/    ✅
internal/ never imports adapter/            ✅
internal/ never imports ent/                ✅  (domain is ORM-agnostic)
```

---

### 2.5 Service Layer

Services contain pure business logic with no infrastructure dependencies. All infrastructure access goes through repository interfaces and the `queue.EventBus`, injected via the domain `bootstrap.go`.

```go
// internal/auth/service.go
package auth

import (
    "context"

    "github.com/kilip/opus/server/internal/shared/logger"
    "github.com/kilip/opus/server/internal/shared/queue"
)

// Service handles all business logic for the Auth domain.
type Service struct {
    repo   Repository
    bus    queue.EventBus
    q      queue.Queue
    logger logger.Logger
    cfg    Config
}

// NewService constructs a new Service with the provided dependencies.
func NewService(repo Repository, q queue.Queue, bus queue.EventBus, log logger.Logger, cfg Config) *Service {
    return &Service{
        repo:   repo,
        bus:    bus,
        q:      q,
        logger: log.With(logger.String("component", "auth_service")),
        cfg:    cfg,
    }
}

// FindUserByID retrieves a User by its unique identifier.
func (s *Service) FindUserByID(ctx context.Context, id string) (*User, error) {
    u, err := s.repo.FindUserByID(ctx, id)
    if err != nil {
        return nil, fmt.Errorf("auth.Service.FindUserByID: %w", err)
    }
    return u, nil
}
```

---

### 2.6 Domain Bootstrap Convention

Each domain owns a `bootstrap.go` file that encapsulates all domain-level initialisation:
repository construction, service construction, job handler registration, and event subscription.
Bootstrap functions are called exclusively by `container.Bootstrap()` in `internal/container/bootstrap.go`.

```go
// internal/auth/bootstrap.go
package auth

import (
    "github.com/kilip/opus/server/ent"
    "github.com/kilip/opus/server/internal/adapter/entgo"
    "github.com/kilip/opus/server/internal/shared/logger"
    "github.com/kilip/opus/server/internal/shared/queue"
)

// Bootstrap initialises the auth domain.
func Bootstrap(
    r   Repository,
    bus queue.EventBus,
    q   queue.Queue,
    log logger.Logger,
    cfg Config,
) {
    // In real implementation, repository might be passed as an interface
    repo = r
    svc  := NewService(repo, q, bus, log, cfg)

    // Example handler and subscription
    q.RegisterHandler("auth:send_welcome_email", svc.HandleSendWelcomeEmail)
    bus.Subscribe("user.created", svc.OnUserCreated)

    setService(svc)
}

var svc *Service

func setService(s *Service) { svc = s }

// GetService returns the initialised auth.Service.
func GetService() *Service {
    if svc == nil {
        panic("auth: service not initialized")
    }
    return svc
}
```

> **See ADR-012** for the complete bootstrap convention, container structure, and
> `container.Bootstrap()` orchestration.

---

### 2.7 Dependency Injection at Startup

All dependency construction and wiring is performed by `container.Bootstrap()`. `main.go` is
reduced to loading configuration, calling `Bootstrap`, starting the queue, and starting the
HTTP server.

```go
// main.go
package main

import (
    "context"

    "github.com/kilip/opus/server/internal/config"
    "github.com/kilip/opus/server/internal/container"
)

func main() {
    cfg, err := config.Load()
    if err != nil {
        panic("config load failed: " + err.Error())
    }

    container.Bootstrap(cfg)

    ctx := context.Background()
    if err := container.GetQueue().Start(ctx); err != nil {
        panic("queue start failed: " + err.Error())
    }

    if err := container.GetFiber().Listen(cfg.Server.Address); err != nil {
        panic("server start failed: " + err.Error())
    }
}
```

---

### 2.8 Delivery Layer

Handlers translate HTTP requests into service calls. No business logic lives in handlers.
Services are accessed via the domain `GetService()` accessor, not injected directly.

```go
// internal/delivery/gofiber/handler/auth.go
package handler

import (
    "fmt"

    "github.com/gofiber/fiber/v3"
    "github.com/kilip/opus/server/internal/auth"
    "github.com/kilip/opus/server/internal/delivery/gofiber"
)

// Auth handles HTTP requests for the Auth domain.
type Auth struct {
    service *auth.Service
}

// NewAuth constructs an Auth handler with the provided service.
func NewAuth(svc *auth.Service) *Auth {
    return &Auth{service: svc}
}

// GetCurrentUser handles GET /auth/me.
func (h *Auth) GetCurrentUser(c fiber.Ctx) error {
    // Implementation details...
    return gofiber.OK(c, user)
}
```

---

### 2.9 Shared Models

`internal/shared/` contains only infrastructure interfaces and utilities that are genuinely
cross-cutting. Domain entity types used by more than one feature domain are placed in the
lower-level domain package and accessed via the EventBus payload — never imported directly by
a sibling domain.

```go
// internal/shared/model.go
package shared

import "time"

// User represents an authenticated Opus user.
type User struct {
    ID        string
    Username  string
    Email     string
    Role      string
    CreatedAt time.Time
}
```

Feature-specific models (e.g. `auth.Token`, `auth.Session`) remain in their respective feature
packages.

---

## 3. Alternatives Considered

### 3.1 Layer-Based Structure (`internal/service/`, `internal/model/`, `internal/adapter/`)

Familiar to developers from Java/Spring backgrounds. Rejected because:

- Not idiomatic Go — Go community prefers package-by-feature over package-by-layer.
- Cross-feature dependencies become implicit and hard to trace.
- Does not naturally map to microservice extraction boundaries.

### 3.2 Flat Package Structure

All code in a single `internal/` level without sub-packages. Rejected because:

- Does not scale beyond a small codebase.
- No clear microservice extraction path.
- Insufficient separation of concerns for a multi-domain system dengan banyak integrasi.

### 3.3 Pass Full Container to Domains

Passing `*container.Container` to each domain bootstrap to give access to all shared deps and
other services. Rejected in ADR-012 karena melanggar Interface Segregation Principle.

---

## 4. Consequences

### 4.1 Positive

- **Testability** — Service layer has zero infrastructure dependencies; unit tests require only mock repositories.
- **Swappable infrastructure** — Replacing entgo or any queue backend requires changes only in `internal/adapter/`.
- **Microservice extraction** — Each `internal/[feature]/` folder carries its own `bootstrap.go`, `config.go`, and event subscriptions.
- **Go-idiomatic** — Package-by-feature aligns with standard Go project layout conventions.
- **Circular imports impossible** — Feature domains never import each other; all cross-domain communication is via EventBus.

### 4.2 Negative / Trade-offs

- **`internal/shared/` discipline required** — Without governance, `shared/` becomes a dumping ground.
- **`container/bootstrap.go` ordering** — Bootstrap call order must be maintained manually.
- **`GetService()` package-level state** — Each domain uses a package-level `var svc *Service`.

---

## 5. References

- [ADR-002: Configuration Management](./ADR-002-server-configuration.md)
- [ADR-004: API Response Contract](./ADR-004-api-response-contract.md)
- [ADR-005: Server Delivery Layer with GoFiber v3](./ADR-005-server-delivery-layer-with-gofiber-v3.md)
- [ADR-006: Server Logger Architecture](./ADR-006-server-logger.md)
- [ADR-007: ORM and Database Strategy](./ADR-007-orm-and-database-strategy.md)
- [ADR-008: Server Queue Architecture](./ADR-008-server-queue.md)
- [ADR-009: Server Testing Strategy](./ADR-009-server-testing-strategy.md)
- [ADR-010: Server Coding Conventions & Linting](./ADR-010-server-coding-and-linting.md)
- [ADR-012: Module System and Dependency Injection](./ADR-012-module-system-and-dependency-injection.md)