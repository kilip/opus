# AGENTS.md

Opus is a single-binary **Modular Monolith** web dashboard for AI coding agents (Go + Fiber v3 backend, Entgo ORM, OpenAPI 3.1 contract, React Router Framework Mode SPA embedded via `ui/embed.go`).

## 1. Git & Branching Gate (Run First)

- **NEVER commit or push to `main`.**
- Before creating or editing any file, check the active branch:
  ```bash
  git branch --show-current
  ```
- If on `main`, switch to an existing branch or create one before writing code:
  ```bash
  git switch -c <type>/<slug>   # e.g. feat/task-crud, fix/api-health, docs/agents-md
  ```
- Push feature branches only when explicitly instructed by the user.
- **Commit messages** are enforced by `commitlint.config.ts` and `.husky/commit-msg`:
  - Format: `<type>(<scope>): <subject>` (header and body lines ≤ 80 chars).
  - Required `<scope>`: `api`, `ui`, `deps`, `release`, `tools`, `deploy`, `docs`.
  - **Default to non-release types** (`chore`, `ci`, `build`, `docs`, `refactor`, `test`, `style`). NEVER use `feat`, `fix`, `perf`, or `revert` unless the change modifies shipped runtime code (`cmd/`, `internal/`, `ent/`, `ui/app/`) and MUST cut a new release version (per `.github/release-please.json`).
  - Scope-to-type rule of thumb: `deploy` → `build`/`chore`, `release` → `ci`/`chore`, `tools` → `ci`/`chore`, `docs` → `docs`, `deps` → `chore(deps)` (only `api`/`ui` runtime changes use `feat`/`fix`/`perf`).
  - **Multi-stage atomic commits**: NEVER bundle unrelated files or scopes into one commit (`git add -A` is forbidden across topics). Group and commit files in separate stages by `<type>(<scope>)`.

## 2. Modular Monolith Architecture ("Feature" Terminology)

Domain units across both backend and frontend are called **features** (never "modules").

```txt
├── cmd/opus/                     # CLI entrypoint & Swag root (@BasePath /api)
├── ent/                          # Entgo unified schema (ent/schema/) & generated graph client
├── internal/
│   ├── app/                      # HTTP server lifecycle, config, Fiber router wiring
│   ├── platform/
│   │   └── httpx/                # { data, meta, error } envelope, sentinel errors, ErrorHandler
│   └── <feature>/                # Go feature (e.g. internal/task, internal/user)
│       ├── <feature>.go          # Public feature facade: New(...), Register(r fiber.Router)
│       ├── domain/               # Leaf domain entities, enums, sentinel errors (errors.go)
│       ├── repo/                 # Storage implementation (in-memory or Ent client)
│       └── handler/              # Fiber handlers, DTOs (dto.go), Swag annotations, handler tests
├── api/
│   └── swagger.json              # Generated OpenAPI 3.1 spec (DO NOT EDIT)
└── ui/
    ├── embed.go                  # //go:embed all:build/client
    └── app/
        ├── routes.ts             # Root route aggregator
        ├── apps.ts               # App switcher registry
        ├── shared/               # Cross-feature UI primitives, theme, api/client.ts, util/
        │   ├── api/              # client.ts (api, unwrap, unwrapMeta, ApiError), errors.ts
        │   └── util/             # dates.ts, cross-feature pure utility functions
        └── features/<feature>/   # UI feature
            ├── app.ts            # App switcher entry (AppDef)
            ├── routes.ts         # React Router route entries
            ├── model.ts          # Feature domain constants, stages, filters, helpers
            ├── api/
            │   ├── keys.ts       # TanStack Query key factory
            │   ├── types.ts      # Schema type aliases from ~/shared/api/schema
            │   ├── errors.ts     # Feature error code -> message map
            │   ├── queries.ts    # queryOptions + useQuery hooks (unwrap)
            │   └── actions.ts    # useMutation hooks + cache invalidation
            ├── components/       # Feature-scoped components
            └── pages/            # Route page components
```

### Feature Boundary Rules

1. **Go Feature Isolation**:
   - External packages (`internal/app` or sibling features) MUST import only the feature root package `internal/<feature>`, NEVER its subpackages (`internal/<feature>/handler`, `internal/<feature>/repo`, etc.).
   - Backend feature layout follows `<feature>.go` facade (`New(...)` + `Register(r fiber.Router)`), `domain/` for domain models and sentinel errors, `repo/` for storage implementations, and `handler/` for HTTP endpoints and DTOs.
   - Unified HTTP response envelope `{ data, meta, error }` and status mapping are provided by `internal/platform/httpx`.
