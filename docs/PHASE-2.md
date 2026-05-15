# Phase 2 — API Foundation (Go Backend)

**Product:** Opus  
**Version:** 1.0.1  
**Status:** Draft  
**Last Updated:** 2026-05-15  
**Authors:** Product & Architecture Team

---

## Phase Goal

Implement the complete Go backend: module initialization, configuration system, database layer (EntGo), repository layer, service layer, middleware, HTTP handlers, SSE endpoint, Cobra CLI commands, and full test coverage. The output is a fully functional API that passes all unit and integration tests.

---

## Prerequisites

- Phase 1 is complete.
- Go 1.23+ is installed.
- Task (Taskfile runner) is installed.
- For integration tests: SQLite3 driver libraries are available.

---

## Context for AI Agent

> Always provide `docs/CONVENTIONS.md`, `docs/PRD.md`, and `docs/ARCHITECTURE.md` alongside this file.

**You are implementing the Go backend for Opus (`api/`).** Follow Clean Architecture strictly: `handler → service → repository → model`. Dependencies flow inward only. The configuration singleton (`internal/config/`) is initialized once at startup and accessed everywhere via `GetConfig()`, `GetLogger()`, `GetDatabase()`.

The API base URL is `/api/v1`. All responses use the standard envelope: `{ success, data, error }`.

---

## P2-T1 — Initialize Go Module and Install Dependencies

### What to Do

Initialize the Go module for `api/` and install all required dependencies. Create `api/go.mod` and run `go mod tidy`.

### Commands to Run

```bash
cd api/
go mod init github.com/kilip/opus/api
go get github.com/gofiber/fiber/v3
go get github.com/spf13/viper
go get github.com/spf13/cobra
go get github.com/BurntSushi/toml
go get entgo.io/ent/cmd/ent
go get go.uber.org/mock/mockgen
go get github.com/mattn/go-sqlite3
go get github.com/lib/pq
go mod tidy
```

### `api/Taskfile.yml` Scaffold

Create a minimal `api/Taskfile.yml` that will be expanded in P2-T13:

```yaml
version: "3"

tasks:
  setup:
    desc: Install Go module dependencies
    cmds:
      - go mod download

  dev:
    desc: Run API in development mode
    cmds:
      - go run ./cmd/opus/... start

  build:
    desc: Build production binary
    cmds:
      - go build -o bin/opus ./cmd/opus

  test:
    desc: Run unit tests
    cmds:
      - go test ./internal/...

  test:integration:
    desc: Run integration tests
    cmds:
      - go test -tags=integration ./internal/...

  test:all:
    desc: Run all tests
    deps: [test, test:integration]

  lint:
    desc: Run golangci-lint
    cmds:
      - golangci-lint run ./...

  ent:generate:
    desc: Regenerate EntGo code from schema
    cmds:
      - go generate ./ent/...

  migrate:
    desc: Run database migrations
    cmds:
      - go run ./cmd/opus/... migrate
```

### Acceptance Criteria

- [x] File `api/go.mod` exists with module name `github.com/kilip/opus/api`.
- [x] `github.com/gofiber/fiber/v3` is listed in `go.mod`.
- [x] `github.com/spf13/viper` is listed in `go.mod`.
- [x] `github.com/spf13/cobra` is listed in `go.mod`.
- [x] `entgo.io/ent` is listed in `go.mod`.
- [x] `go.uber.org/mock` is listed in `go.mod`.
- [x] `github.com/mattn/go-sqlite3` is listed in `go.mod`.
- [x] File `api/Taskfile.yml` exists with all tasks defined.
- [x] Running `task setup` from `api/` completes without errors.

---

## P2-T2 — Implement Configuration System (`internal/config/`)

### What to Do

Implement three singleton accessor files in `api/internal/config/`. These are the only way to access config, logger, and database throughout the application.

### Files to Create

- `api/internal/config/config.go`
- `api/internal/config/logger.go`
- `api/internal/config/database.go`

### `config.go` — Viper + TOML Loader

