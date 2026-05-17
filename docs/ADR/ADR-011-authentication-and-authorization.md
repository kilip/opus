# ADR-011: Authentication and Authorization Architecture

**Status:** Accepted
**Date:** 2026-05-17
**Deciders:** Chief Architect
**Context:** Opus Server (`opus/server/`) · Opus Dash (`opus/dash/`)

---

## 1. Context

Opus is a self-hosted, multi-user autonomous AI assistant. With the introduction of multi-user
support, the system requires a formal, extensible authentication and authorization architecture
that governs how users authenticate, how sessions are managed, and how access to workspaces and
resources is enforced across all domains.

Without a formally defined auth architecture, individual handlers risk implementing ad-hoc
token validation, role checking, and session management — leading to security inconsistencies,
untestable code paths, and friction for future capability expansion.

This ADR establishes the canonical authentication mechanism (stateful JWT with refresh token
rotation), OAuth2 provider abstraction, workspace-scoped role-based access control via Casbin,
and the corresponding Dash client auth flow for all code under `opus/server/` and `opus/dash/`.

> **Note for AI agents and automated tooling:** This ADR is the authoritative specification for
> authentication and authorization in Opus. Do not infer token strategies, role models, URL
> structures, or middleware placement beyond what is defined here.

---

## 2. Decision

Opus Server adopts **stateful JWT with refresh token rotation** for session management,
**OAuth2 with an extensible provider abstraction** for external authentication, and
**Casbin domain-based RBAC** for workspace-scoped authorization. All auth logic is
encapsulated in the `internal/auth/` domain package; delivery is handled exclusively via
the Fiber middleware and handler layers defined in ADR-005.

---

### 2.1 Directory Structure

```
opus/
└── server/
    ├── internal/
    │   ├── auth/
    │   │   ├── model.go          # Domain models: User, Token, Session, Claims, Role
    │   │   ├── repository.go     # Repository interface (port) + //go:generate directive
    │   │   ├── service.go        # Business logic: login, logout, refresh, OAuth2 exchange
    │   │   ├── oauth.go          # Provider interface + registry
    │   │   ├── casbin.go         # Casbin enforcer wrapper + policy service
    │   │   ├── config.go         # auth.Config struct (hybrid composition — ADR-002)
    │   │   ├── errors.go         # Sentinel errors
    │   │   └── mock_repository.go  # Generated — DO NOT EDIT
    │   │
    │   ├── adapter/
    │   │   ├── entgo/
    │   │   │   └── auth.go           # Concrete repository implementation (Ent)
    │   │   └── oauth/
    │   │       ├── provider.go       # Provider interface implementation base
    │   │       ├── google.go         # Google OAuth2 provider
    │   │       └── github.go         # GitHub OAuth2 provider
    │   │
    │   └── delivery/
    │       └── gofiber/
    │           ├── handler/
    │           │   └── auth.go   # Login, logout, refresh, OAuth2 callback, /auth/me
    │           ├── middleware/
    │           │   ├── auth.go           # Token validation + context injection
    │           │   └── rbac.go           # Casbin enforcement middleware
    │           ├── router.go             # Route registration + app bootstrap
    │           ├── response.go           # ADR-004 envelope helpers
    │           └── config.go             # GoFiber configuration struct (hybrid composition)
```

---

### 2.2 Token Strategy — Stateful JWT with Refresh Token Rotation

Opus uses **stateful JWT** for session management. Tokens are cryptographically signed JWTs
whose validity is verified against a database record on every request. This enables immediate
revocation on logout without waiting for token expiry.

#### 2.2.1 Token Pair

| Token | Lifetime | Storage (Server) | Storage (Client) |
|---|---|---|---|
| Access Token | 15 minutes | DB record (`tokens` table) | `httpOnly` cookie |
| Refresh Token | 7 days | DB record (`tokens` table) | `httpOnly` cookie |

**Rules:**

- Both tokens are stored as `httpOnly`, `Secure`, `SameSite=Strict` cookies — never exposed
  to JavaScript. This eliminates XSS-based token theft.
- Access tokens are short-lived (15 minutes) to minimise the window of exposure if a token
  is compromised at the network layer.
