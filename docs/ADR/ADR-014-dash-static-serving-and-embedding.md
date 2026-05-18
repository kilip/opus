# ADR-014: Dash Static Serving and Binary Embedding

**Status:** Accepted
**Date:** 2026-05-19
**Deciders:** Chief Architect, Product Manager
**Context:** Opus Server (`opus/server/`) · Opus Dash (`opus/dash/`) · Installer (`get-opus/`)

---

## 1. Context

Opus Dash (`dash/`) is a Vite + React PWA defined in ADR-003. As of this ADR, Dash has never
been served in any deployment context. The current state is:

- `opus start` starts only the Fiber API server on `:8080`
- `dash/dist/` exists as a build artefact but is not served anywhere
- GoReleaser produces a Go binary with no Dash assets included
- `npx get-opus` opens `http://localhost:8080` but no UI is present there

For Opus to function as a self-hosted personal assistant, Dash must be:

1. **Embedded** into the Go binary so that a single downloaded binary is self-contained — no
   separate file distribution, no path configuration, no web server dependency.
2. **Served automatically** when `opus start` is run — no operator step required.
3. **Isolated** from the API server on a dedicated port to preserve clean separation between
   the REST API and the UI static server.
4. **Transparent to the development workflow** — frontend contributors must retain full Vite
   HMR and fast refresh without interference from the Go embedding mechanism.

This ADR establishes the canonical approach for embedding `dash/dist/` into the Go binary,
serving it on a dedicated port, wiring the build pipeline in GoReleaser, and preserving the
development workflow.

---

## 2. Decision

Opus Server embeds `dash/dist/` into the Go binary at compile time using `//go:embed`. A
dedicated Fiber static file server serves the embedded assets on a second port (`:8081`),
started as a separate goroutine from `opus start`. The API server on `:8080` is unchanged.

Development mode retains the existing Vite dev server workflow — no Go embedding is involved
during development.

---

### 2.1 Core Principles

| Principle | Description |
|---|---|
| **Single binary** | One `opus` binary contains both the API server and all Dash assets — no sidecar files required |
| **Port isolation** | Dash static server runs on `:8081`; API server remains on `:8080`; no routes bleed between ports |
| **ADR-004 intact** | No API URL structure changes; `/agents`, `/auth`, etc. remain on `:8080` as defined |
| **Dev/prod parity** | Dash behaviour is identical in production; only the *delivery mechanism* differs in development |
| **Explicit over implicit** | Embed is unconditional at compile time; no runtime feature flags or `os.Getenv` switches govern asset serving |
| **Zero new operator steps** | A user who runs `opus start` gets both servers; no manual asset copy or path export required |

---

### 2.2 Embedding Strategy

Dash assets are embedded using the Go standard library `embed` package. The embed directive
is placed in a dedicated file `internal/dash/embed.go` to isolate the coupling between the
Go binary and the build artefact path.

```go
// internal/dash/embed.go
package dash

import "embed"

// FS holds the compiled Dash PWA assets embedded at build time.
// The embed path resolves relative to the module root (server/).
// GoReleaser builds dash/dist/ before invoking `go build`, ensuring
// the directory exists when the embed directive is evaluated.
//
//go:embed all:dist
var FS embed.FS
```

> **Note for implementors and AI agents:** The `dist/` directory referenced by `//go:embed`
> is located at `server/internal/dash/dist/` in the repository. GoReleaser copies the output
> of the Dash build (`dash/dist/`) into this path before invoking `go build`. The source
> `dash/` directory and `server/internal/dash/dist/` are separate locations; only the latter
> is committed (as an empty `.gitkeep`) and populated at build time.

**`.gitignore` convention:**

```gitignore
# server/.gitignore
internal/dash/dist/*
!internal/dash/dist/.gitkeep
```

---

### 2.3 Dash Static Server

A dedicated Fiber application serves the embedded assets. It is defined in
`internal/dash/server.go` and is entirely separate from the API Fiber app defined in
`internal/delivery/gofiber/`.

