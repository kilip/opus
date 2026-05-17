# Opus

Opus is an open-source, self-hosted, autonomous AI assistant designed to operate 24/7 on behalf of individuals and teams. It unifies a structured knowledge base, a programmable workflow engine, and a proactive AI agent layer into a single platform.

---

## Project Structure

```
opus/
├── server/     # Go backend — REST/SSE API
├── dash/       # Progressive Web Application (PWA) frontend
├── docs/       # Architecture Decision Records (ADRs)
└── get-opus/   # Installer — npx get-opus
```

---

## Technology Stack

### Backend (`server/`)

| Concern | Choice |
|---|---|
| Language | Go |
| HTTP Framework | GoFiber v3 |
| ORM | Ent + Atlas migrations |
| Config | Viper + JSON + env vars |
| Queue | SQLite (default) / PostgreSQL / Redis (Asynq) |
| Logging | `internal/shared/logger` interface (injected) |
| Mocking | `go.uber.org/mock` |

### Frontend (`dash/`)

| Concern | Choice |
|---|---|
| Language | TypeScript |
| Framework | React 19.x + Vite 6.x |
| Routing | TanStack Router |
| Server State | TanStack Query |
| Styling | Tailwind CSS 4.x + shadcn/ui |
| PWA | vite-plugin-pwa (Workbox) |
| Testing | Vitest + React Testing Library + Playwright (E2E) |

---

## Backend Architecture (ADR-001, ADR-005)

### Directory Layout

```
server/
├── main.go
├── ent/                        # Entgo generated code (never edit except ent/schema/)
│   └── schema/                 # Hand-authored Ent schema definitions
├── internal/
│   ├── config/                 # Config loader (Viper)
│   ├── shared/
│   │   ├── logger/             # Logger interface + NoopLogger + MockLogger
│   │   └── queue/              # Queue + EventBus interfaces + Noop* + Mock*
│   ├── adapter/
│   │   ├── entgo/              # Concrete repository implementations
│   │   └── queue/              # Queue backend implementations (sqlite, postgres, redis, memory)
│   ├── agent/                  # Domain: models, repository interface, service, errors, config
│   ├── delivery/
│   │   └── gofiber/            # Canonical HTTP delivery layer (NOT delivery/http/)
│   │       ├── handler/
│   │       ├── middleware/
│   │       ├── router.go       # Route registration + app bootstrap
│   │       ├── response.go     # ADR-004 envelope helpers
│   │       └── config.go       # GoFiber configuration struct (hybrid composition)
│   ├── vault/
│   ├── workflow/
│   ├── auth/
│   ├── llm/
│   └── testutil/               # Shared test helpers (NewTestEntClient, fixtures)
```

### Dependency Rule

```
internal/delivery/gofiber/ → internal/[feature]/ ← internal/adapter/
```

- `internal/[feature]/` has zero knowledge of delivery or adapter implementations.
- `internal/adapter/` imports `internal/[feature]/` interfaces — never the reverse.
- `internal/delivery/gofiber/` imports `internal/` services — never adapter directly.

### Layer Responsibilities

| Layer | Responsibility |
|---|---|
| `internal/[feature]/` | Domain models, business logic, repository interfaces, sentinel errors, feature config |
| `internal/adapter/entgo/` | Concrete repository implementations (Ent) |
| `internal/adapter/queue/` | Queue backend implementations |
| `internal/delivery/gofiber/` | HTTP handlers, middleware, routing — thin translation layer only |

---

## Coding Conventions (ADR-010)

### GoDoc
Every exported symbol **must** have a GoDoc comment starting with the symbol name, ending with a period.

### Error Handling
- Sentinel errors in `internal/[feature]/errors.go`, prefixed `Err`, checked via `errors.Is`.
- Wrap all infrastructure errors: `fmt.Errorf("agent.Service.FindByID: %w", err)`.
- `panic` is **prohibited** in domain/service/adapter/handler code.
- `log.Fatal` / `os.Exit` are **prohibited** outside `main.go`.

### Interfaces
- Defined in the **consumer** package (`internal/[feature]/`), never in the adapter.
- No `I` prefix (`IRepository` is invalid).
- Carry `//go:generate mockgen` directive above the declaration.

### Context
- `context.Context` is always the first parameter, named `ctx`.
- Never store `context.Context` in a struct field.
- Context enrichment (tracing metadata) happens only in delivery middleware.

### Logging
- Direct use of `fmt.Println`, `log.Print*`, or any concrete logging library is **prohibited** outside `main.go`.
- All logging goes through the injected `logger.Logger` interface.
- Sensitive values use `logger.Redact`.
- Log messages: lowercase, present tense, no trailing punctuation.

### Naming
- Packages: lowercase, single word, no underscores.
- Files: snake_case (`agent.go`, `mock_repository.go`).
- Errors: `internal/[feature]/errors.go`.
- Mocks: `mock_<interface>.go` (generated, never edited manually).

### Import Grouping (enforced by `goimports`)
```go
import (
    // 1. Standard library
    "context"
    "fmt"

    // 2. Third-party
    "github.com/gofiber/fiber/v3"

    // 3. Internal
    "github.com/kilip/opus/server/internal/agent"
)
```

### Linting
- `golangci-lint` with `server/.golangci.yml` — zero warning policy.
- Run: `cd server && golangci-lint run ./...`

