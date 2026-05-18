# ADR-012: Module System and Dependency Injection

**Status:** Accepted
**Date:** 2026-05-18
**Deciders:** Chief Architect
**Context:** Opus Server (`opus/server/`)

---

## 1. Context

Opus Server is a modular monolith structured around Clean Architecture (ADR-001). As the system
grows to support multiple first-class feature domains — `auth`, `agent`, `vault`, `workflow` —
and a growing set of integration domains — `gmail`, `gdrive`, `gcalendar`, `whatsapp`,
`telegram`, `gitsync` — the wiring strategy established in ADR-001 of accumulating all
dependency construction in `main.go` becomes unscalable.

Each integration domain is a first-class feature domain with its own business logic, scheduled
processes, bot logic, and AI agent integration. They are not thin adapter wrappers.

Without a formal module system, the following problems emerge as domain count grows:

- `main.go` becomes a multi-hundred-line wiring file with no clear organisational structure.
- Adding a new domain requires editing `main.go` directly, increasing the risk of accidental
  regressions in unrelated wiring.
- There is no canonical location for domain-level bootstrap logic such as job handler
  registration, event subscription, and scheduled process setup.
- Circular import risks increase as inter-domain dependencies are wired ad-hoc.

This ADR establishes the canonical module system, dependency injection strategy, and bootstrap
pattern for all Go server-side code under `opus/server/`. It supersedes the `main.go` wiring
pattern described in ADR-001 for all domain construction and replaces it with a structured,
scalable alternative.

> **Note for AI agents and automated tooling:** This ADR is the authoritative specification for
> all dependency injection and module bootstrap conventions in `opus/server/`. Do not invent
> wiring patterns, container access patterns, or inter-domain dependency patterns beyond what is
> defined here. When in doubt, refer to this document.

---

## 2. Decision

Opus Server adopts a **structured manual dependency injection system** using a centralised
`internal/container/` package. Each feature domain owns its own `bootstrap.go` file that
encapsulates all domain-level initialisation. A single `container.Bootstrap()` function
orchestrates the initialisation order. All inter-domain communication is strictly via the
`queue.EventBus` interface — no direct imports between feature domains are permitted.

---

### 2.1 Core Principles

| Principle | Description |
|---|---|
| **No DI library** | Manual DI only — no `uber/fx`, `google/wire`, or equivalent |
| **Explicit deps** | Each domain bootstrap receives only the deps it needs — no full container pass-through |
| **Strict EventBus** | All inter-domain communication via `queue.EventBus` — no direct service imports between domains |
| **No circular imports** | Enforced structurally — domains never import each other |
| **Single entry point** | `container.Bootstrap(cfg)` is the only wiring call in `main.go` |
| **Domain owns bootstrap** | Each domain defines `[feature]/bootstrap.go` — registration logic lives with the domain |

---

### 2.2 Directory Structure

```
opus/
└── server/
    ├── main.go                         # Calls container.Bootstrap(cfg) only
    │
    └── internal/
        ├── container/
        │   ├── container.go            # Container struct + shared dep accessors
        │   └── bootstrap.go            # Bootstrap() — orchestrates all domain init
        │
        ├── agent/
        │   ├── bootstrap.go            # agent domain bootstrap
        │   ├── service.go
        │   ├── repository.go
        │   ├── model.go
        │   ├── errors.go
        │   └── config.go
        │
        ├── auth/
        │   ├── bootstrap.go
        │   └── ...
        │
        ├── vault/
        │   ├── bootstrap.go
        │   └── ...
        │
        ├── workflow/
        │   ├── bootstrap.go
        │   └── ...
        │
        ├── gmail/
        │   ├── bootstrap.go
        │   └── ...
        │
        ├── gdrive/
        │   ├── bootstrap.go
        │   └── ...
        │
        ├── whatsapp/
        │   ├── bootstrap.go
        │   └── ...
        │
        ├── telegram/
        │   ├── bootstrap.go
        │   └── ...
        │
        ├── gitsync/
        │   ├── bootstrap.go
        │   └── ...
        │
        └── delivery/
            └── gofiber/
                ├── bootstrap.go        # Fiber app bootstrap — registered last
                └── ...
```