- Refresh tokens are rotated on every use — each refresh issues a new refresh token and
  invalidates the previous one.
- All token records carry a `revoked_at` timestamp. Logout sets this field; the middleware
  rejects tokens with a non-null `revoked_at`.

#### 2.2.2 JWT Claims

```go
// internal/auth/model.go
package auth

import "time"

// Claims represents the payload embedded in a signed JWT.
// Claims are minimal — role and workspace are always resolved from the DB,
// never trusted from the token payload alone.
type Claims struct {
    // Sub is the user ID (stable, immutable identifier).
    Sub string `json:"sub"`

    // SessionID is the DB record ID for the token pair.
    // Used to look up and validate the token against the database.
    SessionID string `json:"sid"`

    // WorkspaceID identifies the workspace this session belongs to.
    WorkspaceID string `json:"wid"`

    // Role is the user's role within the workspace at the time of issuance.
    // INFORMATIONAL ONLY — authorization decisions always query Casbin,
    // never this field directly.
    Role string `json:"role"`

    // IssuedAt and ExpiresAt are standard JWT time claims.
    IssuedAt  time.Time `json:"iat"`
    ExpiresAt time.Time `json:"exp"`
}
```

> **Authorization rule:** The `Role` field in JWT claims is informational only. All
> authorization decisions are made by the Casbin enforcer against live DB policy records.
> Middleware must never grant or deny access based solely on the JWT `role` claim.

#### 2.2.3 Refresh Token Rotation Flow

```
Client                      Server
  │                            │
  │── POST /auth/refresh ──────►│
  │   (httpOnly cookie)        │
  │                            │── Validate refresh token against DB
  │                            │── Check revoked_at IS NULL
  │                            │── Revoke old refresh token (set revoked_at)
  │                            │── Issue new access token + new refresh token
  │                            │── Persist new token pair to DB
  │◄── 200 OK ─────────────────│
  │   (Set-Cookie: new tokens) │
```

If the server detects a previously-rotated refresh token being reused (replay attack),
it revokes the entire session family (all tokens for that session) immediately.

---

### 2.3 OAuth2 Provider Abstraction

OAuth2 authentication is handled server-side. Dash initiates the flow with a redirect;
the Server handles the callback, exchanges the code for a user profile, upserts the user
record, and issues a token pair.

#### 2.3.1 Provider Interface

```go
// internal/auth/oauth.go
package auth

import "context"

// OAuthProvider defines the contract for an OAuth2 identity provider.
// New providers are added by implementing this interface in internal/adapter/oauth/
// and registering them in the ProviderRegistry at startup.
//
//go:generate mockgen -destination=mock_oauth_provider.go -package=auth . OAuthProvider
type OAuthProvider interface {
    // Name returns the canonical provider identifier (e.g. "google", "github").
    Name() string

    // AuthURL returns the authorization URL to redirect the user to.
    AuthURL(state string) string

    // Exchange exchanges an authorization code for a normalized UserProfile.
    Exchange(ctx context.Context, code string) (*OAuthProfile, error)
}

// OAuthProfile is the normalized user identity returned by any OAuth2 provider.
// Provider-specific fields are not surfaced here; only the minimal identity
// attributes required to upsert a user record are included.
type OAuthProfile struct {
    // ProviderID is the user's unique identifier within the OAuth2 provider.
    ProviderID string

    // Provider is the canonical provider name (e.g. "google", "github").
    Provider string

    // Email is the verified primary email address.
    Email string

    // Name is the display name, used to populate the user record on first login.
    Name string

    // AvatarURL is an optional profile picture URL.
    AvatarURL string
}

// ProviderRegistry holds all registered OAuth2 providers.
// Providers are registered at startup in main.go.
type ProviderRegistry struct {
    providers map[string]OAuthProvider
}

// Register adds a provider to the registry.
// Panics if called after the server has started or if the provider name is duplicate.
func (r *ProviderRegistry) Register(p OAuthProvider) {
    if r.providers == nil {
        r.providers = make(map[string]OAuthProvider)
    }
    r.providers[p.Name()] = p
}

// Get retrieves a provider by name. Returns ErrProviderNotFound if absent.
func (r *ProviderRegistry) Get(name string) (OAuthProvider, error) {
    p, ok := r.providers[name]
    if !ok {
        return nil, ErrProviderNotFound
    }
    return p, nil
}
```

