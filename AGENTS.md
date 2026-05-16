# AGENTS.md

> Guidance for AI agents working on the Opus codebase.

---

## Project Overview

Opus is a self-hosted, single-user AI agent platform. Monorepo structure:

- `api/` — Go backend (GoFiber v3, EntGo, Viper, Cobra)
- `dash/` — Next.js 16 frontend (PWA, TanStack Query, Serwist, Shadcn/ui)
- `installer/` — npx installer (Node.js)
- `.agents/` — Agent memory system

---

## Architecture Rules

Strict Clean Architecture. Dependency direction is **inward only**:

```
delivery (handler) → service → repository → model
```

- **Never** import `delivery` or `repository` from `service`
- **Never** import `service` from `model`
- Repository interfaces are defined in `service/`, implemented in `repository/`
- All singletons (`GetConfig()`, `GetLogger()`, `GetDatabase()`) live in `internal/config/`
- `handler` must **never** import `repository` directly — always go through `service`

**Naming:** directories & files singular (`model`, `service`, `repository`); structs singular (`User`, `Session`).

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

Config hierarchy (highest wins): `OPUS_* env vars` → `config.toml` → defaults

Config locations (in order): `$OPUS_HOME` → `./.opus/` → `./` → `~/.opus/`

Key env vars:
```bash
OPUS_SERVER_PORT, OPUS_SERVER_ENV
OPUS_DATABASE_DRIVER (sqlite|postgres), OPUS_DATABASE_DSN
OPUS_AUTH_SECRET, OPUS_AUTH_ACCESS_TOKEN_TTL, OPUS_AUTH_REFRESH_TOKEN_TTL
OPUS_AUTH_GOOGLE_CLIENT_ID, OPUS_AUTH_GOOGLE_CLIENT_SECRET, OPUS_AUTH_GOOGLE_REDIRECT_URL
OPUS_AUTH_GITHUB_CLIENT_ID, OPUS_AUTH_GITHUB_CLIENT_SECRET, OPUS_AUTH_GITHUB_REDIRECT_URL
OPUS_AGENT_PROVIDER, OPUS_AGENT_MODEL, OPUS_AGENT_API_KEY
NEXT_PUBLIC_API_URL, NEXT_PUBLIC_DEV_MODE
```

---

## Memory System

Agent memory lives in `.agents/`. **Every session gets its own file** — never append to an existing session file.

### File Structure

```
.agents/
├── MEMORY.md                        # Long-term: persistent decisions, project context
└── memory/
    ├── YYYY-MM-DD-<slug>.md         # One file per session
    └── YYYY-MM-DD-consolidated.md  # Consolidated summary (auto-generated)
```

**Naming:** `YYYY-MM-DD-<slug>.md` — e.g. `2026-05-16-whatsapp-service.md`

### What Belongs Where

| File | Content |
|------|---------|
| `MEMORY.md` | Architectural decisions, user preferences, feature status, non-negotiables |
| `YYYY-MM-DD-<slug>.md` | Task goal, steps taken, decisions made, blockers, what remains |
| `YYYY-MM-DD-consolidated.md` | Highlights only: Decisions, Progress, Blockers, Key Findings |

### Session Workflow

**Start of session:**
1. Read `.agents/MEMORY.md` in full
2. Read last 3–5 session files in `.agents/memory/`
3. If 5+ non-consolidated session files exist → run consolidation first
4. Create a **new** file `.agents/memory/YYYY-MM-DD-<slug>.md` for this session
5. Write initial context: goal, relevant prior decisions, open questions

**During session:** Update the current session file as work progresses — log decisions, findings, blockers.

**End of session:**
1. Finalize session file with summary of what was done and what remains
2. Promote any lasting decisions to `.agents/MEMORY.md`

### Consolidation (when 5+ non-consolidated files exist)

1. Collect all non-consolidated files (slug not ending in `consolidated`), oldest first
2. Output to `YYYY-MM-DD-consolidated.md` — append if file already exists
3. Content: highlights only (Decisions, Progress, Blockers, Key Findings) — no code snippets or verbose logs
4. Delete source files after consolidated file is written
5. Never consolidate the **current session's file**; never delete consolidated files

---

## Code Quality — Non-Negotiable

### Go (`api/`)
- Every exported function/type **must** have a GoDoc comment
- Every new feature **must** have a unit test (co-located `_test.go`)
- Every repository method **must** have an integration test (`_integration_test.go`, build tag `integration`)
- Use `go.uber.org/mock` for mocking — never use concrete types in service tests
- Run `task test:all` before every commit

### TypeScript (`dash/`)
- Every exported function/component **must** have a JSDoc comment
- Every component **must** have a Vitest + React Testing Library test
- Critical flows **must** be covered by Playwright E2E tests

---

## Git & CI

Follow **Conventional Commits**. Allowed scopes: `api`, `dash`, `installer`, `docs`, `core`, `deps`

| Prefix | Version Bump |
|--------|-------------|
| `feat:` | minor |
| `fix:` | patch |
| `feat!:` / `fix!:` | major |
| `chore:`, `ci:`, `refactor:` | patch |

CI runs on every push to `main` and every PR:
- `ci.yml` — lint + unit + integration tests
- `build.yml` — snapshot build + `:edge` Docker images
- `release-please.yml` — versioning, GoReleaser, npm publish (on release only)

---

## AI Provider Integration

- **Status:** Implementation in progress
- **Location:** `api/internal/agent/` (intended)
- Provider interface must be implemented per provider; no provider is hardcoded
- SSE streaming is the transport layer for all agent responses (`GET /stream`)