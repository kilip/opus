# ADR-007: ORM and Database Strategy

**Status:** Accepted
**Date:** 2026-05-17
**Deciders:** Chief Architect
**Context:** Opus Server (`opus/server/`)

---

## 1. Context

Opus Server requires a persistent storage layer for multiple feature domains: `auth`, `agent`, `vault`, and `workflow`. The ORM must:

- Support both **SQLite** (zero-configuration local development and single-user deployments) and **PostgreSQL** (team and production deployments) from a single codebase
- Provide **type-safe, code-generated** query builders to eliminate stringly-typed SQL at compile time
- Integrate cleanly with the **feature-based Clean Architecture** established in ADR-001 — repository interfaces defined in the domain layer, implementations isolated in the adapter layer
- Handle **schema migrations** safely and reproducibly with minimal operator intervention — a critical requirement for a self-hosted system where users expect reliable upgrades without manual database management

SQLite is appropriate for Opus's local-first, single-user model. With WAL (Write-Ahead Logging) mode enabled, SQLite handles concurrent reads and serialised writes without contention at the access patterns Opus produces. It imposes zero infrastructure dependency for the default deployment.

---

## 2. Decision

Opus Server adopts **Ent** as the exclusive ORM for all database interactions, with **Atlas** for schema migration management.

- Generated code resides in `server/ent/` following Entgo's own project convention
- Repository implementations reside in `internal/adapter/entgo/` and implement the domain interfaces defined in `internal/[feature]/repository.go`
- Atlas **auto-migrate** is used in development; Atlas **versioned migrations** are used in production

---

### 2.1 Directory Structure

```
opus/
└── server/
    ├── ent/                            # Entgo generated code (entgo convention)
    │   ├── schema/                     # Schema definitions (hand-authored)
    │   │   ├── user.go
    │   │   ├── agent.go
    │   │   ├── vault_entry.go
    │   │   └── workflow.go
    │   ├── migrate/
    │   │   └── migrations/             # Atlas versioned migration files
    │   │       ├── 20260517000001_init.sql
    │   │       └── atlas.sum
    │   ├── client.go                   # Generated Ent client
    │   ├── ent.go                      # Generated package entry
    │   └── ...                         # Other generated files (never edit manually)
    │
    └── internal/
        └── adapter/
            └── entgo/
                ├── client.go               # Ent client setup, driver selection, migration bootstrap
                ├── auth.go                 # Implements internal/auth.Repository
                ├── agent.go                # Implements internal/agent.Repository
                ├── vault.go                # Implements internal/vault.Repository
                └── workflow.go             # Implements internal/workflow.Repository
```

> **Note for AI agents and implementors:** The `server/ent/` directory is **entirely generated** — except `server/ent/schema/`. Never manually edit generated files outside `schema/`. All schema changes begin in `server/ent/schema/` and are propagated via `go generate`.

---

### 2.2 Database Driver Support

Both SQLite and PostgreSQL are supported through Ent's driver abstraction. The active driver is selected at runtime based on the `database.driver` configuration field (ADR-002).

```go
// internal/adapter/entgo/client.go
package entgo

import (
    "fmt"

    "entgo.io/ent/dialect"
    entsql "entgo.io/ent/dialect/sql"
    _ "github.com/mattn/go-sqlite3"
    _ "github.com/lib/pq"
    "github.com/kilip/opus/server/ent"
    "github.com/kilip/opus/server/internal/config"
)

// NewClient opens a database connection and returns a configured Ent client.
// SQLite WAL mode is applied automatically when the sqlite3 driver is selected.
func NewClient(cfg config.DatabaseConfig) (*ent.Client, error) {
    drv, err := entsql.Open(cfg.Driver, cfg.DSN)
    if err != nil {
        return nil, fmt.Errorf("entgo: open database: %w", err)
    }

    // Enable WAL mode for SQLite to support concurrent reads
    if cfg.Driver == dialect.SQLite {
        if _, err := drv.DB().Exec("PRAGMA journal_mode=WAL;"); err != nil {
            return nil, fmt.Errorf("entgo: enable WAL mode: %w", err)
        }
    }

    return ent.NewClient(ent.Driver(drv)), nil
}
```

**Supported driver values (`database.driver`):**

| Value | Database | Notes |
|---|---|---|
| `sqlite3` | SQLite | Default; WAL mode applied automatically |
| `postgres` | PostgreSQL | Requires a valid PostgreSQL DSN |

---

### 2.3 Migration Strategy

Opus uses **Atlas** for migration management, operated in two modes depending on context:

#### Development — Auto-Migrate

During development, `client.Schema.Create()` is called at startup to apply any pending schema changes automatically. This eliminates migration friction during iterative schema work.

```go
// internal/adapter/entgo/client.go (development bootstrap)
func AutoMigrate(client *ent.Client, ctx context.Context) error {
    return client.Schema.Create(ctx)
}
```

#### Production — Versioned Migrations