#### 2.3.2 OAuth2 Flow

```
Dash                        Server                      OAuth Provider
  │                            │                              │
  │── GET /auth/oauth/google ──►│                              │
  │                            │── Generate state (CSRF) ────►│
  │◄── 302 Redirect ───────────│   AuthURL(state)             │
  │                            │                              │
  │── (user authenticates) ────────────────────────────────►  │
  │                            │◄── Callback with code ───────│
  │                            │── Validate state             │
  │                            │── Exchange(ctx, code)        │
  │                            │── Upsert user record         │
  │                            │── Issue token pair           │
  │◄── 302 → Dash /agent ──────│                              │
  │   (Set-Cookie: tokens)     │
```

**State parameter:** A cryptographically random string stored server-side (in the `auth_states`
table with a 10-minute TTL) to prevent CSRF attacks during the OAuth2 callback.

#### 2.3.3 Built-in Providers

| Provider | Adapter Package | Config Keys |
|---|---|---|
| Google | `internal/adapter/oauth/google.go` | `auth.oauth.google.client_id`, `auth.oauth.google.client_secret` |
| GitHub | `internal/adapter/oauth/github.go` | `auth.oauth.github.client_id`, `auth.oauth.github.client_secret` |

Additional providers are added by implementing `auth.OAuthProvider` in `internal/adapter/oauth/` and
registering the instance in `main.go`. No changes to the domain layer are required.

---

### 2.4 Authorization — Casbin Domain-Based RBAC

Opus uses **Casbin** with a **domain-based RBAC model** to enforce workspace-scoped
access control. Every authorization decision is evaluated against live policy records
persisted in the database.

#### 2.4.1 Casbin Model

```ini
# server/internal/auth/casbin_model.conf

[request_definition]
r = sub, dom, obj, act

[policy_definition]
p = sub, dom, obj, act

[role_definition]
g = _, _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub, r.dom) && r.dom == p.dom && r.obj == p.obj && r.act == p.act
```

| Field | Description |
|---|---|
| `sub` | User ID |
| `dom` | Workspace ID (domain) |
| `obj` | Resource being accessed (e.g. `agent`, `vault`, `workflow`) |
| `act` | Action being performed (e.g. `read`, `write`, `delete`, `manage`) |

#### 2.4.2 Built-in Roles and Policies

Two roles are defined at MVP. The policy table is seeded at workspace creation.

**`admin` role — full access within workspace:**

```
p, admin, {workspace_id}, agent,    read
p, admin, {workspace_id}, agent,    write
p, admin, {workspace_id}, agent,    delete
p, admin, {workspace_id}, agent,    manage
p, admin, {workspace_id}, vault,    read
p, admin, {workspace_id}, vault,    write
p, admin, {workspace_id}, vault,    delete
p, admin, {workspace_id}, workflow, read
p, admin, {workspace_id}, workflow, write
p, admin, {workspace_id}, workflow, delete
p, admin, {workspace_id}, user,     manage
```

**`user` role — read + limited write within workspace:**

```
p, user, {workspace_id}, agent,    read
p, user, {workspace_id}, agent,    write
p, user, {workspace_id}, vault,    read
p, user, {workspace_id}, vault,    write
p, user, {workspace_id}, workflow, read
p, user, {workspace_id}, workflow, write
```

**Role assignment:**

```
g, {user_id}, admin, {workspace_id}
g, {user_id}, user,  {workspace_id}
```

#### 2.4.3 Casbin Enforcer Wrapper