---

### 2.3 Container

The `Container` struct holds all shared infrastructure dependencies. It is initialised once in
`container.Bootstrap()` and is never passed directly to domain bootstrap functions. Domains
receive only the specific shared deps they require as explicit function parameters.

```go
// internal/container/container.go
package container

import (
    "github.com/kilip/opus/server/ent"
    "github.com/kilip/opus/server/internal/auth"
    "github.com/kilip/opus/server/internal/agent"
    "github.com/kilip/opus/server/internal/vault"
    "github.com/kilip/opus/server/internal/workflow"
    "github.com/kilip/opus/server/internal/gmail"
    "github.com/kilip/opus/server/internal/gdrive"
    "github.com/kilip/opus/server/internal/whatsapp"
    "github.com/kilip/opus/server/internal/telegram"
    "github.com/kilip/opus/server/internal/gitsync"
    "github.com/kilip/opus/server/internal/shared/logger"
    "github.com/kilip/opus/server/internal/shared/queue"
    "github.com/gofiber/fiber/v3"
)

// Container holds all initialised shared infrastructure and domain services.
// It is package-private; external packages access services via typed getter functions.
var c *container

type container struct {
    // Shared infrastructure
    db     *ent.Client
    log    logger.Logger
    queue  queue.Queue
    bus    queue.EventBus

    // Domain services
    auth     *auth.Service
    agent    *agent.Service
    vault    *vault.Service
    workflow *workflow.Service
    gmail    *gmail.Service
    gdrive   *gdrive.Service
    whatsapp *whatsapp.Service
    telegram *telegram.Service
    gitsync  *gitsync.Service

    // Delivery
    fiber *fiber.App
}

// GetAuth returns the initialised auth.Service.
// Panics if Bootstrap has not been called.
func GetAuth() *auth.Service {
    mustInit()
    return c.auth
}

// GetAgent returns the initialised agent.Service.
// Panics if Bootstrap has not been called.
func GetAgent() *agent.Service {
    mustInit()
    return c.agent
}

// GetVault returns the initialised vault.Service.
// Panics if Bootstrap has not been called.
func GetVault() *vault.Service {
    mustInit()
    return c.vault
}

// GetWorkflow returns the initialised workflow.Service.
// Panics if Bootstrap has not been called.
func GetWorkflow() *workflow.Service {
    mustInit()
    return c.workflow
}

// GetGmail returns the initialised gmail.Service.
// Panics if Bootstrap has not been called.
func GetGmail() *gmail.Service {
    mustInit()
    return c.gmail
}

// GetGDrive returns the initialised gdrive.Service.
// Panics if Bootstrap has not been called.
func GetGDrive() *gdrive.Service {
    mustInit()
    return c.gdrive
}

// GetWhatsApp returns the initialised whatsapp.Service.
// Panics if Bootstrap has not been called.
func GetWhatsApp() *whatsapp.Service {
    mustInit()
    return c.whatsapp
}

// GetTelegram returns the initialised telegram.Service.
// Panics if Bootstrap has not been called.
func GetTelegram() *telegram.Service {
    mustInit()
    return c.telegram
}

// GetGitSync returns the initialised gitsync.Service.
// Panics if Bootstrap has not been called.
func GetGitSync() *gitsync.Service {
    mustInit()
    return c.gitsync
}

// GetFiber returns the initialised Fiber application.
// Panics if Bootstrap has not been called.
func GetFiber() *fiber.App {
    mustInit()
    return c.fiber
}

// mustInit panics if the container has not been initialised via Bootstrap.
func mustInit() {
    if c == nil {
        panic("container: Bootstrap has not been called")
    }
}
```

---

### 2.4 Bootstrap Orchestration

`container.Bootstrap()` is the single function responsible for initialising all shared
infrastructure and orchestrating domain bootstrap calls in dependency order.