For production deployments, Atlas generates deterministic SQL migration files. Operators review and apply migrations explicitly. Versioned migration files are committed to the repository at `server/ent/migrate/migrations/`.

**Generate a new migration:**

```bash
# From server/
go run -mod=mod entgo.io/ent/cmd/ent generate ./ent/schema
atlas migrate diff <migration_name> \
  --dir "file://ent/migrate/migrations" \
  --to "ent://ent/schema" \
  --dev-url "sqlite://dev?mode=memory&cache=shared&_fk=1"
```

**Apply migrations (production):**

```bash
atlas migrate apply \
  --dir "file://ent/migrate/migrations" \
  --url "${DATABASE_URL}"
```

The `atlas.sum` file in `server/ent/migrate/migrations/` provides integrity verification for migration history and must be committed alongside migration files.

---

### 2.4 Repository Pattern

Each feature domain defines its own repository interface (port) in `internal/[feature]/repository.go`. The `internal/adapter/entgo/` package provides the concrete implementation. This boundary is identical to the pattern established in ADR-001.

**Interface (domain layer):**

```go
// internal/agent/repository.go
package agent

import "context"

// Repository defines the persistence contract for the Agent domain.
type Repository interface {
    FindByID(ctx context.Context, id string) (*Agent, error)
    FindAll(ctx context.Context, cursor string, limit int) ([]*Agent, string, error)
    Create(ctx context.Context, agent *Agent) (*Agent, error)
    UpdateStatus(ctx context.Context, id string, status Status) error
    Delete(ctx context.Context, id string) error
}
```

**Implementation (adapter layer):**

```go
// internal/adapter/entgo/agent.go
package entgo

import (
    "context"
    "github.com/kilip/opus/server/ent"
    "github.com/kilip/opus/server/internal/agent"
)

// AgentRepo implements agent.Repository using Ent.
type AgentRepo struct {
    client *ent.Client
}

// NewAgentRepo constructs an AgentRepo.
func NewAgentRepo(client *ent.Client) *AgentRepo {
    return &AgentRepo{client: client}
}

// FindByID retrieves an agent by its ID.
func (r *AgentRepo) FindByID(ctx context.Context, id string) (*agent.Agent, error) {
    row, err := r.client.Agent.Get(ctx, id)
    if err != nil {
        if ent.IsNotFound(err) {
            return nil, agent.ErrNotFound
        }
        return nil, err
    }
    return mapAgentFromEnt(row), nil
}
```

**Dependency rule (consistent with ADR-001):**

```
internal/[feature]/repository.go  →  defines interface (port)
internal/adapter/entgo/[feature].go        →  implements interface (adapter)
internal/adapter/entgo imports internal/    ✅
internal/ never imports adapter/   ✅
internal/ never imports ent/       ✅  (domain is ORM-agnostic)
```

---

### 2.5 Schema Definition Convention

All Ent schemas are defined in `server/ent/schema/`. Each schema file corresponds to one domain entity. Schemas use `entgo.io/ent` field and edge declarations; they must not embed domain types from `internal/`.

```go
// ent/schema/agent.go
package schema

import (
    "entgo.io/ent"
    "entgo.io/ent/schema/field"
)

// Agent holds the schema definition for the Agent entity.
type Agent struct {
    ent.Schema
}

// Fields defines the Agent entity fields.
func (Agent) Fields() []ent.Field {
    return []ent.Field{
        field.String("id").Unique().Immutable(),
        field.String("name").NotEmpty(),
        field.Enum("status").Values("idle", "running", "completed", "errored").Default("idle"),
        field.Time("created_at").Immutable(),
        field.Time("updated_at"),
    }
}
```

**Code generation is triggered via `go generate`:**

```go
// server/ent/generate.go
//go:generate go run -mod=mod entgo.io/ent/cmd/ent generate ./schema
package ent
```

Run with:

```bash
cd server && go generate ./ent/...
```

---

### 2.6 Configuration Integration (ADR-002)

Database configuration follows the Hybrid Config Composition Pattern from ADR-002. The `DatabaseConfig` struct is defined in `internal/config/model.go` and injected into `internal/adapter/entgo/client.go` at startup.

```go
// internal/config/model.go (excerpt)
type DatabaseConfig struct {
    Driver string `mapstructure:"driver" json:"driver" jsonschema:"enum=sqlite3,enum=postgres,default=sqlite3,description=Database driver"`
    DSN    string `mapstructure:"dsn"    json:"dsn"    jsonschema:"description=Data source name. Inject via OPUS_DATABASE_DSN for production secrets"`
}
```

**Example `config.json` values:**

```json
{
  "database": {
    "driver": "sqlite3",
    "dsn":    "opus.db"
  }
}
```

```json
{
  "database": {
    "driver": "postgres",
    "dsn":    "host=localhost port=5432 dbname=opus sslmode=disable"
  }
}
```

Production DSN must be injected via the `OPUS_DATABASE_DSN` environment variable. It must never be written to a config file on disk.

