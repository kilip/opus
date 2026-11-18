# AGENTS.md

Opus is the **Command Center for AI-Driven Software Factory**

## Tech Stack

- **Backend**: Go 1.27, Fiber v3, Entgo, SQLite (`modernc.org/sqlite`), Swag v2 (OpenAPI 3.1)
- **Frontend**: React 19, React Router 8 Framework Mode, TypeScript, Tailwind CSS v4
- **Client Data**: TanStack Query + `openapi-fetch` (types generated from `api/swagger.json`)
- **Tooling**: Bun, Vite 8, Biome, `golangci-lint`, `mise`

## Git & Commit Conventions

- **Default to non-release types** (`chore`, `ci`, `build`, `docs`, `refactor`, `test`, `style`). NEVER use `feat`, `fix`, `perf`, or `revert` unless the change modifies shipped runtime code (`cmd/`, `internal/`, `ent/`, `ui/app/`) and intentionally cuts a release (per `.github/release-please.json`).
- **Scope-to-type mapping**: `deploy` → `build`/`chore`, `release` → `ci`/`chore`, `tools` → `ci`/`chore`, `docs` → `docs`, `deps` → `chore(deps)`; only `api`/`ui` runtime changes use `feat`/`fix`/`perf`.
- **Atomic commits**: NEVER bundle unrelated files or scopes into one commit (`git add -A` across topics is forbidden). Stage and commit each `<type>(<scope>)` group separately.

## Repo Layout

- `cmd/opus/` — CLI entrypoint & Swag root (`@BasePath /api`)
- `ent/schema/` — Entgo unified schemas & generated client (`ent/`)
- `extensions/agy/` — Antigravity plugin (`mcp_config.json`, `/vibe` skill, subagents)
- `internal/app/` — HTTP server lifecycle, config, Fiber router wiring
- `internal/platform/` — `eventbus` (pub/sub), `httpx` (`{ data, meta, error }` envelope & errors), `mailer`
- `internal/<feature>/` — `agent`, `auth`, `node`, `notify`, `project`, `run`, `session`, `task`, `tracker`, `user`, `worker`
- `api/swagger.json` — generated OpenAPI 3.1 spec (DO NOT EDIT)
- `ui/embed.go` & `ui/app/` — `routes.ts`, `apps.ts`, `shared/` (`api/`, `auth/`, `util/`), `features/<feature>/`

## Feature Boundary Rules

### Go Feature Isolation

Import boundaries are enforced by `depguard` in `.golangci.yml`.

- **Imports & layers**: external/sibling packages import only `internal/<feature>` root; dependency direction is `handler → service → repo → domain`; `service/` uses `domain` types only (no `fiber`, `ent`, `httpx`, `handler`).
- **Standard layout**: `<feature>.go` facade (`New(...)`, `Register(r fiber.Router)`, wiring only), `domain/` (entities, enums, `errors.go`, `dto.go`), `repo/`, `service/`, `handler/`.
- **Thin handlers**: parse/validate request, call `service`, map result/error via `httpx`. Zero business logic in `handler/`.
- **Event bus**: publish facts through `eventbus`; never import sibling features to react. Event types live in the publisher's `domain/` and are re-exported from its root package.
- **MCP boundary**: `internal/agent` has no `repo/` and uses sibling root facades. MCP transport lives solely in `internal/agent/handler/mcp.go` (`POST /mcp`, mounted outside `/api` with Bearer node or session cookie auth, excluded from `api/swagger.json`).

### UI Feature Isolation

- `ui/app/features/<feature>/` MUST NOT import sibling `~/features/*`. Cross-feature composition happens only in `ui/app/routes.ts` and `ui/app/apps.ts`.
- Shared primitives, hooks, API helpers (`unwrap`, `unwrapMeta`, `ApiError`), error resolvers, and utilities belong in `ui/app/shared/` (`api/`, `auth/`, `util/`).
- **Feature layout**: `app.ts` (`AppDef`), `routes.ts`, `model.ts`, `components/`, `pages/`, and `api/` (`keys.ts`, `types.ts`, `errors.ts`, `queries.ts`, `actions.ts`).

## Backend: Fiber v3, Entgo, Swag & OpenAPI 3.1