```go
// internal/dash/server.go
package dash

import (
    "net/http"

    "github.com/gofiber/fiber/v3"
    "github.com/gofiber/fiber/v3/middleware/filesystem"
)

// NewServer returns a Fiber app configured to serve the embedded Dash PWA assets.
// All unmatched routes fall back to index.html to support client-side routing
// (TanStack Router history mode).
func NewServer() *fiber.App {
    app := fiber.New(fiber.Config{
        DisableStartupMessage: true,
    })

    app.Use(filesystem.New(filesystem.Config{
        Root:         http.FS(FS),
        Index:        "index.html",
        NotFoundFile: "index.html", // SPA fallback for client-side routing
        Browse:       false,
    }))

    return app
}
```

**SPA fallback rationale:** TanStack Router uses history-mode routing (e.g. `/agent/agt_001`).
Without a catch-all fallback to `index.html`, a hard refresh on any non-root route would
return a 404 from the static server. The `NotFoundFile: "index.html"` setting ensures all
unmatched paths return the SPA shell, allowing TanStack Router to handle routing client-side.

---

### 2.4 Configuration — Hybrid Composition (ADR-002)

A `dash.Config` struct is defined in `internal/dash/config.go` following the Hybrid Config
Composition Pattern from ADR-002. It is composed into the root `Config` in
`internal/config/model.go`.

```go
// internal/dash/config.go
package dash

// Config holds configuration for the Dash static server.
// Owned by the dash package; composed into the root config.Config
// by internal/config/model.go.
//
// Environment variable override:
//   OPUS_DASH_ADDRESS — sets the TCP address for the Dash static server
type Config struct {
    // Address is the TCP address the Dash static server listens on.
    // Default: ":8081".
    Address string `mapstructure:"address" json:"address" jsonschema:"default=:8081,description=TCP address the Dash static file server listens on"`
}
```

Root config composition in `internal/config/model.go`:

```go
// internal/config/model.go (excerpt)
import "github.com/kilip/opus/server/internal/dash"

type Config struct {
    Server   gofiber.Config   `mapstructure:"server"   json:"server"   jsonschema:"required"`
    Database DatabaseConfig   `mapstructure:"database" json:"database" jsonschema:"required"`
    Log      LogConfig        `mapstructure:"log"      json:"log"`
    LLM      llm.Config       `mapstructure:"llm"      json:"llm"      jsonschema:"required"`
    Agent    agent.Config     `mapstructure:"agent"    json:"agent"`
    Vault    vault.Config     `mapstructure:"vault"    json:"vault"`
    Workflow workflow.Config  `mapstructure:"workflow" json:"workflow"`
    Queue    queue.Config     `mapstructure:"queue"    json:"queue"`
    Dash     dash.Config      `mapstructure:"dash"     json:"dash"`
}
```

**Example `config.json` addition:**

```json
{
  "dash": {
    "address": ":8081"
  }
}
```

---

### 2.5 Bootstrap Integration (ADR-012)

The Dash static server is bootstrapped as part of `container.Bootstrap()`, consistent with
the module system defined in ADR-012. It is initialised after all domain services but before
the Fiber delivery layer bootstrap.

```go
// internal/dash/bootstrap.go
package dash

import "github.com/gofiber/fiber/v3"

var dashApp *fiber.App

// Bootstrap initialises the Dash static server.
// Called by container.Bootstrap() during startup.
func Bootstrap(cfg Config) {
    dashApp = NewServer()
    setApp(dashApp)
}

func setApp(a *fiber.App) { dashApp = a }

// GetServer returns the initialised Dash Fiber app.
// Panics if Bootstrap has not been called.
func GetServer() *fiber.App {
    if dashApp == nil {
        panic("dash: Bootstrap has not been called")
    }
    return dashApp
}
```

`container/bootstrap.go` addition:

```go
// internal/container/bootstrap.go (excerpt)
import "github.com/kilip/opus/server/internal/dash"

func Bootstrap(cfg *config.Config) {
    initShared(cfg)
    auth.Bootstrap(...)
    vault.Bootstrap(...)
    agent.Bootstrap(...)
    workflow.Bootstrap(...)
    // ... other domains ...
    dash.Bootstrap(cfg.Dash)                    // ← added
    fiberdelivery.Bootstrap(c.fiber, c.log, cfg.Server)
}
```

`container/container.go` addition:

```go
// GetDash returns the initialised Dash static server Fiber app.
// Panics if Bootstrap has not been called.
func GetDash() *fiber.App {
    mustInit()
    return c.dash
}
```

---

### 2.6 `opus start` — Two Goroutines