---

### 2.7 Dependency Injection at Startup

The Ent client is constructed once in `main.go` and injected into every repository constructor. No global Ent client variable is used.

```go
// main.go
func main() {
    cfg, err := config.Load()
    if err != nil {
        log.Fatal("config load failed", err)
    }

    entClient, err := entgo.NewClient(cfg.Database)
    if err != nil {
        log.Fatal("database connection failed", err)
    }
    defer entClient.Close()

    // Development only — replace with versioned migrations in production
    if err := entgo.AutoMigrate(entClient, context.Background()); err != nil {
        log.Fatal("schema migration failed", err)
    }

    // Repository layer
    agentRepo  := entgo.NewAgentRepo(entClient)
    authRepo   := entgo.NewAuthRepo(entClient)
    vaultRepo  := entgo.NewVaultRepo(entClient)

    // Service layer
    agentService := agent.NewService(agentRepo, cfg.Agent)
    // ...
}
```

---

## 3. Alternatives Considered

### 3.1 GORM

The most widely adopted Go ORM. Rejected because:

- Relies on `interface{}` / `any` in its API surface — reduces compile-time type safety
- Magic conventions (auto-timestamps, soft deletes) create implicit behaviour that conflicts with Opus's explicit, auditable domain model
- Code generation is not first-class; schema is defined by struct tags rather than a dedicated schema DSL

### 3.2 sqlx + raw SQL

Thin wrapper over `database/sql` with struct scanning. Rejected because:

- Requires hand-authoring all SQL queries — significant boilerplate for CRUD-heavy domains
- No built-in migration tooling; requires an additional dependency (e.g. `golang-migrate`)
- Schema changes require coordinated updates to multiple SQL files with no compile-time verification

### 3.3 sqlc

Generates type-safe Go code from raw SQL queries. Not chosen over Ent because:

- Requires maintaining raw SQL query files alongside Go code — two sources of truth for schema and queries
- Less ergonomic for relationship traversal (edges) compared to Ent's edge DSL
- Atlas integration is not as seamless as with Ent's native toolchain

### 3.4 Offset-Based Pagination

Considered for repository list methods. Rejected in favour of cursor-based pagination (consistent with ADR-004) because:

- Offset pagination produces unstable results under concurrent inserts — critical for agent log and workflow run streams
- `OFFSET N` queries perform full index scans up to offset N; cursor pagination uses keyset seeks and scales linearly

---

## 4. Consequences

### 4.1 Positive

- **Type safety end-to-end** — Ent's code generation produces fully typed query builders; invalid queries are caught at compile time
- **Single codebase, two databases** — Driver abstraction allows switching between SQLite and PostgreSQL via a single config value with no application code changes
- **SQLite WAL mode** — Enables concurrent reads for the default single-user deployment without contention
- **Atlas migration safety** — Versioned migrations provide an auditable, reviewable migration history; `atlas.sum` integrity checks prevent history tampering
- **ORM-agnostic domain layer** — `internal/[feature]/` has no Ent dependency; replacing the ORM requires changes only in `internal/adapter/entgo/` and `server/ent/`
- **Entgo convention compliance** — Placing generated code in `server/ent/` follows Entgo's documented project layout, reducing onboarding friction for contributors familiar with the framework

### 4.2 Negative / Trade-offs

- **Code generation discipline** — `server/ent/` (except `schema/`) must never be edited manually; enforced by convention. A CI lint step checking for manual edits to generated files is recommended
- **SQLite concurrent write limitation** — WAL mode serialises writes; under high-concurrency write workloads (not expected for a personal assistant), SQLite will become a bottleneck. The migration path to PostgreSQL is one config change
- **Atlas toolchain dependency** — Versioned migrations require the Atlas CLI to be installed in the development and CI environment; documented in project setup instructions
- **`go-sqlite3` CGO dependency** — `github.com/mattn/go-sqlite3` requires CGO to be enabled at build time; cross-compilation to targets without a C toolchain requires additional build configuration (e.g. `modernc.org/sqlite` as a pure-Go alternative)
- **Ent edge verbosity** — Complex relationship queries using Ent edges can be more verbose than equivalent raw SQL; mitigated by isolating complex queries within the adapter layer

---

## 5. References

- [ADR-001: Server Clean Architecture](./ADR-001-server-clean-architecture.md)
- [ADR-002: Configuration Management](./ADR-002-server-configuration.md)
- [ADR-004: API Response Contract](./ADR-004-api-response-contract.md)
- [Ent — entgo.io](https://entgo.io)
- [Atlas — atlasgo.io](https://atlasgo.io)
- [go-sqlite3 — github.com/mattn/go-sqlite3](https://github.com/mattn/go-sqlite3)
- [lib/pq — github.com/lib/pq](https://github.com/lib/pq)
- [SQLite WAL Mode — sqlite.org](https://www.sqlite.org/wal.html)