```go
// api/internal/config/config.go
package config

import (
    "sync"
    "github.com/spf13/viper"
)

type Config struct {
    Server   ServerConfig
    Database DatabaseConfig
    Auth     AuthConfig
}

type ServerConfig struct {
    Port int
    Env  string
}

type DatabaseConfig struct {
    Driver string
    DSN    string
}

type AuthConfig struct {
    Secret          string
    AccessTokenTTL  int
    RefreshTokenTTL int
    Google          OAuthConfig
    GitHub          OAuthConfig
}

type OAuthConfig struct {
    ClientID     string
    ClientSecret string
    RedirectURL  string
}

var (
    cfg     *Config
    cfgOnce sync.Once
)

func GetConfig() *Config {
    cfgOnce.Do(func() {
        viper.SetConfigName("config")
        viper.SetConfigType("toml")
        viper.AddConfigPath("$HOME/.opus")
        viper.SetEnvPrefix("OPUS")
        viper.AutomaticEnv()

        // Defaults
        viper.SetDefault("server.port", 8080)
        viper.SetDefault("server.env", "production")
        viper.SetDefault("database.driver", "sqlite")
        viper.SetDefault("database.dsn", "$HOME/.opus/opus.db")
        viper.SetDefault("auth.access_token_ttl", 15)
        viper.SetDefault("auth.refresh_token_ttl", 10080)

        _ = viper.ReadInConfig()

        cfg = &Config{}
        _ = viper.Unmarshal(cfg)
    })
    return cfg
}
```

### `logger.go` — slog Instance

```go
// api/internal/config/logger.go
package config

import (
    "log/slog"
    "os"
    "sync"
)

var (
    logger     *slog.Logger
    loggerOnce sync.Once
)

func GetLogger() *slog.Logger {
    loggerOnce.Do(func() {
        cfg := GetConfig()
        var handler slog.Handler
        if cfg.Server.Env == "development" {
            handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
                Level: slog.LevelDebug,
            })
        } else {
            handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
                Level: slog.LevelInfo,
            })
        }
        logger = slog.New(handler)
    })
    return logger
}
```

### `database.go` — EntGo Client

```go
// api/internal/config/database.go
package config

import (
    "context"
    "log"
    "sync"

    "github.com/kilip/opus/api/ent"
    _ "github.com/mattn/go-sqlite3"
    _ "github.com/lib/pq"
)

var (
    dbClient *ent.Client
    dbOnce   sync.Once
)

func GetDatabase() *ent.Client {
    dbOnce.Do(func() {
        cfg := GetConfig()
        var err error
        switch cfg.Database.Driver {
        case "sqlite":
            dbClient, err = ent.Open("sqlite3", cfg.Database.DSN+"?_fk=1")
        case "postgres":
            dbClient, err = ent.Open("postgres", cfg.Database.DSN)
        default:
            log.Fatalf("unsupported database driver: %s", cfg.Database.Driver)
        }
        if err != nil {
            log.Fatalf("failed to connect to database: %v", err)
        }
        if err := dbClient.Schema.Create(context.Background()); err != nil {
            log.Fatalf("failed to run migrations: %v", err)
        }
    })
    return dbClient
}
```

### Constraints

- All three files must use `sync.Once` — singletons are initialized only once.
- `GetConfig()` must be called before `GetLogger()` and `GetDatabase()`.
- Never use `viper.Get()` directly outside of `config.go`.
- `GetDatabase()` must run `Schema.Create()` on initialization (auto-migration).

### Acceptance Criteria

- [x] File `api/internal/config/config.go` exists.
- [x] File `api/internal/config/logger.go` exists.
- [x] File `api/internal/config/database.go` exists.
- [x] `GetConfig()` uses `sync.Once` and returns a `*Config` struct.
- [x] `GetConfig()` sets `OPUS_` env prefix and calls `AutomaticEnv()`.
- [x] `GetConfig()` reads from `$HOME/.opus/config.toml`.
- [x] `GetConfig()` has defaults for all non-required fields.
- [x] `GetLogger()` returns a `*slog.Logger` using `slog.NewTextHandler` in development and `slog.NewJSONHandler` in production.
- [x] `GetDatabase()` supports both `sqlite` and `postgres` drivers.
- [x] `GetDatabase()` runs `Schema.Create()` on startup.
- [x] All three use `sync.Once` for singleton initialization.
- [x] Code compiles without errors (`go build ./internal/config/...`).

---

## P2-T3 — Define EntGo Schemas (`User`, `Session`)

### What to Do

Define two EntGo schemas: `User` and `Session`. Then run `go generate` to produce the EntGo client code.

### Files to Create

- `api/ent/schema/user.go`
- `api/ent/schema/session.go`

### `user.go` Schema