Both servers are started from `server/cmd/opus/start.go`. Each runs in its own goroutine.
The process blocks until either server returns an error.

```go
// server/cmd/opus/start.go (updated runStart excerpt)
func runStart(cmd *cobra.Command, args []string) error {
    // ... PID write, config load, container bootstrap, queue start (unchanged) ...

    apiAddress := cfg.Server.Address
    if apiAddress == "" {
        apiAddress = ":8080"
    }

    dashAddress := cfg.Dash.Address
    if dashAddress == "" {
        dashAddress = ":8081"
    }

    log.Info("starting opus api server",  logger.String("address", apiAddress))
    log.Info("starting opus dash server", logger.String("address", dashAddress))

    errCh := make(chan error, 2)

    go func() {
        errCh <- container.GetFiber().Listen(apiAddress)
    }()

    go func() {
        errCh <- container.GetDash().Listen(dashAddress)
    }()

    // Block until the first server exits (error or shutdown signal).
    return <-errCh
}
```

**Shutdown behaviour:** If either server exits (e.g. port conflict, graceful shutdown signal),
the `errCh` receive unblocks and `runStart` returns. The process then exits via `main.go`,
which triggers deferred PID file cleanup. Graceful shutdown of the second server is handled
by the OS process termination; a future ADR may introduce explicit `context.Context`
cancellation propagation across both servers.

---

### 2.7 Vite Configuration

`dash/vite.config.ts` requires no changes to the `base` URL — it is already `/` by default.
The API base URL used by `shared/lib/api-client.ts` must point to the API server on `:8080`,
not the Dash server on `:8081`.

```typescript
// dash/vite.config.ts (relevant excerpt — no base change required)
export default defineConfig({
  base: '/',          // already the default; explicit for clarity
  // ...
});
```

```typescript
// dash/src/shared/lib/api-client.ts
// VITE_API_URL defaults to the API server, not the Dash server.
const BASE_URL = import.meta.env.VITE_API_URL ?? 'http://localhost:8080';
```

**CORS note:** Requests from `http://localhost:8081` (production Dash) to `http://localhost:8080`
(API) are same-host but different-port — the browser treats these as cross-origin. The Fiber
API server must include a CORS middleware allowing `http://localhost:8081` as an allowed origin.
This is an implementation concern and does not require a separate ADR amendment.

---

### 2.8 Development Workflow

Development mode is unchanged from the pre-ADR state. There is no Go embedding in development.

```
Development                    Production
──────────────────────         ──────────────────────────────
task dash:dev  → :5173         opus binary (single file)
task server:dev → :8080            ├── API server → :8080
                                   └── Dash static → :8081
```

**Taskfile convention:**

```yaml
# server/Taskfile.yml (additions)
tasks:
  dev:
    desc: Start Go API server in development mode
    cmds:
      - go run ./cmd/opus start

# dash/Taskfile.yml (existing — no change required)
tasks:
  dev:
    desc: Start Vite development server
    cmds:
      - pnpm dev

# Root Taskfile.yml (addition)
tasks:
  dev:
    desc: Start both API server and Dash dev server in parallel
    deps:
      - server:dev
      - dash:dev
```

**Why no proxy in development:** During development, `VITE_API_URL` defaults to
`http://localhost:8080` in `api-client.ts`. The frontend developer accesses Dash at
`:5173` and all API calls go directly to `:8080`. No Vite proxy configuration is needed.
This avoids a hidden indirection layer that could mask CORS or network errors in development.

---

### 2.9 GoReleaser Build Pipeline

GoReleaser must build Dash assets before invoking `go build`, so that `server/internal/dash/dist/`
is populated when the `//go:embed` directive is evaluated.

```yaml
# .goreleaser.yaml (updated before hooks excerpt)
before:
  hooks:
    - go mod tidy -C server
    # Mock generation (existing)
    - go run -C server go.uber.org/mock/mockgen -destination=mocks/auth.go    -package=mocks github.com/kilip/opus/server/internal/auth Repository,PolicyService,OAuthProvider
    - go run -C server go.uber.org/mock/mockgen -destination=mocks/logger.go  -package=mocks github.com/kilip/opus/server/internal/shared/logger Logger
    - go run -C server go.uber.org/mock/mockgen -destination=mocks/queue.go   -package=mocks github.com/kilip/opus/server/internal/shared/queue Queue
    - go run -C server go.uber.org/mock/mockgen -destination=mocks/eventbus.go -package=mocks github.com/kilip/opus/server/internal/shared/queue EventBus
    - go generate -C server ./...
    # Dash build — must run before go build
    - pnpm --dir dash install --frozen-lockfile
    - pnpm --dir dash build
    - cp -r dash/dist server/internal/dash/dist
```