```go
// internal/container/bootstrap.go
package container

import (
    "context"

    "github.com/kilip/opus/server/internal/adapter/entgo"
    adapterqueue "github.com/kilip/opus/server/internal/adapter/queue"
    "github.com/kilip/opus/server/internal/agent"
    "github.com/kilip/opus/server/internal/auth"
    "github.com/kilip/opus/server/internal/config"
    fiberdelivery "github.com/kilip/opus/server/internal/delivery/gofiber"
    "github.com/kilip/opus/server/internal/gdrive"
    "github.com/kilip/opus/server/internal/gmail"
    "github.com/kilip/opus/server/internal/gitsync"
    "github.com/kilip/opus/server/internal/telegram"
    "github.com/kilip/opus/server/internal/vault"
    "github.com/kilip/opus/server/internal/whatsapp"
    "github.com/kilip/opus/server/internal/workflow"
)

// Bootstrap initialises all shared infrastructure and domain services in dependency order.
// It must be called exactly once at application startup, before any service is accessed.
// Panics on any initialisation failure — startup errors are unrecoverable.
func Bootstrap(cfg *config.Config) {
    initShared(cfg)
    auth.Bootstrap(c.db, c.bus, c.queue, c.log, cfg.Auth)
    vault.Bootstrap(c.db, c.bus, c.queue, c.log, cfg.Vault)
    agent.Bootstrap(c.db, c.bus, c.queue, c.log, cfg.Agent)
    workflow.Bootstrap(c.db, c.bus, c.queue, c.log, cfg.Workflow)
    gmail.Bootstrap(c.db, c.bus, c.queue, c.log, cfg.Gmail)
    gdrive.Bootstrap(c.db, c.bus, c.queue, c.log, cfg.GDrive)
    whatsapp.Bootstrap(c.db, c.bus, c.queue, c.log, cfg.WhatsApp)
    telegram.Bootstrap(c.db, c.bus, c.queue, c.log, cfg.Telegram)
    gitsync.Bootstrap(c.db, c.bus, c.queue, c.log, cfg.GitSync)
    fiberdelivery.Bootstrap(c.fiber, c.log, cfg.Server)
}

// initShared initialises all shared infrastructure dependencies.
// Must be called before any domain bootstrap function.
func initShared(cfg *config.Config) {
    c = &container{}

    var err error

    c.log, err = initLogger(cfg.Log)
    if err != nil {
        panic("container: failed to initialise logger: " + err.Error())
    }

    c.db, err = entgo.NewClient(cfg.Database)
    if err != nil {
        panic("container: failed to connect to database: " + err.Error())
    }

    if err = entgo.AutoMigrate(c.db, context.Background()); err != nil {
        panic("container: failed to run schema migration: " + err.Error())
    }

    c.queue, err = adapterqueue.NewQueue(cfg.Queue)
    if err != nil {
        panic("container: failed to initialise queue: " + err.Error())
    }

    c.bus = adapterqueue.NewEventBus()
    c.fiber = initFiber(cfg.Server)
}
```

---

### 2.5 Domain Bootstrap Convention

Each domain defines a `bootstrap.go` file at `internal/[feature]/bootstrap.go`. The bootstrap
function receives only the shared deps the domain requires as explicit parameters. It is
responsible for:

1. Constructing the domain repository via the Ent adapter.
2. Constructing the domain service.
3. Registering job handlers on the `queue.Queue`.
4. Subscribing to domain events on the `queue.EventBus`.
5. Storing the initialised service in the container via an unexported setter.