- **Fiber v3**: Register feature routes under `/api` in `internal/app/fiber.go` via `<feature>.Register(api)`. Use `fiber.Ctx`.
- **Environment Variables**: Global/platform (including Sentry & UI build/dev config) use `APP_*`, auth uses `AUTH_*`, test-only helper flags use `TEST_*`, and features use `<FEATURE>_*` (`TASK_*`, `TRACKER_*`, `WORKER_*`, `NOTIFY_*`).
- **Entgo (`ent/`)**: Schemas live in `ent/schema/` with UUID v7 IDs via `newID` (`ent/schema/id.go`). Inject `*ent.Client` via `New` / `Register`. Re-run `go generate ./ent` after schema changes and commit generated code.
- **Strict DTO Boundary**: NEVER expose `ent` models in HTTP responses or Swag annotations. Define all DTOs in `internal/<feature>/domain/dto.go`; NEVER define DTOs in `handler/`.
- **OpenAPI 3.1 Pipeline**: Every handler MUST carry Swag v2 annotations (`@Summary`, `@Description`, `@ID`, `@Tags`, `@Produce`, `@Param`, `@Success`, `@Failure`, `@Router`). After changing handlers or DTOs, run `mise run api` — NEVER hand-edit `api/swagger.json` or `ui/app/shared/api/schema.d.ts`.
- **Auth**: Passwordless (email OTP + OAuth) with cookie sessions; providers in `internal/auth/provider/`, UI route guard `RequireAuth` in `ui/app/shared/auth/`.

## Frontend: React Router Framework Mode

- **Skill Reference**: Before changing routing, route modules, navigation, params, or rendering config, read `./ui/.agents/skills/react-router/SKILL.md` and `./ui/.agents/skills/react-router/references/framework-mode.md` (and version-matched docs in `ui/node_modules/react-router/docs/`).
- **Routing & Data Fetching**: Register feature `routes.ts` in `ui/app/routes.ts`. Use React Router for routing, `Outlet`, `meta`, and URL state (`useParams`, `useSearchParams`), and **TanStack Query + `openapi-fetch`** (`api`, `unwrap` from `~/shared/api/client`) inside `features/<feature>/api/` for all reads/writes.

## Lazy Dev & `ponytail:` Philosophy

Lazy means efficient, not careless. The best code is the code never written.

- **Decision Ladder** (stop at the first rung that holds):
  1. **YAGNI**: Does it need to exist?
  2. **Stdlib first**: Go/JS stdlib does it? Use it.
  3. **Native platform**: CSS over JS, DB/Ent constraint over app code, HTML attributes over custom state.
  4. **Existing dependency**: Already in `go.mod` or `package.json`? Use it; never add a package for a few lines.
  5. **One-liner**: Can it be one clear line? Write it.
  6. **Minimum working code**: Fewest files, shortest diff, deletion over addition, boring over clever.
- **No Speculative Abstraction**: No single-implementation interface (e.g. `eventbus.Bus` is a concrete struct), no single-product factory, no config knob for a constant, no `repository/` wrapper without domain complexity. Keep `service/` as plain structs/functions unless 2+ implementations exist.
- **`ponytail:` Comments**: Mark deliberate simplifications inline with ceiling and upgrade path:
  ```go
  // ponytail: in-memory slice filter; move to Ent predicate when tasks > 1k
  ```
- **Never Simplify Away**: Input validation at trust boundaries, data-loss error handling, security, accessibility, or explicitly requested behavior.
- **Testing**: One focused runnable check for non-trivial logic; skip tests for trivial one-liners.

## Verification Commands

Run the narrowest verification covering your change before finishing:

| Scope                             | Command                                                              |
| :-------------------------------- | :------------------------------------------------------------------- |
| Full QC (format + lint + test)    | `mise run qc`                                                        |
| Format Go & UI                    | `mise run format`                                                    |
| All tests (Go + UI)               | `mise run test`                                                      |
| Regenerate OpenAPI 3.1 + UI types | `mise run api`                                                       |
| Go tests (race detector)          | `go test -race ./...`                                                |
| Go vulnerability check            | `govulncheck ./...`                                                  |
| Full lint & typecheck (Go + UI)   | `mise run lint`                                                      |
| UI format / lint check            | `bunx --cwd ui biome check .`                                        |
| UI tests                          | `bun test --cwd ui --pass-with-no-tests`                             |
| Build embedded UI assets          | `mise run build-ui`                                                  |
| Helm chart lint & render          | `helm lint charts --strict && helm template opus charts > /dev/null` |
