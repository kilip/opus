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

## Backend Architecture (ADR-001, ADR-005, ADR-012)

### Directory Layout

```
server/
├── main.go                         # Calls container.Bootstrap(cfg) only
├── ent/                            # Entgo generated code (never edit except ent/schema/)
│   └── schema/
└── internal/
    ├── container/
    │   ├── container.go            # Container struct + typed getter functions
    │   └── bootstrap.go            # Bootstrap() — orchestrates all domain init
    ├── config/                     # Config loader (Viper)
    ├── shared/
    │   ├── logger/                 # Logger interface + NoopLogger + MockLogger
    │   └── queue/                  # Queue + EventBus interfaces + Noop* + Mock*
    ├── adapter/
    │   ├── entgo/                  # Concrete repository implementations
    │   └── queue/                  # Queue backend implementations (sqlite, postgres, redis, memory)
    ├── agent/                      # bootstrap.go, model.go, repository.go, service.go, errors.go, config.go
    ├── auth/                       # bootstrap.go, model.go, repository.go, service.go, errors.go, config.go
    ├── vault/                      # bootstrap.go, ...
    ├── workflow/                   # bootstrap.go, ...
    ├── gmail/                      # bootstrap.go, ...
    ├── gdrive/                     # bootstrap.go, ...
    ├── whatsapp/                   # bootstrap.go, ...
    ├── telegram/                   # bootstrap.go, ...
    ├── gitsync/                    # bootstrap.go, ...
    ├── llm/                        # model.go, router.go, config.go
    ├── delivery/
    │   └── gofiber/                # bootstrap.go, handler/, middleware/, router.go, response.go, config.go
    └── testutil/                   # NewTestEntClient, fixtures
```

### Dependency Rule

```
internal/delivery/gofiber/ → internal/[feature]/ ← internal/adapter/
                    ↑                  ↑
              internal/shared/   internal/container/
```

- `internal/[feature]/` — zero knowledge of delivery, adapter, or container.
- `internal/adapter/` — imports `internal/[feature]/` interfaces only.
- `internal/delivery/gofiber/` — imports `internal/[feature]/` via `GetService()`.
- `internal/container/` — only package permitted to import all domains simultaneously.
- **Feature domains never import each other** — all cross-domain communication via `queue.EventBus`.

### Bootstrap Pattern (ADR-012)

`main.go` calls `container.Bootstrap(cfg)` only. Each domain owns `[feature]/bootstrap.go`.

```go
// main.go
func main() {
    cfg, err := config.Load()
    if err != nil { panic("config load failed: " + err.Error()) }
    container.Bootstrap(cfg)
    ctx := context.Background()
    if err := container.GetQueue().Start(ctx); err != nil { panic(err) }
    if err := container.GetFiber().Listen(cfg.Server.Address); err != nil { panic(err) }
}
```

```go
// internal/agent/bootstrap.go
func Bootstrap(db *ent.Client, bus queue.EventBus, q queue.Queue, log logger.Logger, cfg Config) {
    repo := entgo.NewAgentRepo(db)
    svc  := NewService(repo, q, bus, log, cfg)
    q.RegisterHandler("agent:evaluate", svc.HandleEvaluateJob)
    bus.Subscribe("vault.written", svc.OnVaultWritten)
    setService(svc)
}
```

Adding a new domain = 4 steps: create `internal/[feature]/`, add config, add Ent schema, add one line to `container/bootstrap.go`.

---

## Coding Conventions (ADR-010)

- **GoDoc** — every exported symbol, starts with symbol name, ends with period.
- **Errors** — sentinel in `errors.go` with `Err` prefix; wrap with `fmt.Errorf("pkg.Type.Method: %w", err)`; check with `errors.Is`.
- **No `panic`** outside `main.go`; no `log.Fatal`/`os.Exit` outside `main.go`.
- **Interfaces** — defined in consumer package; no `I` prefix; carry `//go:generate mockgen` directive.
- **Context** — `ctx context.Context` always first param; never stored in struct.
- **Logging** — injected `logger.Logger` only; sensitive values via `logger.Redact`; messages lowercase, present tense, no trailing punctuation.
- **Imports** — stdlib → third-party → internal (enforced by `goimports`).
- **Linting** — `golangci-lint` with `server/.golangci.yml`; zero warning policy.

---

## Configuration (ADR-002)

**Resolution order:** env vars → `$OPUS_HOME/config.json` → `~/.opus/config.json` → `./.opus/config.json`

- Each feature owns its config struct in `internal/[feature]/config.go`.
- Root `internal/config/model.go` composes all feature configs.
- Features never import the root `config` package.
- Secrets via `OPUS_*` env vars only — never in config files.

