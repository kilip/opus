# AGENTS.md

> Guidance for AI agents working on the Opus codebase.

---

## Project Overview

Opus is a self-hosted, single-user AI agent platform. Monorepo structure:

- `api/` — Go backend (GoFiber v3, EntGo, Viper, Cobra)
- `dash/` — Next.js 16 frontend (PWA, TanStack Query, Serwist, Shadcn/ui)
- `installer/` — npx installer (Node.js)
- `.agents/` — Agent memory system (see Memory System below)

---

## Architecture Rules

Strict Clean Architecture. Dependency direction is inward only:

```
handler → service → repository → model
```

- **Never** import `handler` or `repository` from `service`
- **Never** import `service` from `model`
- Repository interfaces are defined in `service/`, implemented in `repository/`
- All singletons (`GetConfig()`, `GetLogger()`, `GetDatabase()`) live in `internal/config/`

**Naming:**
- Directories & files: singular (`model/user.go`, `service/auth.go`)
- Structs: singular (`User`, `Session`)
- Go interfaces: defined in the layer that consumes them

---

## Key Commands

```bash
# Root
task setup        # Install all deps (api + dash)
task dev          # Start full stack (concurrent)
task test:all     # Run all tests
task lint         # Lint api + dash

# API
task test               # Unit tests only
task test:integration   # Integration tests (SQLite in-memory)
task ent:generate       # Regenerate EntGo code after schema change

# Dash
pnpm test         # Vitest unit tests
pnpm test:e2e     # Playwright E2E
pnpm build        # Production build
```

---

## Configuration

Config hierarchy (highest wins): `OPUS_* env vars` → `~/.opus/config.toml` → defaults

Key env vars:
```
OPUS_SERVER_PORT, OPUS_SERVER_ENV
OPUS_DATABASE_DRIVER (sqlite|postgres), OPUS_DATABASE_DSN
OPUS_AUTH_SECRET, OPUS_AUTH_GOOGLE_CLIENT_ID, OPUS_AUTH_GITHUB_CLIENT_ID
OPUS_AGENT_PROVIDER, OPUS_AGENT_MODEL, OPUS_AGENT_API_KEY
```

---

## Memory System

Agents working on this project **must** read and update the memory files.

| File | Purpose |
|------|---------|
| `.agents/MEMORY.md` | Long-term memory — user profile, project decisions, persistent context |
| `.agents/memory/[timestamp].md` | Short-term memory — per-session context, task progress, intermediate findings |

**Rules:**
- Read `.agents/MEMORY.md` at the start of every session
- Write a new `.agents/memory/[timestamp].md` at session start; update as work progresses
- Promote important findings/decisions to `.agents/MEMORY.md` at session end
- Short-term files use ISO 8601 timestamp format: `20260515T083000.md`

---

## AI Provider Integration

The agent system is provider-agnostic. All providers are configured globally via `config.toml` or env vars — no provider is hardcoded.

- Go implementation lives in `api/internal/agent/`
- Provider interface must be implemented for each provider; new providers are added by implementing the interface
- SSE streaming is the transport layer for all agent responses (`GET /stream`)

---

## Code Quality — Non-Negotiable

### Go (`api/`)
- Every exported function/type **must** have a GoDoc comment
- Every new feature **must** have a unit test (`_test.go`, co-located)
- Every repository method **must** have an integration test (`_integration_test.go`, build tag `integration`)
- Use `go.uber.org/mock` for mocking — never use concrete types in service tests

### TypeScript (`dash/`)
- Every exported function/component **must** have a JSDoc comment
- Every component **must** have a Vitest + React Testing Library test
- Critical flows **must** be covered by Playwright E2E tests

---

## Git & CI

Follow **Conventional Commits** — Release Please derives versions from these:

| Prefix | Effect |
|--------|--------|
| `feat:` | minor bump |
| `fix:` | patch bump |
| `feat!:` / `fix!:` | major bump |
| `chore:`, `ci:`, `refactor:` | patch, hidden in changelog |

CI runs on every push to `main` and every PR:
- `ci.yml` — lint + unit + integration tests (api & dash)
- `build.yml` — snapshot build + push `:edge` Docker images
- `release-please.yml` — versioning, GoReleaser, npm publish (on release only)

---

## Do's & Don'ts

**Do:**
- Run `task test:all` before considering any task complete
- Update `.agents/MEMORY.md` with decisions that affect future sessions
- Follow the existing file/struct naming conventions strictly
- Use `internal/config` singletons — never instantiate config/db/logger directly

**Don't:**
- Edit files under `ent/` manually — always use `task ent:generate`
- Add `handler` or framework imports into `service/` or `model/`
- Hardcode any provider, secret, or environment-specific value
- Skip tests or comments — both are required, not optional