```go
// api/ent/schema/user.go
package schema

import (
    "time"
    "entgo.io/ent"
    "entgo.io/ent/schema/field"
    "entgo.io/ent/schema/edge"
)

type User struct {
    ent.Schema
}

func (User) Fields() []ent.Field {
    return []ent.Field{
        field.String("id").Unique().Immutable(),
        field.String("email").Unique(),
        field.String("name"),
        field.String("avatar_url").Optional(),
        field.String("provider"),          // "google" | "github" | "email"
        field.String("provider_id").Optional(),
        field.Time("created_at").Default(time.Now).Immutable(),
        field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
    }
}

func (User) Edges() []ent.Edge {
    return []ent.Edge{
        edge.To("sessions", Session.Type),
    }
}
```

### `session.go` Schema

```go
// api/ent/schema/session.go
package schema

import (
    "time"
    "entgo.io/ent"
    "entgo.io/ent/schema/field"
    "entgo.io/ent/schema/edge"
)

type Session struct {
    ent.Schema
}

func (Session) Fields() []ent.Field {
    return []ent.Field{
        field.String("id").Unique().Immutable(),
        field.String("token_hash").Unique(),   // Hashed refresh token
        field.String("user_id"),
        field.Time("expires_at"),
        field.Bool("revoked").Default(false),
        field.Time("created_at").Default(time.Now).Immutable(),
    }
}

func (Session) Edges() []ent.Edge {
    return []ent.Edge{
        edge.From("user", User.Type).Ref("sessions").Field("user_id").Unique().Required(),
    }
}
```

### Commands to Run After Creating Schemas

```bash
cd api/
go generate ./ent/...
```

### Constraints

- Do not manually edit any file inside `api/ent/` except `api/ent/schema/`.
- `token_hash` stores a bcrypt or SHA-256 hash of the refresh token — never the raw token.
- `provider` field values: `"google"`, `"github"`, `"email"`.

### Acceptance Criteria

- [x] File `api/ent/schema/user.go` exists.
- [x] File `api/ent/schema/session.go` exists.
- [x] `User` schema has fields: `id`, `email`, `name`, `avatar_url`, `provider`, `provider_id`, `created_at`, `updated_at`.
- [x] `Session` schema has fields: `id`, `token_hash`, `user_id`, `expires_at`, `revoked`, `created_at`.
- [x] `User` has edge `To("sessions", Session.Type)`.
- [x] `Session` has edge `From("user", User.Type)`.
- [x] Running `go generate ./ent/...` completes without errors.
- [x] `api/ent/` directory contains generated client code after `go generate`.

---

## P2-T4 — Implement Repository Layer (`user`, `session`)

### What to Do

Implement the data access layer. Interfaces are defined here but declared in `service/` (as per Clean Architecture). The repository structs implement those interfaces using the EntGo client.

### Files to Create

- `api/internal/repository/user.go`
- `api/internal/repository/session.go`

### `repository/user.go`

Implement a `userRepository` struct with the following methods:

| Method | Signature |
|--------|-----------|
| `FindByID` | `(ctx, id string) (*model.User, error)` |
| `FindByEmail` | `(ctx, email string) (*model.User, error)` |
| `FindByProviderID` | `(ctx, provider, providerID string) (*model.User, error)` |
| `Create` | `(ctx, user *model.User) (*model.User, error)` |
| `Update` | `(ctx, user *model.User) (*model.User, error)` |

Also implement a constructor: `func NewUserRepository(client *ent.Client) UserRepository`

### `repository/session.go`

Implement a `sessionRepository` struct with the following methods:

| Method | Signature |
|--------|-----------|
| `Create` | `(ctx, session *model.Session) (*model.Session, error)` |
| `FindByTokenHash` | `(ctx, hash string) (*model.Session, error)` |
| `RevokeByID` | `(ctx, id string) error` |
| `RevokeAllByUserID` | `(ctx, userID string) error` |

Also implement a constructor: `func NewSessionRepository(client *ent.Client) SessionRepository`

### Domain Models

Before writing repository code, create model files:

**`api/internal/model/user.go`**

```go
package model

import "time"

type User struct {
    ID         string
    Email      string
    Name       string
    AvatarURL  string
    Provider   string
    ProviderID string
    CreatedAt  time.Time
    UpdatedAt  time.Time
}
```

**`api/internal/model/session.go`**