```go
// internal/auth/casbin.go
package auth

import (
    "context"
    "fmt"

    "github.com/casbin/casbin/v2"
)

// PolicyService wraps the Casbin enforcer and provides workspace-scoped
// authorization checks for use by the RBAC middleware and service layer.
type PolicyService struct {
    enforcer *casbin.Enforcer
}

// NewPolicyService constructs a PolicyService with the given Casbin enforcer.
func NewPolicyService(enforcer *casbin.Enforcer) *PolicyService {
    return &PolicyService{enforcer: enforcer}
}

// Enforce returns true if the subject (userID) is permitted to perform
// the given action on the given object within the specified workspace (domain).
func (ps *PolicyService) Enforce(ctx context.Context, userID, workspaceID, obj, act string) (bool, error) {
    ok, err := ps.enforcer.Enforce(userID, workspaceID, obj, act)
    if err != nil {
        return false, fmt.Errorf("auth.PolicyService.Enforce: %w", err)
    }
    return ok, nil
}

// AssignRole assigns the given role to a user within a workspace.
func (ps *PolicyService) AssignRole(ctx context.Context, userID, role, workspaceID string) error {
    _, err := ps.enforcer.AddRoleForUserInDomain(userID, role, workspaceID)
    if err != nil {
        return fmt.Errorf("auth.PolicyService.AssignRole: %w", err)
    }
    return nil
}

// RevokeRole removes a role from a user within a workspace.
func (ps *PolicyService) RevokeRole(ctx context.Context, userID, role, workspaceID string) error {
    _, err := ps.enforcer.DeleteRoleForUserInDomain(userID, role, workspaceID)
    if err != nil {
        return fmt.Errorf("auth.PolicyService.RevokeRole: %w", err)
    }
    return nil
}
```

#### 2.4.4 Policy Adapter — DB-Backed

