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
delivery (handler) → service → repository → model
```

- **Never** import `delivery` or `repository` from `service`
- **Never** import `service` from `model`
- Repository interfaces are defined in `service/`, implemented in `repository/`
- All singletons (`GetConfig()`, `GetLogger()`, `GetDatabase()`) live in `internal/config/`

**Naming:**
- Directories & files: singular (`model`, `service`, `repository`)
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

Config hierarchy (highest wins): `OPUS_* env vars` → `config.toml` → defaults

Opus looks for `config.toml` in the following locations:
1. `OPUS_HOME` environment variable
2. Local `./.opus/` directory
3. Current directory `./`
4. Default `~/.opus/` directory

Key env vars:
```bash
# API
OPUS_SERVER_PORT, OPUS_SERVER_ENV
OPUS_DATABASE_DRIVER (sqlite|postgres), OPUS_DATABASE_DSN
OPUS_AUTH_SECRET, OPUS_AUTH_ACCESS_TOKEN_TTL, OPUS_AUTH_REFRESH_TOKEN_TTL
OPUS_AUTH_GOOGLE_CLIENT_ID, OPUS_AUTH_GOOGLE_CLIENT_SECRET, OPUS_AUTH_GOOGLE_REDIRECT_URL
OPUS_AUTH_GITHUB_CLIENT_ID, OPUS_AUTH_GITHUB_CLIENT_SECRET, OPUS_AUTH_GITHUB_REDIRECT_URL
OPUS_AGENT_PROVIDER, OPUS_AGENT_MODEL, OPUS_AGENT_API_KEY

# Dashboard
NEXT_PUBLIC_API_URL, NEXT_PUBLIC_DEV_MODE
```

---

## Memory System

Agents working on this project **must** read and update the memory files at the start and end of every session.

### File Structure

```
.agents/
├── MEMORY.md                          # Long-term memory — persistent decisions, user profile, project context
└── memory/
    ├── YYYY-MM-DD-<slug>.md           # Date-based session memory files
    ├── YYYY-MM-DD-consolidated.md     # Consolidated summary of multiple session files
    ├── 2026-05-15-auth-implementation.md
    ├── 2026-05-16-ent-schema-refactor.md
    └── ...
```

### Naming Convention

Memory files under `.agents/memory/` **must** follow this format:

```
YYYY-MM-DD-<slug>.md
```

- `YYYY-MM-DD` — ISO 8601 date of the session (e.g., `2026-05-16`)
- `<slug>` — short kebab-case descriptor of the session topic (e.g., `auth-refactor`, `ent-migration`, `ci-fix`)

**Examples:**
```
2026-05-16-oauth2-google-setup.md
2026-05-16-ent-schema-session.md
2026-05-17-goreleaser-multiarch-fix.md
```

If multiple sessions occur on the same date for different topics, create separate files per topic. Never overwrite a file — append or create a new one with a more specific slug.

### File Roles

| File | Purpose |
|------|---------|
| `.agents/MEMORY.md` | Long-term memory — user preferences, architectural decisions, persistent context that survives across sessions |
| `.agents/memory/YYYY-MM-DD-<slug>.md` | Short-term session memory — task progress, intermediate findings, decisions made during this session |
| `.agents/memory/YYYY-MM-DD-consolidated.md` | Auto-summary of multiple session files — produced by the consolidation process |

### Session Workflow

**At session start:**
1. Read `.agents/MEMORY.md` in full
2. Scan `.agents/memory/` for the most recent files (last 3–5) to understand recent context
3. **Check if consolidation is needed** — if there are 5 or more non-consolidated session files, run the consolidation process before proceeding (see Memory Consolidation below)
4. Create a new file `.agents/memory/YYYY-MM-DD-<slug>.md` for the current session
5. Write initial context: goal, relevant prior decisions, any open questions

**During session:**
- Update the current session file as work progresses
- Log decisions, discoveries, and blockers as they happen

**At session end:**
1. Finalize the session file with a summary of what was done and what remains
2. Promote any decisions or findings that affect future sessions to `.agents/MEMORY.md`
3. Do **not** delete old session files manually — only the consolidation process may delete them

### What Belongs Where

**`.agents/MEMORY.md` (long-term):**
- User identity and preferences
- Confirmed architectural decisions
- Agreed-upon conventions that deviate from defaults
- Known constraints or non-negotiables
- Status of major features (done / in progress / blocked)

**`.agents/memory/YYYY-MM-DD-<slug>.md` (short-term):**
- Task description and acceptance criteria for this session
- Steps taken and their outcomes
- Intermediate findings (e.g., "discovered EntGo does not support X")
- Open questions to resolve in a follow-up session
- What was left incomplete and why

**`.agents/memory/YYYY-MM-DD-consolidated.md` (consolidated):**
- Highlight-only summary extracted from multiple session files
- Sections: Decisions, Progress, Blockers, Key Findings
- Replaces the individual session files it summarises

---

## Memory Consolidation

Consolidation reduces clutter in `.agents/memory/` by merging older session files into a single summary per day.

### When to Consolidate

Consolidation **must** be triggered in either of these conditions:

- **Automatic** — at session start, if 5 or more non-consolidated session files exist in `.agents/memory/`
- **On request** — when the user explicitly asks for consolidation

A "non-consolidated" file is any file whose slug does not end in `consolidated`.

### Consolidation Process

**Step 1 — Identify files to consolidate**

Collect all non-consolidated session files in `.agents/memory/`, sorted by date (oldest first). If there are fewer than 5 and consolidation was not explicitly requested, skip.

**Step 2 — Determine output filename**

Use today's date and the fixed slug `consolidated`:

```
YYYY-MM-DD-consolidated.md
```

If a consolidated file for today already exists, **append** a new section to it rather than overwriting.

**Step 3 — Write the consolidated summary**

For each source file, extract only the highlights. The output format is:

```markdown
# Consolidated Memory — YYYY-MM-DD