---

## API Contract (ADR-004)

- Envelope: `{ "data": ..., "error": ..., "meta": ... }` on every response.
- Errors: RFC 7807 Problem Details inside `error`.
- URLs: `/{resource}` — **no version prefix**.
- Pagination: cursor-based only.
- SSE: `GET /agents/{id}/logs/stream`.
- Helpers: `internal/delivery/gofiber/response.go` — use `gofiber.OK`, `gofiber.Error`, etc.

---

## Queue & Events (ADR-008)

- **`queue.Queue`** — durable background jobs; type convention `"<domain>:<action>"` e.g. `"agent:evaluate"`.
- **`queue.EventBus`** — in-process pub/sub; topic convention `"<domain>.<action>"` e.g. `"agent.completed"`.
- Use `queue.NoopQueue` / `queue.NoopEventBus` in unit tests.

---

## Testing Strategy (ADR-009)

| Category | File Suffix | Build Tag | Infrastructure |
|---|---|---|---|
| Unit | `_test.go` | _(none)_ | Mocks / Noop* only |
| Integration | `_integration_test.go` | `//go:build integration` | SQLite in-memory |

- Table-driven tests required for all exported functions.
- Default: `package foo_test` (black-box); white-box requires justification comment.
- Mocks via `go.uber.org/mock` only — `testify/mock` prohibited.
- Shared helpers in `internal/testutil/` — call `t.Helper()` first.
- Handler tests use `app.Test()` and must assert ADR-004 envelope shape.
- Coverage: 80% per unit-tested package, 70% per adapter package.

```bash
go test -race ./...                    # unit tests
go test -race -tags integration ./...  # unit + integration
go generate ./...                      # regenerate mocks after interface changes
```

---

## Frontend Architecture (ADR-003)

```
dash/src/
├── app/           # Entry point, router, global providers
├── routes/        # TanStack Router file-based pages (thin — no business logic)
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

- Features import from `shared/` only — never from sibling features.
- All API calls via `shared/lib/api-client.ts` — no raw `fetch` in components.
- Offline reads: `NetworkFirst` + cache fallback. Offline writes: `BackgroundSyncPlugin` queue.

---

## Conventional Commits

**Format:** `<type>(<scope>): <description>` — description max 70 chars.

**Scopes:** `server`, `dash`, `get-opus`, `ci`, `deps`

| Prefix | Version Bump |
|---|---|
| `feat:` | minor |
| `fix:` | patch |
| `feat!:` / `fix!:` | major |
| `chore:`, `ci:`, `refactor:` | patch |

---

## Agent Memory System

Memory lives in `.agents/` (gitignored). Every session gets its own file.

```
.agents/
├── MEMORY.md                        # Long-term: decisions, project context
└── memory/
    ├── sessions/                    # Active session files
    │   └── YYYY-MM-DD-<slug>.md
    └── consolidated/                # Historical consolidated summaries
        └── YYYY-MM-DD.md
```

### Decision Format

All decisions in session files and `MEMORY.md` use this format:

```
[YYYY-MM-DDTHH:MM:SSZ] [domain.topic]: decision
```

Examples:
```
[2026-05-18T10:00:00Z] [database.driver]: Use SQLite as default
[2026-05-18T14:00:00Z] [queue.backend]: Use Asynq for Redis backend
[2026-05-18T14:30:00Z] [auth.token]: Use stateful JWT with refresh rotation
```

**Key convention:** `[domain.topic]` — domain first, then topic. Examples: `[database.driver]`, `[auth.token]`, `[queue.backend]`, `[delivery.framework]`.

### Conflict Resolution

When promoting decisions to `MEMORY.md`:
1. Search `MEMORY.md` for entries with the same `[domain.topic]` key.
2. **Latest timestamp wins** — overwrite older entry.
3. To explicitly override an older decision, add a `supersedes` note:
   ```
   [2026-05-19T09:00:00Z] [database.driver]: Use PostgreSQL in production (supersedes 2026-05-18T10:00:00Z)
   ```

### Session Workflow

**Start:** Read `MEMORY.md` + last 3–5 session files → create new session file → write goal and open questions.

**Consolidation** when 10+ non-consolidated session files exist: summarise into `memory/consolidated/YYYY-MM-DD.md` (decisions, progress, blockers — no code snippets), delete source files. Never consolidate the current session's file.

**During:** Record decisions using `[timestamp] [domain.topic]: decision` format. Update session file with findings and blockers.

**End:** Promote lasting decisions to `MEMORY.md` (latest timestamp wins per key) → finalise session file.