```go
// internal/agent/bootstrap.go
package agent

import (
    "github.com/kilip/opus/server/ent"
    "github.com/kilip/opus/server/internal/adapter/entgo"
    "github.com/kilip/opus/server/internal/shared/logger"
    "github.com/kilip/opus/server/internal/shared/queue"
)

// Bootstrap initialises the agent domain: repository, service, job handlers,
// and event subscriptions. Called by container.Bootstrap() during startup.
func Bootstrap(
    db  *ent.Client,
    bus queue.EventBus,
    q   queue.Queue,
    log logger.Logger,
    cfg Config,
) {
    repo := entgo.NewAgentRepo(db)
    svc  := NewService(repo, q, bus, log, cfg)

    // Register job handlers — must be called before queue.Start().
    q.RegisterHandler("agent:evaluate", svc.HandleEvaluateJob)
    q.RegisterHandler("agent:retry",    svc.HandleRetryJob)

    // Subscribe to domain events from other domains.
    // agent never imports vault, workflow, gmail, etc.
    bus.Subscribe("vault.written",      svc.OnVaultWritten)
    bus.Subscribe("workflow.completed", svc.OnWorkflowCompleted)

    setService(svc)
}

// svc is the package-level service instance, accessible via GetService().
var svc *Service

// setService stores the initialised service. Called exclusively by Bootstrap.
func setService(s *Service) { svc = s }

// GetService returns the initialised agent.Service.
// Panics if Bootstrap has not been called.
func GetService() *Service {
    if svc == nil {
        panic("agent: Bootstrap has not been called")
    }
    return svc
}
```

**Gmail domain bootstrap example — subscribes to agent events without importing agent:**

```go
// internal/gmail/bootstrap.go
package gmail

import (
    "github.com/kilip/opus/server/ent"
    "github.com/kilip/opus/server/internal/adapter/entgo"
    "github.com/kilip/opus/server/internal/shared/logger"
    "github.com/kilip/opus/server/internal/shared/queue"
)

// Bootstrap initialises the gmail domain: repository, service, job handlers,
// and event subscriptions. Called by container.Bootstrap() during startup.
func Bootstrap(
    db  *ent.Client,
    bus queue.EventBus,
    q   queue.Queue,
    log logger.Logger,
    cfg Config,
) {
    repo := entgo.NewGmailRepo(db)
    svc  := NewService(repo, q, bus, log, cfg)

    // Register job handlers.
    q.RegisterHandler("gmail:send",    svc.HandleSendJob)
    q.RegisterHandler("gmail:sync",    svc.HandleSyncJob)
    q.RegisterHandler("gmail:process", svc.HandleProcessJob)

    // Subscribe to domain events.
    // gmail does not import agent — it reacts to published events only.
    bus.Subscribe("agent.completed",   svc.OnAgentCompleted)
    bus.Subscribe("vault.written",     svc.OnVaultWritten)

    setService(svc)
}
```

---

### 2.6 Delivery Layer Bootstrap

The Fiber delivery layer is bootstrapped last, after all domain services are initialised. It
receives only the services it needs to register routes.

```go
// internal/delivery/gofiber/bootstrap.go
package gofiber

import (
    "github.com/gofiber/fiber/v3"
    "github.com/kilip/opus/server/internal/agent"
    "github.com/kilip/opus/server/internal/auth"
    "github.com/kilip/opus/server/internal/shared/logger"
    "github.com/kilip/opus/server/internal/vault"
    "github.com/kilip/opus/server/internal/workflow"
)

// Bootstrap registers all HTTP routes on the provided Fiber app.
// Must be called after all domain Bootstrap functions have completed.
func Bootstrap(
    app *fiber.App,
    log logger.Logger,
    cfg Config,
) {
    registerRoutes(
        app,
        auth.GetService(),
        agent.GetService(),
        vault.GetService(),
        workflow.GetService(),
        log,
    )
}
```

---

### 2.7 main.go — Final Form

With this architecture, `main.go` is reduced to its essential responsibilities: loading
configuration, calling `Bootstrap`, starting the queue worker loop, and starting the HTTP server.

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

### 2.8 Inter-Domain Communication — Strict EventBus Rule

All communication between feature domains is performed exclusively via `queue.EventBus`. Direct
service-to-service calls across domain boundaries are prohibited.

**Dependency rule:**

```
internal/[feature_a]/  →  internal/[feature_b]/   ❌  PROHIBITED
internal/[feature_a]/  →  internal/shared/queue/  ✅  via EventBus only
internal/[feature_b]/  →  internal/shared/queue/  ✅  via EventBus only
```

**Correct inter-domain pattern:**