```go
package model

import "time"

type Session struct {
    ID        string
    TokenHash string
    UserID    string
    ExpiresAt time.Time
    Revoked   bool
    CreatedAt time.Time
}
```

**`api/internal/model/errors.go`**

```go
package model

import "errors"

var (
    ErrUserNotFound    = errors.New("user not found")
    ErrSessionNotFound = errors.New("session not found")
    ErrInvalidToken    = errors.New("invalid token")
    ErrTokenExpired    = errors.New("token expired")
    ErrTokenRevoked    = errors.New("token revoked")
)
```

### Constraints

- Repository structs must NOT be exported — only the interface and constructor are exported.
- Every method must wrap errors with `fmt.Errorf("userRepository.FindByEmail: %w", err)`.
- Map EntGo entities to `model.*` types inside repository methods — never return raw `*ent.User`.

### Acceptance Criteria

- [x] File `api/internal/model/user.go` exists with `User` struct.
- [x] File `api/internal/model/session.go` exists with `Session` struct.
- [x] File `api/internal/model/errors.go` exists with all sentinel errors.
- [x] File `api/internal/repository/user.go` exists.
- [x] File `api/internal/repository/session.go` exists.
- [x] `userRepository` implements all 5 methods.
- [x] `sessionRepository` implements all 4 methods.
- [x] Both constructors return the interface type, not the concrete struct.
- [x] No `*ent.User` or `*ent.Session` types are returned from repository methods.
- [x] All methods wrap errors with `fmt.Errorf`.
- [x] Code compiles without errors.

---

## P2-T5 — Implement Auth Service (`internal/service/auth.go`)

### What to Do

Implement the authentication business logic. This service defines the repository interfaces it depends on, implements JWT access token issuance, refresh token generation, rotation, and OAuth2 user upsert logic.

### File to Create

`api/internal/service/auth.go`

### Repository Interfaces (defined in service layer)

```go
// Defined in service/auth.go
type UserRepository interface {
    FindByEmail(ctx context.Context, email string) (*model.User, error)
    FindByProviderID(ctx context.Context, provider, providerID string) (*model.User, error)
    Create(ctx context.Context, user *model.User) (*model.User, error)
    Update(ctx context.Context, user *model.User) (*model.User, error)
}

type SessionRepository interface {
    Create(ctx context.Context, session *model.Session) (*model.Session, error)
    FindByTokenHash(ctx context.Context, hash string) (*model.Session, error)
    RevokeByID(ctx context.Context, id string) error
    RevokeAllByUserID(ctx context.Context, userID string) error
}
```

### `AuthService` Methods to Implement

| Method | Description |
|--------|-------------|
| `UpsertOAuthUser(ctx, provider, providerID, email, name, avatarURL string) (*model.User, error)` | Find or create user from OAuth2 provider data |
| `IssueTokens(ctx, userID string) (accessToken, refreshToken string, err error)` | Issue JWT access token + opaque refresh token, store hashed refresh token |
| `RefreshTokens(ctx, rawRefreshToken string) (accessToken, refreshToken string, err error)` | Validate refresh token, rotate (revoke old, issue new) |
| `Logout(ctx, rawRefreshToken string) error` | Revoke the refresh token |
| `ValidateAccessToken(tokenString string) (userID string, err error)` | Validate JWT and return user ID |

### Token Strategy

- **Access Token:** JWT, signed with HS256 using `config.Auth.Secret`. Claims: `sub` (userID), `exp`, `iat`.
- **Refresh Token:** 32 bytes of `crypto/rand`, base64url-encoded. Stored as SHA-256 hash in the database.
- **Rotation:** On refresh, revoke the old session and create a new one. If the old token is already revoked (replay attack), call `RevokeAllByUserID` for that user.

### Constructor

```go
func NewAuthService(userRepo UserRepository, sessionRepo SessionRepository, cfg *config.Config) *AuthService
```

### Constraints

- Use `crypto/rand` for refresh token generation — never `math/rand`.
- Use `golang-jwt/jwt` or the standard `crypto` approach for JWT.
- Never log raw refresh tokens or JWT secrets.
- `UpsertOAuthUser` must call `FindByProviderID` first; if not found, call `Create`.

### Acceptance Criteria