2. **UI Feature Isolation**:
   - Files in `ui/app/features/<feature>/` MUST NOT import from sibling `~/features/*`.
   - Shared components, hooks, api client helpers (`unwrap`, `unwrapMeta`), base error resolvers, and utilities belong in `ui/app/shared/` (`ui/app/shared/util/`).
   - Feature UI topology organizes data access under `api/` (`keys.ts`, `types.ts`, `errors.ts`, `queries.ts`, `actions.ts`), domain helpers/filters in `model.ts`, components in `components/`, and routes in `pages/`.
   - Cross-feature composition happens only in `ui/app/routes.ts` and `ui/app/apps.ts`.

## 3. Backend: Fiber v3, Entgo, Swag & OpenAPI 3.1

- **Fiber v3**: Register feature routes under the `/api` group in `internal/app/fiber.go` via `<feature>.Register(api)`. Use `fiber.Ctx` (v3 signature: `func(c fiber.Ctx) error`).
- **Entgo (`ent/`)**: All database schemas live in `ent/schema/`. Share the generated `*ent.Client` across features via dependency injection in `Register` / `New`.
- **Strict DTO Boundary**:
  - NEVER expose generated `ent` models directly in HTTP responses or Swag annotations.
  - Define dedicated request/response structs in `internal/<feature>/handler/dto.go`.
- **OpenAPI 3.1 Contract Pipeline**:
  - Every handler MUST carry Swag v2 godoc annotations (`@Summary`, `@Description`, `@ID`, `@Tags`, `@Produce`, `@Param`, `@Success`, `@Failure`, `@Router`).
  - After changing any handler annotation or DTO, regenerate the OpenAPI 3.1 spec and TypeScript schema:
    ```bash
    mise run api
    ```
  - NEVER hand-edit `api/swagger.json` or `ui/app/shared/api/schema.d.ts`.

## 4. Frontend: React Router Framework Mode (`ssr: false`)

- **Skill Reference**: Before modifying routing, route modules, navigation, params, or rendering config, read `./ui/.agents/skills/react-router/SKILL.md` and `./ui/.agents/skills/react-router/references/framework-mode.md` (plus version-matched docs in `ui/node_modules/react-router/docs/`).
- **SPA Framework Mode**:
  - `ui/react-router.config.ts` sets `ssr: false`.
  - NEVER define server-only route exports (`loader`, `action`, `headers`).
  - Each feature defines routes in `ui/app/features/<feature>/routes.ts` using `@react-router/dev/routes` and registers them in `ui/app/routes.ts`.
- **Data Fetching & Mutations**:
  - Use React Router Framework Mode for routing, layouts (`Outlet`), `meta`, and URL state (`useParams`, `useSearchParams`).
  - Use **TanStack Query + `openapi-fetch`** (`api` and `unwrap` from `~/shared/api/client`) for all API reads and writes inside `ui/app/features/<feature>/api/`:
    - `keys.ts` — query key factory
    - `queries.ts` — `queryOptions` + `useQuery` hooks
    - `actions.ts` — mutations + cache invalidation
    - `types.ts` — aliases from `~/shared/api/schema`

## 5. Lazy Dev & `ponytail:` Philosophy

Lazy means efficient, not careless. The best code is the code never written.

- **Decision Ladder** (stop at the first rung that holds):
  1. **YAGNI**: Does this need to exist at all?
  2. **Stdlib first**: Go/JS standard library does it? Use it.
  3. **Native platform**: CSS over JS, DB/Ent constraint over app code, HTML form/input attributes over custom state.
  4. **Existing dependency**: Already in `go.mod` or `package.json`? Use it; never add a new package for what a few lines can do.
  5. **One-liner**: Can it be one clear line? Write one line.
  6. **Minimum working code**: Fewest files, shortest working diff, deletion over addition, boring over clever.
- **No Speculative Abstraction**: No interface with one implementation, no factory for one product, no config knob for a constant, no `service/` or `repository/` wrapper without real domain complexity.
- **`ponytail:` Comments**: Mark deliberate simplifications inline with a `ponytail:` comment naming the ceiling and upgrade path:
  ```go
  // ponytail: in-memory slice filter; move to Ent predicate when tasks > 1k
  ```
- **Never Simplify Away**: Input validation at trust boundaries, error handling that prevents data loss, security, accessibility, or explicitly requested behavior.
- **Testing**: Leave one focused runnable check for non-trivial logic; skip tests for trivial one-liners.

## 6. Verification Commands

Run the narrowest verification covering your change before finishing:

| Scope | Command |
| :--- | :--- |
| Regenerate OpenAPI 3.1 + UI types | `mise run api` |
| Go tests | `go test ./...` |
| Full lint & typecheck (Go + UI) | `mise run lint` |
| UI format / lint check | `bunx biome check .` |
| Build embedded UI assets | `mise run build-ui` |
