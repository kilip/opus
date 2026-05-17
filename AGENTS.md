# Opus

Opus is an open-source, self-hosted, autonomous AI assistant designed to operate 24/7 on behalf of individuals and teams. It unifies a structured knowledge base, a programmable workflow engine, and a proactive AI agent layer into a single platform.

## Project Structure
- `server/`: Go-based backend application serving REST/SSE API.
- `dash/`: Progressive Web Application (PWA) frontend.
- `docs/`: Project documentation including Architecture Decision Records (ADRs).

## Key Technology Stack

### Backend (`server/`)
- **Language**: Go
- **Framework**: Fiber (HTTP delivery layer)
- **ORM**: Ent
- **Configuration**: Viper (layered configuration via files and env vars)

### Frontend (`dash/`)
- **Language**: TypeScript
- **Framework**: React 19.x with Vite 6.x
- **Routing**: TanStack Router
- **Data Fetching/State**: TanStack Query
- **Styling/UI**: Tailwind CSS 4.x, shadcn/ui
- **PWA**: vite-plugin-pwa (Workbox)

## Development Conventions

### Backend Architecture (ADR-001)
- Follows a feature-based Clean Architecture with explicit layer boundaries.
- **Directories**:
  - `internal/[feature]/`: Domain models, business logic (Services), and repository interfaces.
  - `adapter/entgo/`: Concrete database implementations (Repositories).
  - `delivery/http/`: HTTP handlers and routing.
  - `internal/config/`: Configuration loading.
  - `internal/shared/`: Cross-cutting domain entities.
- **Dependency Rule**: Dependencies flow inward. Delivery and Adapter layers depend on the Domain layer, never the reverse.

### Frontend Architecture (ADR-003)
- Feature-based directory structure mirroring the backend domains (e.g., `auth`, `agent`, `vault`, `workflow`).
- **Directories**:
  - `dash/src/app/`: Entry point, global providers, and router tree.
  - `dash/src/routes/`: Page-level components defined using TanStack Router file-based routing.
  - `dash/src/features/[feature]/`: Domain-scoped components, hooks, and API queries.
  - `dash/src/shared/`: Cross-feature UI components, hooks, and types.
- **Offline Strategy**:
  - GET requests use `NetworkFirst` with cache fallback.
  - Mutations (POST/PUT/DELETE) use a background sync queue when offline.

### Configuration (ADR-002)
- Managed by `server/internal/config`. Uses a layered approach:
  - **DEVELOPMENT**: `./.opus` (current-working-directory/.opus)
  - Default: `~/.opus/config.json`
  - Environment: `OPUS_*` variables
  - Override: `OPUS_HOME` environment variable for custom config location.

## Building and Running
- TBD

## Memory System

Agent memory lives in current working direcotry `.agents/`. **Every session gets its own file** — never append to an existing session file.

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