- [x] File `api/internal/service/auth.go` exists.
- [x] `UserRepository` and `SessionRepository` interfaces are defined in this file.
- [x] `AuthService` struct implements all 5 methods.
- [x] `IssueTokens` generates a JWT access token signed with HS256.
- [x] `IssueTokens` generates a cryptographically random refresh token.
- [x] `IssueTokens` stores the SHA-256 hash of the refresh token in the database.
- [x] `RefreshTokens` revokes the old session before issuing new tokens.
- [x] `RefreshTokens` calls `RevokeAllByUserID` on replay detection (revoked token used again).
- [x] `UpsertOAuthUser` finds existing user by provider ID before creating.
- [x] No raw tokens or secrets are logged.
- [x] Code compiles without errors.

---

## P2-T6 — Implement User Service (`internal/service/user.go`)

### What to Do

Implement the user business logic service.

### File to Create

`api/internal/service/user.go`

### Repository Interface

```go
type UserReader interface {
    FindByID(ctx context.Context, id string) (*model.User, error)
}
```

### `UserService` Methods to Implement

| Method | Description |
|--------|-------------|
| `GetCurrentUser(ctx context.Context, userID string) (*model.User, error)` | Return user by ID |

### Constructor

```go
func NewUserService(userRepo UserReader, cfg *config.Config) *UserService
```

### Acceptance Criteria

- [x] File `api/internal/service/user.go` exists.
- [x] `UserReader` interface is defined in this file.
- [x] `UserService` implements `GetCurrentUser`.
- [x] Returns `model.ErrUserNotFound` when user does not exist.
- [x] Code compiles without errors.

---

## P2-T7 — Implement Middleware (`auth`, `logger`, `recovery`)

### What to Do

Implement three middleware files used by the GoFiber router.

### Files to Create

- `api/internal/middleware/auth.go`
- `api/internal/middleware/logger.go`
- `api/internal/middleware/recovery.go`

### `middleware/auth.go`

JWT validation middleware. Reads `Authorization: Bearer <token>` header, validates the token using `AuthService.ValidateAccessToken`, and sets `userID` in the Fiber context locals.

```go
func Auth(authService *service.AuthService) fiber.Handler {
    return func(c fiber.Ctx) error {
        // Extract Bearer token
        // Validate via authService.ValidateAccessToken()
        // Set c.Locals("userID", userID)
        // Return 401 on failure
    }
}
```

### `middleware/logger.go`

Request logging middleware using `slog`. Log fields: `method`, `path`, `status`, `latency_ms`, `request_id`.

### `middleware/recovery.go`

Panic recovery middleware. Catches panics, logs them with `slog`, and returns a `500 Internal Server Error` JSON response using the standard envelope.

### Acceptance Criteria

- [x] File `api/internal/middleware/auth.go` exists.
- [x] File `api/internal/middleware/logger.go` exists.
- [x] File `api/internal/middleware/recovery.go` exists.
- [x] Auth middleware extracts Bearer token from `Authorization` header.
- [x] Auth middleware returns 401 with standard error envelope on invalid/missing token.
- [x] Auth middleware sets `userID` in `c.Locals("userID", userID)`.
- [x] Logger middleware logs `method`, `path`, `status`, `latency_ms`, `request_id` using `slog`.
- [x] Recovery middleware catches panics and returns 500 with standard error envelope.
- [x] Code compiles without errors.

---

## P2-T8 — Implement Auth Handlers

### What to Do

Implement HTTP handlers for all authentication endpoints.

### File to Create

`api/internal/handler/auth.go`

### Endpoints to Implement

| Method | Path | Handler Function |
|--------|------|-----------------|
| `GET` | `/auth/google` | `RedirectToGoogle` |
| `GET` | `/auth/google/callback` | `GoogleCallback` |
| `GET` | `/auth/github` | `RedirectToGitHub` |
| `GET` | `/auth/github/callback` | `GitHubCallback` |
| `POST` | `/auth/login` | `EmailPasswordLogin` (dev only) |
| `POST` | `/auth/refresh` | `RefreshToken` |
| `POST` | `/auth/logout` | `Logout` |

### Handler Struct and Constructor

```go
type AuthHandler struct {
    authService *service.AuthService
    cfg         *config.Config
}

func NewAuthHandler(authService *service.AuthService, cfg *config.Config) *AuthHandler
```

### Key Implementation Notes