**Build order guarantee:** GoReleaser executes `before.hooks` sequentially before any
`builds` entry. The `cp` step populates `server/internal/dash/dist/` immediately before
`go build` runs, ensuring the embed directive resolves successfully.

**CI pipeline (`ci.yml`) addition:** The `server-test` job does not require a Dash build
(unit and integration tests do not exercise the embedded assets). The `server-lint` job
must create an empty `server/internal/dash/dist/.gitkeep` before running `golangci-lint`
to prevent the `//go:embed` directive from failing on a missing directory.

```yaml
# .github/workflows/ci.yml (server-lint job addition)
- name: Prepare embed placeholder
  run: mkdir -p server/internal/dash/dist && touch server/internal/dash/dist/.gitkeep
```

---

### 2.10 `npx get-opus` Update (ADR-013 Amendment)

The `get-opus` installer opens `http://localhost:{dashPort}` (default `http://localhost:8081`)
instead of `http://localhost:{apiPort}`. The health check used to detect server readiness
remains against the API server (`GET /health` on `:8080`), since the API server initialises
before the Dash server and is the authoritative readiness signal.

```javascript
// get-opus/src/health.js (updated)
const API_PORT  = options.port     ?? 8080;
const DASH_PORT = options.dashPort ?? 8081;

// Health check against API server
await waitForHealth(`http://localhost:${API_PORT}/health`);

// Open Dash in browser
await open(`http://localhost:${DASH_PORT}`);
```

The generated `config.json` produced by the installer is updated to include the `dash` section:

```json
{
  "$schema": "...",
  "server":   { "address": ":{port}" },
  "dash":     { "address": ":{dashPort}" },
  "database": { "driver": "sqlite3", "dsn": "{dataDir}/opus.db" },
  "log":      { "level": "info", "format": "json" },
  "agent":    { "tick_interval": "60s", "max_retries": 3 },
  "queue":    { "driver": "database", "concurrency": 10 }
}
```

---

### 2.11 Directory Structure

This ADR introduces one new package and one new build-time directory.

```
opus/
├── server/
│   └── internal/
│       └── dash/
│           ├── embed.go        # //go:embed all:dist
│           ├── server.go       # NewServer() — Fiber static file server
│           ├── config.go       # dash.Config — owned by dash package
│           ├── bootstrap.go    # Bootstrap() + GetServer()
│           └── dist/           # Populated at build time by GoReleaser; gitignored except .gitkeep
│               └── .gitkeep
│
├── dash/
│   └── (unchanged — ADR-003)
│
└── get-opus/
    └── src/
        └── health.js           # Updated: open :8081 after health check on :8080
```

---

## 3. Alternatives Considered

### 3.1 Serve Dash on the Same Port as the API (`:8080`)

Mount Dash assets under a path prefix (e.g. `/app`) on the same Fiber app as the API.
Rejected because:

- Risks route collisions between API routes and static file paths as both surfaces grow.
- Makes it harder to apply different middleware (e.g. CORS, rate limiting, caching headers)
  to API vs static content independently.
- Violates the principle of explicit separation between the REST API and the UI delivery
  mechanism.

### 3.2 Serve Dash via a Separate Process / Sidecar

Run a standalone static file server (e.g. `caddy`, `nginx`, or a second Go binary) alongside
the main `opus` binary. Rejected because:

- Contradicts the self-hosted, single-binary distribution model that is a core Opus value
  proposition.
- Adds an operational dependency that non-technical users cannot be expected to manage.
- Breaks the `npx get-opus` one-command install promise.

### 3.3 Serve Dash Directly from `dash/dist/` on Disk

Reference `dash/dist/` as a filesystem path at runtime rather than embedding. Rejected because:

- Requires `dash/dist/` to exist on disk relative to the binary at runtime — fragile for
  packaged distributions.
- Breaks single-binary distribution; the installer would need to manage two artefacts.
- `//go:embed` adds zero runtime overhead for a local-first application.

