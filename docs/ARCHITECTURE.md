# Architecture Document

**Product:** Opus  
**Version:** 1.0.0  
**Status:** Draft  
**Last Updated:** 2026-05-15  
**Authors:** Product & Architecture Team

---

## Table of Contents

1. [System Overview](#1-system-overview)
2. [High-Level Architecture](#2-high-level-architecture)
3. [API — Go Backend](#3-api--go-backend)
   - 3.1 [Technology Stack](#31-technology-stack)
   - 3.2 [Project Structure](#32-project-structure)
   - 3.3 [Clean Architecture Layers](#33-clean-architecture-layers)
   - 3.4 [Configuration System](#34-configuration-system)
   - 3.5 [Authentication & Session Management](#35-authentication--session-management)
   - 3.6 [Database Layer](#36-database-layer)
   - 3.7 [API Contracts](#37-api-contracts)
   - 3.8 [Testing Strategy](#38-testing-strategy)
   - 3.9 [Task Automation](#39-task-automation)
4. [Frontend — Next.js](#4-frontend--nextjs)
   - 4.1 [Technology Stack](#41-technology-stack)
   - 4.2 [Project Structure](#42-project-structure)
   - 4.3 [PWA Configuration](#43-pwa-configuration)
   - 4.4 [State Management & Data Fetching](#44-state-management--data-fetching)
   - 4.5 [Testing Strategy](#45-testing-strategy)
5. [Deployment & Distribution](#5-deployment--distribution)
   - 5.1 [Docker](#51-docker)
   - 5.2 [Bare Metal](#52-bare-metal)
   - 5.3 [npx Installer](#53-npx-installer)
   - 5.4 [CLI Commands](#54-cli-commands)
   - 5.5 [Auto-Restart](#55-auto-restart)
6. [Security](#6-security)
7. [Observability](#7-observability)
8. [Reserved: pkg/ Module](#8-reserved-pkg-module)

---

## 1. System Overview

Opus is a self-hosted, single-user AI agent platform. The system consists of two primary components:

- **opus-api** — A Go backend exposing a REST + SSE API.
- **opus-web** — A Next.js 16 Progressive Web App consuming the API.

Both components are distributed as a single installable unit via `npx opus install`, Docker, or pre-built binaries.

```mermaid
graph TD
    subgraph "User Device"
        WEB["opus-web (PWA)<br/>Next.js 16 + Serwist + Shadcn"]
    end

    subgraph "Host Machine"
        API["opus-api (Go)<br/>GoFiber v3 + EntGo + Viper"]
        DB[("Database<br/>SQLite or PostgreSQL")]
    end

    WEB -- "HTTP / SSE" --> API
    API --> DB
```

---

## 2. High-Level Architecture

### Request Flow

```mermaid
graph TD
    BROWSER["Browser (opus-web)"]
    
    subgraph "REST Path"
        ROUTER["GoFiber Router"]
        MIDDLEWARE["Middleware<br/>(Auth, Logger, Recovery)"]
        HANDLER["Handler"]
        SERVICE["Service (Usecase)"]
        REPO["Repository"]
        ENT["EntGo Client"]
        DB[("SQLite / PostgreSQL")]
    end

    subgraph "SSE Path"
        SSE_HANDLER["GoFiber SSE Handler"]
        SSE_SERVICE["Service (Usecase)"]
        AI["AI Provider (future)"]
    end

    BROWSER -- "REST Request" --> ROUTER
    ROUTER --> MIDDLEWARE
    MIDDLEWARE --> HANDLER
    HANDLER --> SERVICE
    SERVICE --> REPO
    REPO --> ENT
    ENT --> DB

    BROWSER -- "SSE Stream" --> SSE_HANDLER
    SSE_HANDLER --> SSE_SERVICE
    SSE_SERVICE --> AI
```

### Component Responsibilities

| Component | Responsibility |
|-----------|---------------|
| `handler` | HTTP request parsing, response serialization, route registration |
| `service` | Business logic, orchestration, validation |
| `repository` | Data access abstraction over EntGo |
| `model` | Domain entity definitions |
| `middleware` | Cross-cutting concerns: auth, logging, recovery, CORS |
| `internal/config` | Singleton accessors: config, logger, database client |

---

## 3. API — Go Backend

### 3.1 Technology Stack

| Dependency | Version | Purpose |
|------------|---------|---------|
| `github.com/gofiber/fiber/v3` | v3.x | HTTP framework |
| `github.com/spf13/viper` | latest | Configuration management |
| `github.com/spf13/cobra` | latest | CLI framework |
| `log/slog` | stdlib | Structured logging |
| `entgo.io/ent` | latest | ORM / schema & migration |
| `go.uber.org/mock` | latest | Mock generation for unit tests |
| `github.com/BurntSushi/toml` | latest | TOML config parsing |
| Taskfile | v3 | Task automation |

### 3.2 Project Structure

```
opus-api/
├── cmd/
│   └── opus/
│       ├── main.go              # Entry point
│       ├── root.go              # Cobra root command
│       ├── start.go             # `opus start` command
│       ├── stop.go              # `opus stop` command
│       ├── restart.go           # `opus restart` command
│       ├── status.go            # `opus status` command
│       └── logs.go              # `opus logs` command
├── internal/
│   ├── model/
│   │   ├── user.go              # User domain entity
│   │   └── session.go           # Session / refresh token entity
│   ├── service/
│   │   ├── auth.go              # Auth business logic
│   │   └── user.go              # User business logic
│   ├── repository/
│   │   ├── user.go              # User data access (EntGo)
│   │   └── session.go           # Session data access (EntGo)
│   ├── handler/
│   │   ├── auth.go              # Auth HTTP handlers
│   │   ├── user.go              # User HTTP handlers
│   │   └── health.go            # Health check handler
│   ├── middleware/
│   │   ├── auth.go              # JWT validation middleware
│   │   ├── logger.go            # Request logging middleware
│   │   └── recovery.go          # Panic recovery middleware
│   └── config/
│       ├── config.go            # GetConfig() — Viper + TOML loader
│       ├── logger.go            # GetLogger() — slog instance
│       └── database.go          # GetDatabase() — EntGo client
├── ent/                         # EntGo generated code (do not edit manually)
│   ├── schema/
│   │   ├── user.go
│   │   └── session.go
│   └── ...
├── Taskfile.yml
├── .env.example
└── go.mod
```

### 3.3 Clean Architecture Layers

Dependency direction is strictly inward. Outer layers depend on inner layers; inner layers have no knowledge of outer layers.

```mermaid
graph TD
    subgraph "Layers"
        HANDLER["handler/<br/>HTTP I/O, framework-aware"]
        SERVICE["service/<br/>Business logic, framework-agnostic"]
        REPO["repository/<br/>Data access, EntGo-aware"]
        MODEL["model/<br/>Pure domain entities, no dependencies"]
    end

    HANDLER --> SERVICE
    SERVICE --> REPO
    REPO --> MODEL
```

**Naming Convention:**

- Directories: singular (`model`, `service`, `repository`, `handler`)
- Files: singular, named after the domain entity (`user.go`, `session.go`)
- Structs: singular (`User`, `Session`)
- Interfaces (repository): defined in `service/` layer, implemented in `repository/`

**Example interface pattern:**

```go
// internal/service/auth.go
type UserRepository interface {
    FindByEmail(ctx context.Context, email string) (*model.User, error)
    Create(ctx context.Context, user *model.User) (*model.User, error)
}

// internal/repository/user.go
type userRepository struct {
    client *ent.Client
}

func (r *userRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
    // EntGo query
}
```

### 3.4 Configuration System

#### Hierarchy

```
OPUS_* environment variables
        ↓  (overrides)
~/.opus/config.toml
        ↓  (overrides)
built-in defaults
```

Environment variables always win. This guarantees Docker and CI/CD deployments behave predictably regardless of any local config file.

#### TOML Structure

```toml
[server]
port    = 8080
env     = "production"   # "development" | "production"

[database]
driver  = "sqlite"       # "sqlite" | "postgres"
dsn     = "~/.opus/opus.db"

[auth]
secret           = ""    # JWT signing secret — REQUIRED
access_token_ttl = 15    # minutes
refresh_token_ttl = 10080 # minutes (7 days)

[auth.google]
client_id     = ""
client_secret = ""
redirect_url  = "http://localhost:8080/auth/google/callback"

[auth.github]
client_id     = ""
client_secret = ""
redirect_url  = "http://localhost:8080/auth/github/callback"
```

#### Environment Variable Mapping

Nested keys are mapped using underscores. Examples:

| TOML Key | Environment Variable |
|----------|---------------------|
| `server.port` | `OPUS_SERVER_PORT` |
| `server.env` | `OPUS_SERVER_ENV` |
| `database.driver` | `OPUS_DATABASE_DRIVER` |
| `database.dsn` | `OPUS_DATABASE_DSN` |
| `auth.secret` | `OPUS_AUTH_SECRET` |
| `auth.google.client_id` | `OPUS_AUTH_GOOGLE_CLIENT_ID` |
| `auth.google.client_secret` | `OPUS_AUTH_GOOGLE_CLIENT_SECRET` |
| `auth.github.client_id` | `OPUS_AUTH_GITHUB_CLIENT_ID` |
| `auth.github.client_secret` | `OPUS_AUTH_GITHUB_CLIENT_SECRET` |

#### Singleton Accessors (`internal/config/`)

```go
// internal/config/config.go
func GetConfig() *Config

// internal/config/logger.go
func GetLogger() *slog.Logger

// internal/config/database.go
func GetDatabase() *ent.Client
```

All singletons are initialized once at startup and are safe for concurrent read access.

### 3.5 Authentication & Session Management

#### OAuth2 Flow

```mermaid
sequenceDiagram
    participant Browser
    participant API as opus-api
    participant Provider as OAuth Provider

    Browser->>API: GET /auth/{provider}
    API-->>Browser: Redirect to provider
    Browser->>Provider: Authorize
    Provider-->>Browser: Redirect with code
    Browser->>API: GET /auth/{provider}/callback?code=...
    API->>Provider: Exchange code for token
    Provider-->>API: User info
    API->>API: Upsert user in DB
    API->>API: Issue access + refresh token
    API-->>Browser: Set-Cookie (refresh) + JSON (access)
```

#### Token Strategy

| Token | Storage | TTL | Notes |
|-------|---------|-----|-------|
| Access Token | Memory (JS) | 15 minutes | JWT, signed with `auth.secret` |
| Refresh Token | HttpOnly Cookie | 7 days | Opaque, hashed in DB |

#### Refresh Token Rotation

1. Client sends expired access token + refresh token cookie.
2. Server validates refresh token against DB hash.
3. Server invalidates old refresh token.
4. Server issues new access token + new refresh token.
5. If refresh token is replayed (already invalidated), entire session family is revoked.

#### Email/Password (Development Only)

Enabled only when `OPUS_SERVER_ENV=development`. The handler returns `403 Forbidden` in production mode.

### 3.6 Database Layer

#### Dual-Driver Support

EntGo supports both SQLite and PostgreSQL via the same generated client. The driver is selected at startup:

```go
// internal/config/database.go
func GetDatabase() *ent.Client {
    cfg := GetConfig()
    switch cfg.Database.Driver {
    case "sqlite":
        client, _ = ent.Open("sqlite3", cfg.Database.DSN+"?_fk=1")
    case "postgres":
        client, _ = ent.Open("postgres", cfg.Database.DSN)
    }
    // Run migrations
    client.Schema.Create(context.Background())
    return client
}
```

Both drivers are **first-class citizens**. SQLite is a valid choice for production in single-user, lightweight deployments.

#### EntGo Schema Location

```
ent/schema/
├── user.go       # User schema
└── session.go    # Refresh token / session schema
```

### 3.7 API Contracts

**Base URL:** `/api/v1`

**Standard Response Envelope:**

```json
{
  "success": true,
  "data": { },
  "error": null
}
```

**Error Response:**

```json
{
  "success": false,
  "data": null,
  "error": {
    "code": "UNAUTHORIZED",
    "message": "Access token is expired or invalid."
  }
}
```

#### Authentication Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET` | `/auth/google` | No | Redirect to Google OAuth2 |
| `GET` | `/auth/google/callback` | No | Google OAuth2 callback |
| `GET` | `/auth/github` | No | Redirect to GitHub OAuth2 |
| `GET` | `/auth/github/callback` | No | GitHub OAuth2 callback |
| `POST` | `/auth/login` | No | Email/Password login (dev only) |
| `POST` | `/auth/refresh` | No | Refresh access token |
| `POST` | `/auth/logout` | Yes | Invalidate refresh token |

#### User Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET` | `/user/me` | Yes | Get current authenticated user |

#### System Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET` | `/health` | No | Health check |

#### SSE Endpoint

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET` | `/stream` | Yes | Server-Sent Events stream |

SSE endpoint streams events using the standard `text/event-stream` content type. Serwist (Service Worker) does not interfere with SSE streams as they are handled natively by the browser's `EventSource` API.

### 3.8 Testing Strategy

#### Unit Tests

- Location: `internal/*/<file>_test.go`
- Framework: standard `testing` package
- Mocks: generated via `go.uber.org/mock/mockgen`
- Scope: `service/` layer — repository interfaces are mocked

```go
// Example: internal/service/auth_test.go
mockRepo := mocks.NewMockUserRepository(ctrl)
mockRepo.EXPECT().FindByEmail(ctx, "test@example.com").Return(user, nil)
```

#### Integration Tests

- Location: `internal/repository/<file>_integration_test.go`
- Database: SQLite in-memory (`file::memory:?cache=shared&_fk=1`)
- Scope: full `repository → EntGo → SQLite` stack
- No mocks; real EntGo client with real queries

```go
// Build tag: //go:build integration
client, _ := ent.Open("sqlite3", "file::memory:?cache=shared&_fk=1")
client.Schema.Create(ctx)
```

#### Running Tests

```bash
# Unit tests only
task test

# Integration tests
task test:integration

# All tests
task test:all
```

### 3.9 Task Automation

`Taskfile.yml` defines the following tasks:

| Task | Description |
|------|-------------|
| `task dev` | Run API in development mode with live reload |
| `task build` | Build production binary |
| `task test` | Run unit tests |
| `task test:integration` | Run integration tests |
| `task test:all` | Run all tests |
| `task ent:generate` | Regenerate EntGo code from schema |
| `task migrate` | Run database migrations |
| `task lint` | Run golangci-lint |

---

## 4. Frontend — Next.js

### 4.1 Technology Stack

| Dependency | Version | Purpose |
|------------|---------|---------|
| Next.js | 16.x | React framework with App Router |
| Serwist | latest | Service Worker / PWA management |
| Shadcn/ui | latest | Accessible UI component library |
| Tailwind CSS | v4.x | Utility-first CSS framework |
| TanStack Query | v5.x | Server state management & data fetching |
| Vitest | latest | Unit & component testing |
| Playwright | latest | End-to-end testing |

### 4.2 Project Structure

```
opus-web/
├── app/
│   ├── (auth)/
│   │   ├── login/
│   │   │   └── page.tsx         # Login page
│   │   └── layout.tsx           # Auth layout
│   ├── (dashboard)/
│   │   ├── page.tsx             # Main dashboard
│   │   └── layout.tsx           # Dashboard layout (protected)
│   ├── offline/
│   │   └── page.tsx             # PWA offline fallback page
│   ├── layout.tsx               # Root layout
│   └── globals.css
├── components/
│   ├── ui/                      # Shadcn/ui generated components
│   └── shared/                  # Application-level shared components
│       ├── stream-output.tsx    # SSE streaming output component
│       └── auth-guard.tsx       # Client-side route protection
├── lib/
│   ├── api/
│   │   ├── client.ts            # Axios / fetch base client
│   │   ├── auth.ts              # Auth query hooks (TanStack Query)
│   │   └── user.ts              # User query hooks (TanStack Query)
│   └── utils/
│       └── cn.ts                # Tailwind class merge utility
├── public/
│   ├── manifest.webmanifest     # PWA manifest
│   └── icons/                   # PWA icons (192x192, 512x512)
├── sw.ts                        # Serwist Service Worker entry point
├── next.config.ts
├── tailwind.config.ts
├── vitest.config.ts
└── playwright.config.ts
```

### 4.3 PWA Configuration

PWA is managed entirely by **Serwist**, which wraps Workbox and integrates with Next.js 16's App Router.

#### Service Worker Strategy

| Route Pattern | Strategy | Notes |
|---------------|----------|-------|
| API calls (`/api/*`) | Network Only | Never cache API responses |
| Static assets | Stale While Revalidate | CSS, JS, fonts |
| Pages | Network First | Fall back to cache |
| Offline fallback | Cache First | `/offline` page always available |

#### PWA Manifest (`manifest.webmanifest`)

```json
{
  "name": "Opus",
  "short_name": "Opus",
  "description": "Your 24/7 autonomous AI assistant",
  "start_url": "/",
  "display": "standalone",
  "background_color": "#ffffff",
  "theme_color": "#000000",
  "icons": [
    { "src": "/icons/icon-192.png", "sizes": "192x192", "type": "image/png" },
    { "src": "/icons/icon-512.png", "sizes": "512x512", "type": "image/png" }
  ]
}
```

### 4.4 State Management & Data Fetching

- **TanStack Query** handles all server state: fetching, caching, invalidation, and background refetching.
- No global client-side state manager (Zustand/Redux) in v1.0. React context is used for lightweight UI state where needed.
- SSE streaming is handled via the browser-native `EventSource` API within a custom React hook, not via TanStack Query.

```typescript
// lib/api/useStream.ts — conceptual
export function useStream(enabled: boolean) {
  const [output, setOutput] = useState<string>("");
  useEffect(() => {
    if (!enabled) return;
    const es = new EventSource("/stream", { withCredentials: true });
    es.onmessage = (e) => setOutput((prev) => prev + e.data);
    return () => es.close();
  }, [enabled]);
  return output;
}
```

### 4.5 Testing Strategy

#### Unit & Component Tests (Vitest)

- Framework: Vitest + React Testing Library
- Scope: utility functions, custom hooks, individual components
- Location: co-located `*.test.tsx` files or `__tests__/` directory

#### End-to-End Tests (Playwright)

- Framework: Playwright
- Scope: critical user flows
- Location: `e2e/` directory

**Covered E2E flows (v1.0):**

| Test | Description |
|------|-------------|
| `auth.spec.ts` | Login via Google OAuth2 (mocked), GitHub OAuth2 (mocked), logout |
| `dashboard.spec.ts` | Authenticated dashboard render, protected route redirect |
| `stream.spec.ts` | SSE stream connection and output rendering |
| `pwa.spec.ts` | PWA installability, offline fallback page |

#### Running Tests

```bash
# Unit tests
pnpm test

# Unit tests with coverage
pnpm test:coverage

# E2E tests
pnpm test:e2e

# E2E tests with UI
pnpm test:e2e:ui
```

---

## 5. Deployment & Distribution

### 5.1 Docker

The Docker image is built from a multi-stage Dockerfile. Environment variables prefixed with `OPUS_` always override `config.toml`.

**Docker Compose (example):**

```yaml
services:
  opus-api:
    image: ghcr.io/opus/opus-api:latest
    environment:
      OPUS_SERVER_PORT: "8080"
      OPUS_SERVER_ENV: "production"
      OPUS_DATABASE_DRIVER: "postgres"
      OPUS_DATABASE_DSN: "postgres://opus:secret@db:5432/opus"
      OPUS_AUTH_SECRET: "your-secret-here"
      OPUS_AUTH_GOOGLE_CLIENT_ID: "..."
      OPUS_AUTH_GOOGLE_CLIENT_SECRET: "..."
    ports:
      - "8080:8080"
    depends_on:
      - db

  db:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: opus
      POSTGRES_PASSWORD: secret
      POSTGRES_DB: opus
    volumes:
      - pgdata:/var/lib/postgresql/data

volumes:
  pgdata:
```

**SQLite with Docker (single-file, lightweight):**

```yaml
services:
  opus-api:
    image: ghcr.io/opus/opus-api:latest
    environment:
      OPUS_DATABASE_DRIVER: "sqlite"
      OPUS_DATABASE_DSN: "/data/opus.db"
    volumes:
      - ./data:/data
```

### 5.2 Bare Metal

Pre-built binaries are published to GitHub Releases for the following targets:

| OS | Architecture | Binary |
|----|-------------|--------|
| Linux | amd64 | `opus-linux-amd64` |
| Linux | arm64 | `opus-linux-arm64` |
| macOS | amd64 | `opus-darwin-amd64` |
| macOS | arm64 | `opus-darwin-arm64` |
| Windows | amd64 | `opus-windows-amd64.exe` |

**Manual installation:**

```bash
# Download binary
curl -L https://github.com/opus/opus/releases/latest/download/opus-linux-amd64 -o opus
chmod +x opus
sudo mv opus /usr/local/bin/

# Initialize configuration
opus init

# Start
opus start
```

### 5.3 npx Installer

`npx opus install` provides an interactive setup wizard targeting end-users.

**Prerequisites:** Node.js LTS (user-facing requirement).

**Installer flow:**

```
$ npx opus install

  ██████╗ ██████╗ ██╗   ██╗███████╗
 ██╔═══██╗██╔══██╗██║   ██║██╔════╝
 ██║   ██║██████╔╝██║   ██║███████╗
 ██║   ██║██╔═══╝ ██║   ██║╚════██║
 ╚██████╔╝██║     ╚██████╔╝███████║
  ╚═════╝ ╚═╝      ╚═════╝ ╚══════╝

  Opus Installer v1.0.0

[1/5] Detecting platform...          linux/amd64
[2/5] Downloading binary...          ████████████ 100%
[3/5] Configuring...

  ? Database driver:           › sqlite / postgres
  ? Database path (sqlite):    › ~/.opus/opus.db
  ? Server port:               › 8080
  ? Google Client ID:          › (leave blank to skip)
  ? GitHub Client ID:          › (leave blank to skip)
  ? JWT Secret:                › (auto-generated)

[4/5] Writing ~/.opus/config.toml... ✓
[5/5] Installing system service...   ✓ (systemd)

  ✓ Opus is installed and running!
  ✓ Open http://localhost:8080 to get started.

  Manage with: opus start | stop | restart | status | logs
```

**Installer implementation notes:**

- Published to npm as `opus` package with a `bin` entry.
- Downloads the correct binary from GitHub Releases based on `process.platform` + `process.arch`.
- Generates `~/.opus/config.toml` from interactive prompts.
- Auto-generates a cryptographically random `auth.secret` if not provided.
- Registers the system service (see [5.5 Auto-Restart](#55-auto-restart)).

### 5.4 CLI Commands

Implemented via Cobra in `cmd/opus/`:

| Command | Description |
|---------|-------------|
| `opus start` | Start the Opus service (or via system service manager) |
| `opus stop` | Stop the Opus service |
| `opus restart` | Restart the Opus service |
| `opus status` | Display current service status and uptime |
| `opus logs` | Tail live log output |
| `opus init` | Initialize `~/.opus/config.toml` interactively |
| `opus version` | Print current version |

### 5.5 Auto-Restart

The installer registers Opus as a persistent system service to survive reboots and crashes.

| Platform | Service Manager | Unit/Plist File |
|----------|----------------|-----------------|
| Linux | systemd | `/etc/systemd/system/opus.service` |
| macOS | launchd | `~/Library/LaunchAgents/com.opus.agent.plist` |
| Windows | Windows Service | Registered via `sc.exe` |

**systemd unit (Linux):**

```ini
[Unit]
Description=Opus AI Agent
After=network.target

[Service]
ExecStart=/usr/local/bin/opus serve
Restart=always
RestartSec=5
User=%i
Environment=HOME=/home/%i

[Install]
WantedBy=multi-user.target
```

---

## 6. Security

| Concern | Mitigation |
|---------|-----------|
| JWT secret exposure | Loaded from config, never logged, auto-generated on install |
| Refresh token theft | HttpOnly + Secure + SameSite=Strict cookie; hashed in DB |
| Refresh token replay | Rotation with family revocation on replay detection |
| Email/Password in production | Handler returns `403 Forbidden` when `env != development` |
| CORS | Configured via GoFiber CORS middleware; restricted to known origins |
| SQL Injection | Prevented by EntGo parameterised queries |
| Path traversal | Not applicable; no user-controlled file paths |

---

## 7. Observability

### Logging

All structured logs use Go's standard `log/slog` package.

| Environment | Format | Level |
|-------------|--------|-------|
| `development` | Text (human-readable) | `DEBUG` |
| `production` | JSON | `INFO` |

Log fields included on every request:

```json
{
  "time": "2026-05-15T08:00:00Z",
  "level": "INFO",
  "msg": "request",
  "method": "GET",
  "path": "/user/me",
  "status": 200,
  "latency_ms": 3,
  "request_id": "abc123"
}
```

### Health Check

`GET /health` returns:

```json
{
  "success": true,
  "data": {
    "status": "ok",
    "version": "1.0.0",
    "db": "sqlite"
  }
}
```

---

## 8. Reserved: pkg/ Module

The `pkg/` directory is intentionally absent in v1.0. It is reserved for future use as a shared utilities module in the event that the project grows into multiple Go modules or requires shared code between the API server and the CLI installer.

**When to introduce `pkg/`:**

- A utility is needed by two or more independent Go modules.
- A library is mature enough to be versioned independently.
- A capability is generic enough to be useful outside the Opus project.

Until that threshold is reached, all shared utilities shall reside in `internal/config/` or adjacent `internal/` packages.