- `EmailPasswordLogin`: Return `fiber.StatusForbidden` with error `{"code": "FORBIDDEN", "message": "Email/password login is disabled in production."}` when `cfg.Server.Env != "development"`.
- `RefreshToken`: Read refresh token from `HttpOnly` cookie named `refresh_token`. Call `authService.RefreshTokens()`. Set new refresh token as HttpOnly cookie. Return new access token in JSON response body.
- `Logout`: Read refresh token from cookie. Call `authService.Logout()`. Clear the cookie.
- OAuth2 callbacks: Call `authService.UpsertOAuthUser()`, then `authService.IssueTokens()`, then redirect to dashboard with access token.
- All responses use the standard envelope: `{ success, data, error }`.

### Refresh Token Cookie Settings

```go
cookie := &fiber.Cookie{
    Name:     "refresh_token",
    Value:    refreshToken,
    HTTPOnly: true,
    Secure:   cfg.Server.Env == "production",
    SameSite: "Strict",
    MaxAge:   cfg.Auth.RefreshTokenTTL * 60,
}
```

### Constraints

- Never return the raw refresh token in the JSON response body — only in the HttpOnly cookie.
- The access token is returned in the JSON response body only.
- `EmailPasswordLogin` must check `cfg.Server.Env` before processing credentials.

### Acceptance Criteria

- [x] File `api/internal/handler/auth.go` exists.
- [x] All 7 endpoints are implemented.
- [x] `EmailPasswordLogin` returns 403 in production mode.
- [x] `RefreshToken` reads refresh token from cookie and sets a new HttpOnly cookie.
- [x] `Logout` clears the `refresh_token` cookie.
- [x] All responses use the standard envelope.
- [x] Cookie is set with `HTTPOnly: true`, `SameSite: "Strict"`, and `Secure: true` in production.
- [x] Code compiles without errors.

---

## P2-T9 — Implement User Handler (`/user/me`)

### What to Do

Implement the user endpoint handler.

### File to Create

`api/internal/handler/user.go`

### Handler Struct

```go
type UserHandler struct {
    userService *service.UserService
}

func NewUserHandler(userService *service.UserService) *UserHandler
```

### Endpoint

`GET /api/v1/user/me` — Protected by Auth middleware. Reads `userID` from `c.Locals("userID")`. Calls `userService.GetCurrentUser()`. Returns user data in standard envelope.

### Acceptance Criteria

- [x] File `api/internal/handler/user.go` exists.
- [x] Handler reads `userID` from `c.Locals("userID")`.
- [x] Handler returns user data in standard envelope.
- [x] Handler returns 404 with standard error envelope when user is not found.
- [x] Code compiles without errors.

---

## P2-T10 — Implement Health Check Handler

### What to Do

Implement the health check endpoint.

### File to Create

`api/internal/handler/health.go`

### Endpoint

`GET /health` — Unauthenticated. Returns:

```json
{
  "success": true,
  "data": {
    "status": "ok",
    "version": "1.0.1",
    "db": "sqlite"
  },
  "error": null
}
```

### Acceptance Criteria

- [x] File `api/internal/handler/health.go` exists.
- [x] Returns 200 with `status: "ok"`, `version`, and `db` driver name.
- [x] Response uses standard envelope.
- [x] Code compiles without errors.

---

## P2-T11 — Implement SSE Handler (`/stream`)

### What to Do

Implement the Server-Sent Events endpoint. This is a placeholder for the future AI agent integration — it currently sends a heartbeat event every 30 seconds to keep the connection alive.

### File to Create

`api/internal/handler/stream.go`

### Endpoint

`GET /api/v1/stream` — Protected by Auth middleware.

### SSE Implementation

```go
// Use GoFiber's built-in SSE support
// Content-Type: text/event-stream
// Send heartbeat: event: heartbeat, data: {"ts": "<timestamp>"}
// Every 30 seconds
// Respect client disconnect via c.Context().Done()
```

### Acceptance Criteria

- [x] File `api/internal/handler/stream.go` exists.
- [x] Endpoint is protected by Auth middleware.
- [x] Content-Type is `text/event-stream`.
- [x] Sends heartbeat event every 30 seconds.
- [x] Closes gracefully on client disconnect.
- [x] Code compiles without errors.

---

## P2-T12 — Implement Cobra CLI Commands

### What to Do

Implement the CLI entry point using Cobra. Wire up all five service commands and the main server startup.

### Files to Create

- `api/cmd/opus/main.go`
- `api/cmd/opus/root.go`
- `api/cmd/opus/start.go`
- `api/cmd/opus/stop.go`
- `api/cmd/opus/restart.go`
- `api/cmd/opus/status.go`
- `api/cmd/opus/logs.go`