### 3.4 Build Tag to Conditionally Exclude Embed in Development

Use a `//go:build !dev` build tag to exclude the embed directive in development mode,
allowing `go run` without requiring `dash/dist/` to exist. Rejected because:

- Introduces conditional compilation paths that must be tested separately, increasing CI
  surface area.
- Developers running `go run ./cmd/opus` in development do not need the Dash server at all —
  they use Vite at `:5173`. The `.gitkeep` placeholder is sufficient to satisfy the embed
  directive during development without a build tag.
- Explicit over implicit: a failing embed at `go build` time is a clear, immediate signal
  that the Dash build step was skipped, rather than a subtle runtime mismatch.

### 3.5 Proxy Vite Dev Server from Go in Development

Configure the Dash Fiber server to proxy requests to the Vite dev server when an env var
(e.g. `OPUS_DEV_DASH_URL`) is set, unifying the `:8081` endpoint across dev and production.
Deferred (not rejected) because:

- Adds non-trivial reverse proxy logic to the Go server with limited MVP benefit.
- Vite's HMR websocket proxying is complex and error-prone.
- The two-URL model (`:5173` dev, `:8081` production) is well-understood and imposes no
  friction on the development workflow.
- May be revisited in a future ADR if contributors report confusion from the port difference.

---

## 4. Consequences

### 4.1 Positive

- **Single-binary distribution** — `opus` contains everything; no file layout assumptions
  at runtime. The installer downloads one file and the product is complete.
- **Zero operator configuration** — `opus start` brings up both servers; `opus.pid` tracks
  the single process. No service file changes required.
- **Port isolation** — API and Dash are independently addressable; middleware, logging, and
  rate limiting can be applied differently to each.
- **ADR-004 fully intact** — No API URL changes; all existing and future API documentation
  remains accurate.
- **Dev workflow unchanged** — Frontend contributors run `pnpm dev` exactly as before;
  no new tooling or environment variables required.
- **Deterministic build** — GoReleaser's sequential `before.hooks` guarantee that
  `dash/dist/` is always populated before `go build`; no race condition in the build pipeline.

### 4.2 Negative / Trade-offs

- **Binary size increase** — Embedding `dash/dist/` adds the Dash bundle size to the Go
  binary (typically 1–3 MB gzipped). Acceptable for a self-hosted tool; documented in
  release notes.
- **CORS configuration required** — Requests from `:8081` to `:8080` are cross-origin;
  the API server must explicitly allow this origin. This is a one-time configuration addition
  to the Fiber CORS middleware, not an architectural concern.
- **`.gitkeep` discipline** — `server/internal/dash/dist/.gitkeep` must not be deleted;
  its absence causes `//go:embed` to fail at compile time. A CI lint step validates its
  presence.
- **GoReleaser `pnpm` dependency** — The release pipeline requires `pnpm` to be available
  in the GoReleaser CI environment. The `goreleaser` job in `.github/workflows/release.yml`
  must install `pnpm` before invoking GoReleaser.
- **Two ports to document** — User-facing documentation and the installer completion message
  must clearly distinguish `:8080` (API) from `:8081` (Dash). The `get-opus` completion
  output is the primary surface for communicating this to end users.

---

## 5. References

- [ADR-001: Server Clean Architecture](./ADR-001-server-clean-architecture.md)
- [ADR-002: Configuration Management](./ADR-002-server-configuration.md)
- [ADR-003: Opus Dash Frontend Architecture](./ADR-003-dash-frontend-architecture.md)
- [ADR-004: API Response Contract](./ADR-004-api-response-contract.md)
- [ADR-005: Server Delivery Layer with GoFiber v3](./ADR-005-server-delivery-layer-with-gofiber-v3.md)
- [ADR-012: Module System and Dependency Injection](./ADR-012-module-system-and-dependency-injection.md)
- [ADR-013: `npx get-opus` Installer](./ADR-013-get-opus-installer.md)
- [Go embed package — pkg.go.dev/embed](https://pkg.go.dev/embed)
- [GoFiber filesystem middleware — docs.gofiber.io](https://docs.gofiber.io/api/middleware/filesystem)
- [GoReleaser before hooks — goreleaser.com](https://goreleaser.com/customization/hooks/)
- [Vite base option — vitejs.dev](https://vitejs.dev/config/shared-options.html#base)