---

## Configuration (ADR-002)

**Resolution order (highest → lowest priority):**
1. `OPUS_*` environment variables
2. `$OPUS_HOME/config.json`
3. `~/.opus/config.json`
4. `./.opus/config.json` (development)

**Hybrid composition:** each feature owns its config struct in `internal/[feature]/config.go`; root `internal/config/model.go` composes them. Features never import the root `config` package.

Secrets (API keys, DSNs) via env vars only — never in config files.

---

## API Contract (ADR-004)

- All responses use the envelope: `{ "data": ..., "error": ..., "meta": ... }`.
- Errors follow RFC 7807 Problem Details inside `error`.
- URL structure: `/{resource}` — **no prefix**.
- Pagination: cursor-based only (no offset).
- SSE endpoint: `GET /agents/{id}/logs/stream`.
- Response helpers: `internal/delivery/gofiber/response.go` — use `gofiber.OK`, `gofiber.Error`, etc.

---

## Queue & Events (ADR-008)

- **`queue.Queue`** — durable background jobs (agent tasks, emails, vault indexing).
- **`queue.EventBus`** — in-process pub/sub for domain decoupling (no persistence).
- Job type convention: `"<domain>:<action>"` (e.g. `"agent:evaluate"`).
- Event topic convention: `"<domain>.<action>"` (e.g. `"agent.completed"`).
- Use `queue.NoopQueue` / `queue.NoopEventBus` in unit tests.

---

## Testing Strategy (ADR-009)

| Category | File Suffix | Build Tag | Infrastructure |
|---|---|---|---|
| Unit | `_test.go` | _(none)_ | Mocks / Noop* only |
| Integration | `_integration_test.go` | `//go:build integration` | SQLite in-memory |

- All unit tests use the **table-driven** pattern.
- Default package style: `package foo_test` (black-box); white-box (`package foo`) requires a justification comment.
- Mocks via `go.uber.org/mock` only — `testify/mock` is **prohibited**.
- Shared test helpers: `internal/testutil/` — call `t.Helper()` as first statement.
- Handler tests use Fiber's `app.Test()` and **must** assert the ADR-004 envelope shape.
- Coverage thresholds: 80% per unit-tested package, 70% per adapter package.

**Run commands:**
```bash
go test -race ./...                        # unit tests
go test -race -tags integration ./...      # unit + integration
go generate ./...                          # regenerate mocks after interface changes
```

---

## Frontend Architecture (ADR-003)

```
dash/src/
├── app/           # Entry point, router, global providers
├── routes/        # TanStack Router file-based page components (thin — no business logic)
├── features/
│   ├── agent/     # components/, hooks/, api.ts, types.ts
│   ├── vault/
│   └── workflow/
└── shared/
    ├── components/ # Layout, OfflineBanner, shadcn/ui wrappers
    ├── hooks/      # useNetworkStatus, useServiceWorkerUpdate, useTheme
    ├── lib/        # api-client.ts, utils.ts
    └── types/      # ApiEnvelope<T>, ProblemDetail, PaginationMeta
```

**Rules:**
- Features import from `shared/` only — never from sibling features.
- Cross-feature data is resolved at the route level.
- All API calls go through `shared/lib/api-client.ts` — no raw `fetch` in components.

**Offline strategy:**
- `GET` requests: `NetworkFirst` with 5s timeout, falls back to cache.
- Mutations: `BackgroundSyncPlugin` queue (IndexedDB); optimistic updates via TanStack Query `onMutate`.
- Offline state: `useNetworkStatus()` → `OfflineBanner` in root layout.

**Frontend code quality:**
- Every exported function/component: JSDoc comment.
- Every component: Vitest + React Testing Library test.
- Critical flows: Playwright E2E test.

---

## Conventional Commits

**Format:** `<type>(<scope>): <description>`
**Length:** `<description>` must be max 70 characters. Use the commit body for a more detailed explanation if necessary.
**Scopes:** `server`, `dash`, `get-opus`, `ci`, `deps`

**Example:**
```text
feat(dash): integrate biome and replace eslint

- Remove eslint and prettier dependencies
- Add biome configurations
- Update npm scripts to use biome for linting and formatting
- Fix existing formatting issues based on biome rules
```

| Prefix | Version Bump |
|---|---|
| `feat:` | minor |
| `fix:` | patch |
| `feat!:` / `fix!:` | major |
| `chore:`, `ci:`, `refactor:` | patch |

---

## Agent Memory System

Memory lives in `.agents/` (gitignored). **Every session gets its own file.**

```
.agents/
├── MEMORY.md                        # Long-term: decisions, project context
└── memory/
    ├── YYYY-MM-DD-<slug>.md         # One file per session
    └── YYYY-MM-DD-consolidated.md  # Auto-generated consolidated summary
```

### Session Workflow

**Start:** Read `MEMORY.md` + last 3–5 session files → create new session file → write goal and open questions.

**During:** Update session file with decisions, findings, blockers.

**End:** Finalise session file → promote lasting decisions to `MEMORY.md`.

**Consolidation** (when 5+ non-consolidated files exist): summarise into `YYYY-MM-DD-consolidated.md` (decisions, progress, blockers, key findings — no code snippets), then delete source files. Never consolidate the current session's file.