### Wire Up the Router in `start.go`

In `start.go`, the `opus start` command must:

1. Call `config.GetConfig()`, `config.GetLogger()`, `config.GetDatabase()`.
2. Instantiate repositories: `repository.NewUserRepository(db)`, `repository.NewSessionRepository(db)`.
3. Instantiate services: `service.NewAuthService(...)`, `service.NewUserService(...)`.
4. Instantiate handlers: `handler.NewAuthHandler(...)`, `handler.NewUserHandler(...)`, `handler.NewHealthHandler()`, `handler.NewStreamHandler(...)`.
5. Create GoFiber app.
6. Register middleware: recovery, logger, CORS.
7. Register routes under `/api/v1` with Auth middleware on protected routes.
8. Start server on `cfg.Server.Port`.

### Route Registration

```
GET  /health                      → healthHandler.Check (no auth)
GET  /auth/google                 → authHandler.RedirectToGoogle (no auth)
GET  /auth/google/callback        → authHandler.GoogleCallback (no auth)
GET  /auth/github                 → authHandler.RedirectToGitHub (no auth)
GET  /auth/github/callback        → authHandler.GitHubCallback (no auth)
POST /api/v1/auth/login           → authHandler.EmailPasswordLogin (no auth)
POST /api/v1/auth/refresh         → authHandler.RefreshToken (no auth)
POST /api/v1/auth/logout          → authHandler.Logout (auth required)
GET  /api/v1/user/me              → userHandler.GetMe (auth required)
GET  /api/v1/stream               → streamHandler.Stream (auth required)
```

### Acceptance Criteria

- [x] File `api/cmd/opus/main.go` exists with `main()` function calling root command.
- [x] File `api/cmd/opus/root.go` exists with Cobra root command.
- [x] Files for `start`, `stop`, `restart`, `status`, `logs` commands exist.
- [x] `start` command initializes all dependencies and starts the GoFiber server.
- [x] All routes are registered as specified.
- [x] Protected routes use Auth middleware.
- [x] CORS middleware is registered.
- [x] Recovery and logger middleware are registered.
- [x] Running `go build ./cmd/opus` produces a binary without errors.
- [x] Running `./bin/opus --help` lists all commands.

---

## P2-T13 — Configure `api/Taskfile.yml` (Final)

### What to Do

Finalize `api/Taskfile.yml` with all tasks including live reload (`air`) for development.

### Install `air` for Live Reload

```bash
go install github.com/air-verse/air@latest
```

### Final `api/Taskfile.yml`

```yaml
version: "3"

env:
  GOBIN: "{{.HOME}}/.local/bin"

tasks:
  setup:
    desc: Install Go module dependencies and tools
    cmds:
      - go mod download
      - go install github.com/air-verse/air@latest
      - go install go.uber.org/mock/mockgen@latest
      - go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

  dev:
    desc: Run API in development mode with live reload
    cmds:
      - air

  build:
    desc: Build production binary
    cmds:
      - go build -o bin/opus ./cmd/opus

  test:
    desc: Run unit tests
    cmds:
      - go test -v -race ./internal/...

  test:integration:
    desc: Run integration tests
    cmds:
      - go test -v -tags=integration -race ./internal/...

  test:all:
    desc: Run all tests
    deps: [test, test:integration]

  ent:generate:
    desc: Regenerate EntGo code from schema
    cmds:
      - go generate ./ent/...

  mock:generate:
    desc: Generate mocks for all repository interfaces
    cmds:
      - mockgen -source=internal/service/auth.go -destination=internal/mocks/mock_auth_repo.go -package=mocks
      - mockgen -source=internal/service/user.go -destination=internal/mocks/mock_user_repo.go -package=mocks

  migrate:
    desc: Run database migrations (auto-run on startup via GetDatabase)
    cmds:
      - go run ./cmd/opus migrate

  lint:
    desc: Run golangci-lint
    cmds:
      - golangci-lint run ./...
```

### Acceptance Criteria

- [x] File `api/Taskfile.yml` is updated with all tasks.
- [x] `task setup` installs `air`, `mockgen`, and `golangci-lint`.
- [x] `task dev` runs `air` for live reload.
- [x] `task mock:generate` generates mocks into `internal/mocks/`.
- [x] `task test` runs with `-race` flag.
- [x] `task test:integration` uses `-tags=integration`.