```go
// agent/service.go — publishes an event; does not import gmail, workflow, etc.
func (s *Service) completeRun(ctx context.Context, runID string) error {
    // ... business logic ...

    return s.bus.Publish(ctx, queue.Event{
        Topic:   "agent.completed",
        Payload: payload,
        Source:  "agent",
    })
}

// gmail/service.go — reacts to agent events; does not import agent
func (s *Service) OnAgentCompleted(ctx context.Context, event queue.Event) error {
    // ... handle agent completion ...
    return nil
}
```

**Event topic registry** (extends ADR-008):

| Topic | Producer | Consumers |
|---|---|---|
| `agent.completed` | `agent` | `gmail`, `workflow`, `telegram`, `whatsapp` |
| `agent.failed` | `agent` | `gmail`, `telegram`, `whatsapp` |
| `agent.started` | `agent` | `workflow` |
| `vault.written` | `vault` | `agent`, `gmail`, `gdrive`, `gitsync` |
| `workflow.completed` | `workflow` | `agent`, `gmail`, `telegram` |
| `gmail.received` | `gmail` | `agent`, `vault` |
| `gdrive.changed` | `gdrive` | `agent`, `vault`, `gitsync` |
| `gitsync.pushed` | `gitsync` | `agent`, `vault` |
| `telegram.message` | `telegram` | `agent` |
| `whatsapp.message` | `whatsapp` | `agent` |

---

### 2.9 Adding a New Domain

Adding a new integration domain requires exactly four steps with no changes to existing domain
code:

1. Create `internal/[feature]/` with `model.go`, `repository.go`, `service.go`, `errors.go`,
   `config.go`, and `bootstrap.go`.
2. Add `[feature].Config` to `internal/config/model.go`.
3. Add the Ent schema to `server/ent/schema/[feature].go` and run `go generate ./ent/...`.
4. Add one line to `container/bootstrap.go`:
   ```go
   [feature].Bootstrap(c.db, c.bus, c.queue, c.log, cfg.[Feature])
   ```

No changes to `main.go`, no changes to any existing domain, no changes to the delivery layer
(unless the new domain exposes HTTP endpoints).

---

### 2.10 Alignment with Existing ADRs

| Convention | This ADR | Related ADR |
|---|---|---|
| Repository interface in domain | `[feature]/repository.go` | ADR-001 |
| Explicit dep injection | `Bootstrap(db, bus, q, log, cfg)` | ADR-001, ADR-002 |
| EventBus for async decoupling | `bus.Subscribe` / `bus.Publish` | ADR-008 |
| Job handler registration | `q.RegisterHandler` | ADR-008 |
| Logger interface injection | `log logger.Logger` param | ADR-006 |
| Feature config ownership | `[feature]/config.go` | ADR-002 |
| No global logger / no `log.Fatal` | All init errors via `panic` in `main.go` only | ADR-010 |
| GoDoc on all exported symbols | All exported funcs documented | ADR-010 |

---

## 3. Alternatives Considered

### 3.1 Accumulate All Wiring in `main.go` (ADR-001 Original Pattern)

Keep the existing pattern of constructing all deps in `main.go`. Rejected because:

- Does not scale beyond ~5 domains; `main.go` becomes unmanageable at 10+ domains.
- No canonical location for domain-level bootstrap logic (job handler registration, event
  subscriptions, scheduled process setup).
- Adding a domain requires engineers to understand the full wiring order in `main.go` rather
  than a self-contained `[feature]/bootstrap.go`.

### 3.2 DI Container Library (`uber/fx`, `google/wire`)

Use a DI library to auto-wire dependencies via reflection or code generation. Rejected because:

- `uber/fx` introduces runtime reflection-based wiring that obscures the dependency graph and
  makes debugging startup failures harder.
- `google/wire` requires an additional code generation step and adds toolchain complexity.
- Manual DI is idiomatic Go and is sufficient at the scale Opus will reach.
- The explicit `Bootstrap(db, bus, q, log, cfg)` signature is self-documenting and compiler-
  verified; no external tooling is required to understand what a domain needs.

### 3.3 Pass Full Container to Each Domain Bootstrap

Pass `*container.Container` as a single parameter to each domain bootstrap function, giving
each domain access to all shared deps and all other services. Rejected because:

- Violates the Interface Segregation Principle — domains gain access to deps they do not need.
- Undermines the strict inter-domain isolation goal; a domain could trivially access another
  domain's service via the container, bypassing the EventBus rule.
- Harder to test — bootstrapping a domain for a unit test would require constructing a full
  container rather than passing only the required deps.

### 3.4 Single `di/` Package with Flat Files (`di/auth.go`, `di/agent.go`)

Place all domain bootstrap logic in a flat `internal/di/` package. Rejected in favour of
`[feature]/bootstrap.go` because:

- `internal/di/auth.go` is separated from the `auth` domain it bootstraps; a contributor
  navigating `internal/auth/` would not find the bootstrap logic there.
- `[feature]/bootstrap.go` keeps all domain concerns co-located, consistent with the
  feature-based clean architecture from ADR-001.
- A domain carrying its own `bootstrap.go` is fully self-contained during microservice
  extraction.

---

## 4. Consequences

### 4.1 Positive

- **`main.go` remains minimal** — three meaningful lines: load config, bootstrap, start server.
- **Circular imports impossible by design** — domains never import each other; all cross-domain
  communication is via EventBus, which is defined in `internal/shared/queue/`.
- **New domain in four steps** — adding an integration domain requires no changes to existing
  code beyond `container/bootstrap.go` and `internal/config/model.go`.
- **Self-contained domains** — each domain's `bootstrap.go` documents exactly what it depends
  on, what jobs it handles, and what events it subscribes to — in one place.
- **Microservice extraction path** — a domain carries its own `bootstrap.go`, `config.go`, and
  event subscriptions; extraction to a standalone service requires replacing the EventBus with
  a network-capable message broker and the DB client with a gRPC or REST client.
- **Testable in isolation** — domain bootstrap functions accept explicit deps; unit tests pass
  in-memory or mock implementations without constructing a full container.
- **Consistent with ADR-001** — the explicit constructor injection pattern established for
  services (`NewService(repo, log, cfg)`) is extended uniformly to bootstrap functions.

### 4.2 Negative / Trade-offs

- **Manual ordering discipline** — `container/bootstrap.go` must declare domains in the correct
  initialisation order. This is a programmer responsibility enforced by convention, not by the
  compiler. Incorrect ordering results in a runtime panic at startup, which is detectable
  immediately.
- **Package-level `var svc *Service`** — each domain uses a package-level variable to store its
  initialised service for access via `GetService()`. This is a controlled use of package state,
  consistent with Go's standard library conventions for package-level initialisation (e.g.
  `database/sql` drivers, `flag` package). It is acceptable here because the variable is set
  exactly once at startup and never mutated thereafter.
- **`panic` at startup** — initialisation failures in `container/bootstrap.go` result in
  panics. This is consistent with ADR-010 (panic permitted in `main.go` for unrecoverable
  startup failures) and is the correct behaviour for a self-hosted server — a misconfigured
  startup should fail loudly rather than silently degrade.
- **`container/bootstrap.go` grows with domain count** — each new domain adds one line to
  `container/bootstrap.go`. This is intentional and acceptable; the file remains a linear,
  readable orchestration manifest rather than a complex wiring graph.

---

## 5. References

- [ADR-001: Server Clean Architecture](./ADR-001-server-clean-architecture.md)
- [ADR-002: Configuration Management](./ADR-002-server-configuration.md)
- [ADR-005: Server Delivery Layer with GoFiber v3](./ADR-005-server-delivery-layer-with-gofiber-v3.md)
- [ADR-006: Server Logger Architecture](./ADR-006-server-logger.md)
- [ADR-007: ORM and Database Strategy](./ADR-007-orm-and-database-strategy.md)
- [ADR-008: Server Queue Architecture](./ADR-008-server-queue.md)
- [ADR-009: Server Testing Strategy](./ADR-009-server-testing-strategy.md)
- [ADR-010: Server Coding Conventions & Linting](./ADR-010-server-coding-and-linting.md)
- [Effective Go — Package Initialisation](https://go.dev/doc/effective_go#init)
- [Go Code Review Comments — Package Names](https://github.com/golang/go/wiki/CodeReviewComments#package-names)