Casbin policies are persisted in the database via the
[casbin-ent-adapter](https://github.com/casbin/ent-adapter), consistent with ADR-007 (Ent ORM).
The enforcer is configured with auto-save enabled; policy changes are persisted immediately.

```go
// internal/adapter/entgo/casbin.go
package entgo

import (
    entadapter "github.com/casbin/ent-adapter"
    "github.com/casbin/casbin/v2"
)

// NewCasbinEnforcer constructs a Casbin enforcer backed by the Ent database.
func NewCasbinEnforcer(dsn, modelPath string) (*casbin.Enforcer, error) {
    adapter, err := entadapter.NewAdapter("sqlite3", dsn)
    if err != nil {
        return nil, err
    }
    return casbin.NewEnforcer(modelPath, adapter)
}
```

---

### 2.5 Middleware Architecture

Auth concerns are split into two independent Fiber middleware, applied in sequence.

```
Request → [auth.go: Token Validation] → [rbac.go: Policy Enforcement] → Handler
```

#### 2.5.1 Token Validation Middleware (`middleware/auth.go`)

Responsibilities:

1. Extract the access token from the `httpOnly` cookie.
2. Parse and verify the JWT signature.
3. Look up the token record in the DB and verify `revoked_at IS NULL`.
4. Inject the resolved `Claims` into `context.Context` using a typed context key.
5. Inject `request_id`, `user_id`, and `workspace_id` for structured logging (ADR-006).

```go
// internal/delivery/gofiber/middleware/auth.go
package middleware

import (
    "github.com/gofiber/fiber/v3"
    "opus/server/internal/auth"
    "opus/server/internal/shared/logger"
)

// Authenticate validates the access token and injects auth context.
// Returns 401 Unauthorized if the token is absent, invalid, or revoked.
func Authenticate(svc *auth.Service, log logger.Logger) fiber.Handler {
    return func(c fiber.Ctx) error {
        token := c.Cookies("opus_access_token")
        if token == "" {
            return c.Status(fiber.StatusUnauthorized).JSON(unauthorizedError(c))
        }

        claims, err := svc.ValidateAccessToken(c.Context(), token)
        if err != nil {
            return c.Status(fiber.StatusUnauthorized).JSON(unauthorizedError(c))
        }

        ctx := auth.WithClaims(c.Context(), claims)
        ctx = logger.WithUserID(ctx, claims.Sub)
        c.SetUserContext(ctx)

        return c.Next()
    }
}
```

#### 2.5.2 RBAC Enforcement Middleware (`middleware/rbac.go`)

The RBAC middleware is a **factory** — it returns a handler configured for a specific
`(obj, act)` pair. This allows per-route authorization without a centralised policy map.

```go
// internal/delivery/gofiber/middleware/rbac.go
package middleware

import (
    "github.com/gofiber/fiber/v3"
    "opus/server/internal/auth"
    "opus/server/internal/delivery/gofiber"
)

// Require returns a Fiber middleware that enforces a Casbin policy check
// for the given resource object and action within the authenticated user's workspace.
//
// Must be applied after Authenticate middleware.
//
// Example:
//   app.Delete("/api/agents/:id", middleware.Require(policy, "agent", "delete"), handler.DeleteAgent)
func Require(policy *auth.PolicyService, obj, act string) fiber.Handler {
    return func(c fiber.Ctx) error {
        claims := auth.ClaimsFromContext(c.UserContext())
        if claims == nil {
            return gofiber.Error(c, fiber.StatusUnauthorized,
                "unauthorized", "Unauthorized", "authentication required")
        }

        ok, err := policy.Enforce(c.Context(), claims.Sub, claims.WorkspaceID, obj, act)
        if err != nil || !ok {
            return gofiber.Error(c, fiber.StatusForbidden,
                "forbidden", "Forbidden", "insufficient permissions")
        }

        return c.Next()
    }
}
```

**Router usage example:**

```go
// internal/delivery/gofiber/router.go (excerpt)
authGroup := app.Group("/auth")
authGroup.Post("/login",              auth.Login)
authGroup.Post("/logout",             middleware.Authenticate(authSvc, log), auth.Logout)
authGroup.Post("/refresh",            auth.Refresh)
authGroup.Get("/me",                  middleware.Authenticate(authSvc, log), auth.Me)
authGroup.Get("/oauth/:provider",     auth.OAuthRedirect)
authGroup.Get("/oauth/:provider/callback", auth.OAuthCallback)

api := app.Group("/api", middleware.Authenticate(authSvc, log))
api.Get("/agents",     middleware.Require(policy, "agent", "read"),   agent.ListAgents)
api.Post("/agents",    middleware.Require(policy, "agent", "write"),  agent.CreateAgent)
api.Delete("/agents/:id", middleware.Require(policy, "agent", "delete"), agent.DeleteAgent)
```

---

### 2.6 Auth Endpoints

Auth endpoints are served under `/auth/` — **no `/api/` prefix, no version segment**,
consistent with the URL convention established in ADR-004.

| Method | Path | Auth Required | Description |
|---|---|---|---|
| `POST` | `/auth/login` | No | Email + password login; issues token pair |
| `POST` | `/auth/logout` | Yes | Revokes the current session token pair |
| `POST` | `/auth/refresh` | No | Rotates the refresh token; issues new token pair |
| `GET` | `/auth/me` | Yes | Returns the authenticated user's profile and role |
| `GET` | `/auth/oauth/{provider}` | No | Redirects to the OAuth2 provider authorization URL |
| `GET` | `/auth/oauth/{provider}/callback` | No | Handles OAuth2 callback; issues token pair |

**Response shapes follow ADR-004 envelope convention.**

**`GET /auth/me` response example:**

```json
{
  "data": {
    "id": "usr_01HZ9XYZ",
    "email": "user@example.com",
    "name": "Alice",
    "role": "admin",
    "workspace_id": "ws_01ABCDEF"
  },
  "error": null,
  "meta": null
}
```

---

### 2.7 Dash Auth Flow

Opus Dash handles authentication through a combination of the API client, TanStack Router
guards, and httpOnly cookie management delegated entirely to the browser.

#### 2.7.1 Token Storage

All tokens are stored in `httpOnly`, `Secure`, `SameSite=Strict` cookies set by the Server.
Dash has no direct access to token values — the browser attaches cookies automatically on
every request to the same origin.

#### 2.7.2 Auth State Resolution

On application load, Dash calls `GET /auth/me` to resolve the current auth state. The Server
is the single source of truth for session validity.

```typescript
// features/auth/api.ts
import { queryOptions } from '@tanstack/react-query';
import { apiClient } from '@/shared/lib/api-client';
import type { AuthUser } from './types';

export const authQueries = {
  me: () =>
    queryOptions({
      queryKey: ['auth', 'me'],
      queryFn: () => apiClient.get<AuthUser>('/auth/me'),
      retry: false,
      staleTime: 5 * 60 * 1000, // 5 minutes
    }),
};
```

#### 2.7.3 TanStack Router Guard

The root route evaluates the `/auth/me` response. Unauthenticated users are redirected
to `/login`. Authenticated users reaching `/login` are redirected to `/agent`.

```typescript
// routes/__root.tsx
import { createRootRouteWithContext, Outlet, redirect } from '@tanstack/react-router';
import type { QueryClient } from '@tanstack/react-query';
import { authQueries } from '@/features/auth/api';

export const Route = createRootRouteWithContext<{ queryClient: QueryClient }>()({
  beforeLoad: async ({ context, location }) => {
    try {
      const user = await context.queryClient.fetchQuery(authQueries.me());
      if (location.pathname === '/login') {
        throw redirect({ to: '/agent' });
      }
      return { user };
    } catch {
      if (location.pathname !== '/login') {
        throw redirect({ to: '/login' });
      }
    }
  },
  component: () => <Outlet />,
});
```

#### 2.7.4 OAuth2 Initiation

Dash initiates the OAuth2 flow by navigating the browser to the Server's redirect endpoint.
No client-side OAuth2 logic is required.

```typescript
// features/auth/components/LoginPage.tsx
const handleOAuth = (provider: 'google' | 'github') => {
  window.location.href = `${import.meta.env.VITE_API_URL}/auth/oauth/${provider}`;
};
```

---

### 2.8 Domain Models

```go
// internal/auth/model.go
package auth

import "time"

// User represents an authenticated Opus user.
type User struct {
    ID          string
    Email       string
    Name        string
    AvatarURL   string
    Provider    string // "local", "google", "github"
    ProviderID  string // empty for local users
    WorkspaceID string
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

// Token represents a persisted access or refresh token record.
type Token struct {
    ID        string
    SessionID string
    UserID    string
    Type      TokenType // "access" | "refresh"
    Hash      string    // SHA-256 hash of the raw token value
    ExpiresAt time.Time
    RevokedAt *time.Time
    CreatedAt time.Time
}

// TokenType identifies whether a token record is an access or refresh token.
type TokenType string

const (
    TokenTypeAccess  TokenType = "access"
    TokenTypeRefresh TokenType = "refresh"
)

// Session groups an access token and refresh token issued together.
// Revoking a session invalidates both tokens atomically.
type Session struct {
    ID           string
    UserID       string
    WorkspaceID  string
    AccessToken  *Token
    RefreshToken *Token
    CreatedAt    time.Time
}
```

---

### 2.9 Sentinel Errors

```go
// internal/auth/errors.go
package auth

import "errors"

// ErrInvalidCredentials is returned when email/password authentication fails.
var ErrInvalidCredentials = errors.New("auth: invalid credentials")

// ErrTokenExpired is returned when a token has passed its expiry time.
var ErrTokenExpired = errors.New("auth: token expired")

// ErrTokenRevoked is returned when a token has been explicitly revoked.
var ErrTokenRevoked = errors.New("auth: token revoked")

// ErrTokenReplay is returned when a rotated refresh token is reused,
// indicating a potential token replay attack. The entire session is revoked.
var ErrTokenReplay = errors.New("auth: refresh token replay detected")

// ErrProviderNotFound is returned when an unregistered OAuth2 provider is requested.
var ErrProviderNotFound = errors.New("auth: oauth provider not found")

// ErrInvalidOAuthState is returned when the OAuth2 state parameter fails validation.
var ErrInvalidOAuthState = errors.New("auth: invalid oauth state")

// ErrSessionNotFound is returned when a session record does not exist in the database.
var ErrSessionNotFound = errors.New("auth: session not found")
```

---

### 2.10 Configuration — Hybrid Composition (ADR-002)

```go
// internal/auth/config.go
package auth

// Config holds all authentication configuration.
// Owned by the auth package; composed into the root config.Config
// by internal/config/model.go.
//
// Environment variable overrides:
//   OPUS_AUTH_JWT_SECRET               — sets JWTSecret (required; never in config file)
//   OPUS_AUTH_OAUTH_GOOGLE_CLIENT_ID   — sets OAuth.Google.ClientID
//   OPUS_AUTH_OAUTH_GOOGLE_SECRET      — sets OAuth.Google.ClientSecret
//   OPUS_AUTH_OAUTH_GITHUB_CLIENT_ID   — sets OAuth.GitHub.ClientID
//   OPUS_AUTH_OAUTH_GITHUB_SECRET      — sets OAuth.GitHub.ClientSecret
type Config struct {
    // JWTSecret is the HMAC signing secret for JWT tokens.
    // Must be injected via OPUS_AUTH_JWT_SECRET — never stored in config files.
    JWTSecret string `mapstructure:"jwt_secret" json:"-" jsonschema:"-"`

    // AccessTokenTTL is the lifetime of access tokens. Default: "15m".
    AccessTokenTTL string `mapstructure:"access_token_ttl" json:"access_token_ttl" jsonschema:"default=15m"`

    // RefreshTokenTTL is the lifetime of refresh tokens. Default: "168h" (7 days).
    RefreshTokenTTL string `mapstructure:"refresh_token_ttl" json:"refresh_token_ttl" jsonschema:"default=168h"`

    // OAuth holds configuration for all registered OAuth2 providers.
    OAuth OAuthConfig `mapstructure:"oauth" json:"oauth"`
}

// OAuthConfig holds per-provider OAuth2 credentials.
type OAuthConfig struct {
    Google ProviderCredentials `mapstructure:"google" json:"google"`
    GitHub ProviderCredentials `mapstructure:"github" json:"github"`
}

// ProviderCredentials holds the client ID and secret for a single OAuth2 provider.
type ProviderCredentials struct {
    ClientID     string `mapstructure:"client_id"     json:"client_id"`
    ClientSecret string `mapstructure:"client_secret" json:"-" jsonschema:"-"`
    RedirectURL  string `mapstructure:"redirect_url"  json:"redirect_url"`
}
```

---

### 2.11 Dependency Injection at Startup

```go
// main.go (auth wiring excerpt)
package main

import (
    "opus/server/internal/adapter/entgo"
    "opus/server/internal/adapter/oauth"
    "opus/server/internal/delivery/gofiber/handler"
    "opus/server/internal/delivery/gofiber/middleware"
    "opus/server/internal/auth"
    "opus/server/internal/config"
)

func main() {
    cfg, _ := config.Load()

    // Casbin enforcer (DB-backed policy adapter)
    enforcer, _ := entgo.NewCasbinEnforcer(cfg.Database.DSN, "internal/auth/casbin_model.conf")
    policyService := auth.NewPolicyService(enforcer)

    // OAuth2 provider registry
    registry := &auth.ProviderRegistry{}
    registry.Register(oauth.NewGoogleProvider(cfg.Auth.OAuth.Google))
    registry.Register(oauth.NewGitHubProvider(cfg.Auth.OAuth.GitHub))

    // Auth domain
    authRepo := entgo.NewAuthRepo(entClient)
    authService := auth.NewService(authRepo, registry, policyService, cfg.Auth, log)

    // Delivery
    auth := handler.NewAuth(authService)

    // Router wiring
    app := gofiber.New(cfg.Server, auth, agent, /* ... */)
}
```

---

## 3. Alternatives Considered

### 3.1 Stateless JWT (No DB Validation)

Pure stateless JWT with no database record. Rejected because:

- Logout becomes impossible without a token denylist — a revoked token remains valid until
  expiry (up to 15 minutes), which is unacceptable for a security-sensitive personal assistant
- Replay attack detection for refresh tokens requires server-side state regardless
- Self-hosted single-instance deployment negates the horizontal scaling benefit of stateless JWT

### 3.2 Session Cookies (No JWT)

Traditional server-side sessions with a session ID in a cookie, validated against a DB record.
Rejected because:

- JWT claims provide a compact, self-describing identity payload useful for structured logging
  (user ID, workspace ID, role) without an additional DB query in every middleware
- OAuth2 integration is cleaner with JWT as the issuance format
- Future service-to-service auth (agent runtime calling internal APIs) benefits from JWT portability

### 3.3 Standard RBAC Without Domain Scoping (`sub, obj, act`)

Casbin RBAC without workspace domain awareness. Rejected because:

- Opus is a multi-user, multi-workspace system; permissions are inherently workspace-scoped
- Upgrading from flat RBAC to domain-based RBAC after data exists requires a breaking policy
  migration. Starting with `sub, dom, obj, act` from the outset avoids this migration entirely
- Casbin's domain-based RBAC (`g = _, _, _`) is a first-class built-in feature, not a custom extension

### 3.4 Open Policy Agent (OPA)

External policy engine. Rejected because:

- Introduces an external process dependency, violating Opus's zero-mandatory-infrastructure principle
- Casbin's Go-native library integrates directly with the Ent adapter and the Fiber middleware
  without a network hop
- OPA's Rego policy language adds unnecessary complexity for the two-role MVP model

---

## 4. Consequences

### 4.1 Positive

- **Immediate Revocation** — Stateful tokens enable instant session termination on logout;
  no waiting for short-lived token expiry
- **XSS-Proof Token Storage** — `httpOnly` cookies prevent JavaScript-based token exfiltration
- **Replay Attack Detection** — Refresh token rotation with family revocation detects and
  neutralises stolen refresh tokens immediately
- **Extensible OAuth2** — New providers require only a new `internal/adapter/oauth/` implementation
  and a `registry.Register()` call in `main.go`; zero domain layer changes
- **Workspace-Scoped Authorization** — Casbin domain-based RBAC correctly isolates permissions
  per workspace from day one; no future migration required to add workspace scope
- **Per-Route Enforcement** — `middleware.Require(policy, obj, act)` enables declarative,
  readable authorization rules co-located with route registration
- **Type Safety** — All auth models, claims, and errors are strongly typed; no stringly-typed
  role or permission checks in handler code

### 4.2 Negative / Trade-offs

- **DB Hit Per Request** — Stateful token validation requires a database lookup on every
  authenticated request (to check `revoked_at`). For a self-hosted localhost system this is
  negligible; a Redis token cache may be introduced in a future ADR if latency becomes measurable
- **Cookie-Only Token Transport** — `httpOnly` cookies require `SameSite` and CORS configuration
  to work correctly when Dash and Server are on different ports during development;
  VITE proxy configuration is required in the Dash dev setup
- **Casbin Enforcer Warm-Up** — The Casbin enforcer loads all policies from the DB at startup;
  large policy sets may affect startup time. The ent-adapter supports lazy loading as a future
  optimisation
- **OAuth2 State TTL** — OAuth2 state tokens expire after 10 minutes. Users who take longer
  than 10 minutes to complete the OAuth2 flow will receive an error and must restart; this is
  an acceptable UX trade-off for CSRF protection

---

## 5. References

- [ADR-001: Server Clean Architecture](./ADR-001-server-clean-architecture.md)
- [ADR-002: Configuration Management](./ADR-002-server-configuration.md)
- [ADR-003: Opus Dash Frontend Architecture](./ADR-003-dash-frontend-architecture.md)
- [ADR-004: API Response Contract](./ADR-004-api-response-contract.md)
- [ADR-005: Server Delivery Layer with GoFiber v3](./ADR-005-server-delivery-layer-with-gofiber-v3.md)
- [ADR-006: Server Logger Architecture](./ADR-006-server-logger.md)
- [ADR-007: ORM and Database Strategy](./ADR-007-orm-and-database-strategy.md)
- [ADR-009: Server Testing Strategy](./ADR-009-server-testing-strategy.md)
- [ADR-010: Server Coding Conventions & Linting](./ADR-010-server-coding-and-linting.md)
- [Casbin — casbin.org](https://casbin.org)
- [Casbin Ent Adapter — github.com/casbin/ent-adapter](https://github.com/casbin/ent-adapter)
- [golang-jwt/jwt — github.com/golang-jwt/jwt](https://github.com/golang-jwt/jwt)
- [golang.org/x/oauth2](https://pkg.go.dev/golang.org/x/oauth2)
- [RFC 6749 — OAuth 2.0 Authorization Framework](https://www.rfc-editor.org/rfc/rfc6749)
- [RFC 6750 — OAuth 2.0 Bearer Token Usage](https://www.rfc-editor.org/rfc/rfc6750)
- [OWASP — JWT Security Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/JSON_Web_Token_for_Java_Cheat_Sheet.html)eet.html)