---

## P2-T14 — Write Unit Tests for Auth Service

### What to Do

Write unit tests for `internal/service/auth.go` using `uber/mock`. Generate mocks first with `task mock:generate`.

### File to Create

`api/internal/service/auth_test.go`

### Test Cases to Implement

| Test | Description |
|------|-------------|
| `TestAuthService_UpsertOAuthUser_NewUser` | User doesn't exist → creates new user |
| `TestAuthService_UpsertOAuthUser_ExistingUser` | User exists → returns existing user |
| `TestAuthService_IssueTokens_Success` | Issues valid JWT access token and refresh token |
| `TestAuthService_RefreshTokens_Success` | Valid refresh token → rotates tokens |
| `TestAuthService_RefreshTokens_RevokedToken` | Revoked token → revokes all user sessions |
| `TestAuthService_RefreshTokens_ExpiredToken` | Expired token → returns error |
| `TestAuthService_Logout_Success` | Valid token → revokes session |
| `TestAuthService_ValidateAccessToken_Valid` | Valid JWT → returns user ID |
| `TestAuthService_ValidateAccessToken_Invalid` | Invalid JWT → returns error |

### Constraints

- All repository calls must be mocked via `uber/mock`.
- No real database calls in unit tests.
- Use `t.Run` for subtests.
- Use `require` from `testify` for fatal assertions.

### Acceptance Criteria

- [x] File `api/internal/service/auth_test.go` exists.
- [x] All 9 test cases are implemented.
- [x] No real database calls — all repository methods are mocked.
- [x] Running `task test` passes all unit tests.
- [x] Test coverage for `auth.go` is ≥ 80%.

---

## P2-T15 — Write Integration Tests for Repository Layer

### What to Do

Write integration tests for `internal/repository/user.go` and `internal/repository/session.go` using a real SQLite in-memory database.

### Files to Create

- `api/internal/repository/user_integration_test.go`
- `api/internal/repository/session_integration_test.go`

### Build Tag

All integration test files must start with:

```go
//go:build integration
```

### Test Cases — User Repository

| Test | Description |
|------|-------------|
| `TestUserRepository_Create` | Creates user and retrieves by ID |
| `TestUserRepository_FindByEmail` | Finds user by email |
| `TestUserRepository_FindByProviderID` | Finds user by provider + provider ID |
| `TestUserRepository_FindByEmail_NotFound` | Returns `ErrUserNotFound` |
| `TestUserRepository_Update` | Updates user name and avatar |

### Test Cases — Session Repository

| Test | Description |
|------|-------------|
| `TestSessionRepository_Create` | Creates session |
| `TestSessionRepository_FindByTokenHash` | Finds session by hash |
| `TestSessionRepository_RevokeByID` | Revokes single session |
| `TestSessionRepository_RevokeAllByUserID` | Revokes all sessions for a user |

### In-Memory SQLite Setup

```go
func setupTestDB(t *testing.T) *ent.Client {
    client, err := ent.Open("sqlite3", "file::memory:?cache=shared&_fk=1")
    require.NoError(t, err)
    require.NoError(t, client.Schema.Create(context.Background()))
    t.Cleanup(func() { client.Close() })
    return client
}
```

### Acceptance Criteria

- [x] File `api/internal/repository/user_integration_test.go` exists with `//go:build integration` tag.
- [x] File `api/internal/repository/session_integration_test.go` exists with `//go:build integration` tag.
- [x] All user repository test cases are implemented.
- [x] All session repository test cases are implemented.
- [x] Tests use SQLite in-memory database — no mocks.
- [x] Running `task test:integration` passes all integration tests.

---

## Phase 2 Completion Checklist

Before proceeding to Phase 3, verify:

- [x] All 15 tasks (P2-T1 through P2-T15) are marked complete.
- [x] Running `go build ./cmd/opus` produces a binary without errors.
- [x] Running `./bin/opus --help` lists all Cobra commands.
- [x] Running `task test` passes all unit tests.
- [x] Running `task test:integration` passes all integration tests.
- [x] Running `task lint` passes without errors.
- [x] `GET /health` returns `{"success": true, "data": {"status": "ok", ...}}`.
- [x] Email/Password login returns 403 when `OPUS_SERVER_ENV=production`.
- [x] No raw tokens or secrets appear in any log output.