> Auto-generated summary. Source files: [list of filenames]

---

## YYYY-MM-DD — <original-slug>

### Decisions
- <key decisions made>

### Progress
- <features completed or advanced>

### Blockers
- <unresolved blockers or open questions>

### Key Findings
- <discoveries relevant to future sessions>

---

## YYYY-MM-DD — <original-slug>

...
```

Omit any section that has no content. Do not copy full task logs, code snippets, or verbose notes — highlights only.

**Step 4 — Delete source files**

After the consolidated file is written, delete each source file that was included in the summary. Consolidated files (slug ending in `consolidated`) are never deleted.

**Step 5 — Verify**

Confirm `.agents/memory/` now contains only the consolidated file (and any files newer than those consolidated).

### Consolidation Rules

- **Never consolidate the current session's file** — only files from prior sessions
- **Never delete a consolidated file** — they are permanent summaries
- **Append, never overwrite** — if `YYYY-MM-DD-consolidated.md` already exists, add a new dated section
- **Highlights only** — keep each session summary to the essential decisions, progress, blockers, and findings
- **Promote critical decisions** — if consolidation surfaces a decision not yet in `.agents/MEMORY.md`, add it there before finishing

---

## AI Provider Integration

The agent system is provider-agnostic. All providers are configured globally via `config.toml` or env vars — no provider is hardcoded.

- **Status:** Implementation in progress.
- **Location:** Go implementation lives in `api/internal/agent/` (intended).
- Provider interface must be implemented for each provider; new providers are added by implementing the interface.
- SSE streaming is the transport layer for all agent responses (`GET /stream`).

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

**ONLY USE THIS SCOPE**: `api`, `dash`, `installer`, `docs`, `core`, `deps`

CI runs on every push to `main` and every PR:
- `ci.yml` — lint + unit + integration tests (api & dash)
- `build.yml` — snapshot build + push `:edge` Docker images
- `release-please.yml` — versioning, GoReleaser, npm publish (on release only)

**ALWAYS**: `task test:all` before commit

---

## Do's & Don'ts

**Do:**
- Run `task test:all` before considering any task complete
- Update `.agents/MEMORY.md` with decisions that affect future sessions
- Follow the existing file/struct naming conventions strictly
- Use `internal/config` singletons — never instantiate config/db/logger directly
- Name session memory files as `YYYY-MM-DD-<slug>.md` — always
- Run consolidation at session start if 5+ non-consolidated files exist

**Don't:**
- Edit files under `ent/` manually — always use `task ent:generate`
- Add `handler` or framework imports into `service/` or `model/`
- Hardcode any provider, secret, or environment-specific value
- Skip tests or comments — both are required, not optional
- Delete or overwrite existing files under `.agents/memory/` manually
- Include verbose logs or code snippets in consolidated memory files