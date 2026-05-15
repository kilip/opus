# Conventions

**Product:** Opus  
**Version:** 1.0.1  
**Status:** Progress  
**Last Updated:** 2026-05-15  
**Authors:** Product & Architecture Team

---

## Purpose

This document defines the coding conventions, naming rules, and structural standards for the Opus project. Provide this file to your AI agent at the start of every session to ensure consistency across all phases of implementation.

---

## 1. General Principles

- Follow **Clean Architecture** strictly: dependencies flow inward only (`handler → service → repository → model`).
- Prefer **explicit over implicit**: no magic, no hidden side effects.
- Every file has **one responsibility**.
- No code is written without a corresponding test (unit or integration).
- Never commit secrets, `.env` files, or generated `ent/` code overrides.

---

## 2. Repository Structure

```
opus/
├── api/                  # Go backend (GoFiber v3 + EntGo + Viper)
├── dash/            # Next.js 16 frontend (PWA)
├── docs/                 # Project documentation
├── .github/workflows/    # CI/CD pipelines
├── docker-compose.yml
├── docker-compose.dev.yml
├── Taskfile.yml          # Root orchestrator
├── .env.example
└── README.md
```

> **Rule:** Never place business logic at the root level. Root only contains orchestration and configuration files.

---

## 3. Go Backend (`api/`)

### 3.1 Language & Toolchain

| Item | Standard |
|------|----------|
| Go version | 1.23+ |
| Module name | `github.com/kilip/opus/api` |
| Linter | `golangci-lint` |
| Task runner | Taskfile v3 |

### 3.2 Naming Conventions

| Element | Convention | Example |
|---------|-----------|---------|
| Packages | lowercase, singular | `model`, `service`, `repository`, `handler` |
| Files | lowercase, singular, snake_case | `user.go`, `auth.go`, `session.go` |
| Structs | PascalCase, singular | `User`, `Session`, `AuthService` |
| Interfaces | PascalCase, descriptive | `UserRepository`, `AuthService` |
| Functions | PascalCase (exported), camelCase (unexported) | `FindByEmail`, `hashToken` |
| Constants | PascalCase or SCREAMING_SNAKE_CASE | `DefaultPort`, `JWT_ALGORITHM` |
| Error variables | `Err` prefix | `ErrUserNotFound`, `ErrInvalidToken` |

### 3.3 Package Structure

```
api/internal/
├── model/          # Pure domain structs — no imports from other internal packages
├── service/        # Business logic — imports model only; defines repository interfaces
├── repository/     # Data access — imports model and ent; implements service interfaces
├── handler/        # HTTP layer — imports service only; no direct repository access
├── middleware/     # Cross-cutting concerns — imports config and model
└── config/         # Singleton accessors: GetConfig(), GetLogger(), GetDatabase()
```

> **Rule:** `handler` must never import `repository` directly. Always go through `service`.

### 3.4 Interface Definition Rule

Interfaces are defined in the **consumer** layer (`service/`), not the implementor layer (`repository/`).

```go
// CORRECT — interface defined in service/
// internal/service/auth.go
type UserRepository interface {
    FindByEmail(ctx context.Context, email string) (*model.User, error)
    Create(ctx context.Context, user *model.User) (*model.User, error)
}

// INCORRECT — do not define interfaces in repository/
```

### 3.5 Error Handling

- Always return errors; never `panic` in production code.
- Use `fmt.Errorf("context: %w", err)` for error wrapping.
- Define sentinel errors in `model/` for domain-level errors.

```go
// model/errors.go
var (
    ErrUserNotFound    = errors.New("user not found")
    ErrInvalidToken    = errors.New("invalid token")
    ErrTokenExpired    = errors.New("token expired")
)
```

### 3.6 HTTP Response Envelope

All API responses must use the standard envelope:

```go
// Success
{
  "success": true,
  "data": { },
  "error": null
}

// Error
{
  "success": false,
  "data": null,
  "error": {
    "code": "UNAUTHORIZED",
    "message": "Access token is expired or invalid."
  }
}
```

### 3.7 Configuration Access

Always use singleton accessors. Never instantiate config, logger, or database client directly.

```go
// CORRECT
cfg := config.GetConfig()
log := config.GetLogger()
db  := config.GetDatabase()

// INCORRECT — do not use viper.Get() or slog.New() directly in handlers/services
```

### 3.8 Logging

Use `slog` with structured fields. Never use `fmt.Println` or `log.Printf` in production code.

```go
// CORRECT
logger.Info("user authenticated", "user_id", user.ID, "provider", "google")

// INCORRECT
fmt.Println("user authenticated:", user.ID)
```

### 3.9 Testing Conventions

| Test Type | Location | Tool | Scope |
|-----------|----------|------|-------|
| Unit | `internal/*/<file>_test.go` | `testing` + `uber/mock` | `service/` layer only |
| Integration | `internal/repository/<file>_integration_test.go` | `testing` + SQLite in-memory | `repository → EntGo → DB` |

- Unit tests mock all repository interfaces via `mockgen`.
- Integration tests use a real SQLite in-memory database — no mocks.
- Build tag for integration tests: `//go:build integration`

```go
// Unit test example
func TestAuthService_Login(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()
    mockRepo := mocks.NewMockUserRepository(ctrl)
    mockRepo.EXPECT().FindByEmail(ctx, "test@example.com").Return(user, nil)
}
```

---

## 4. Frontend (`dash/`)

### 4.1 Language & Toolchain

| Item | Standard |
|------|----------|
| Node.js | LTS (22.x) |
| Package manager | `pnpm` |
| Framework | Next.js 16 (App Router) |
| Language | TypeScript (strict mode) |
| Linter | ESLint + TypeScript strict |
| Formatter | Prettier |
| Task runner | Taskfile v3 |

### 4.2 Naming Conventions

| Element | Convention | Example |
|---------|-----------|---------|
| Files (components) | PascalCase | `StreamOutput.tsx`, `AuthGuard.tsx` |
| Files (utilities/hooks) | camelCase | `useStream.ts`, `cn.ts` |
| Files (pages/layouts) | lowercase | `page.tsx`, `layout.tsx` |
| Components | PascalCase | `StreamOutput`, `AuthGuard` |
| Hooks | `use` prefix, camelCase | `useStream`, `useAuth` |
| Types/Interfaces | PascalCase | `User`, `ApiResponse`, `StreamEvent` |
| Constants | SCREAMING_SNAKE_CASE | `API_BASE_URL`, `TOKEN_KEY` |

### 4.3 Directory Structure

```
dash/
├── app/                    # Next.js App Router pages and layouts
│   ├── (auth)/             # Auth route group
│   ├── (dash)/        # Protected route group
│   └── offline/            # PWA offline fallback
├── components/
│   ├── ui/                 # Shadcn/ui generated — do not edit manually
│   └── shared/             # Application-level reusable components
├── lib/
│   ├── api/                # API client + TanStack Query hooks
│   └── utils/              # Pure utility functions
├── public/                 # Static assets, PWA manifest, icons
└── sw.ts                   # Serwist Service Worker entry point
```

> **Rule:** Never import from `app/` into `components/` or `lib/`. Data flows down via props or hooks.

### 4.4 Component Rules

- Every component is a **named export** (not default export), except page and layout files.
- Page and layout files use **default export** (Next.js requirement).
- Props interfaces are defined in the same file as the component.
- No inline styles — use Tailwind CSS utility classes only.

```tsx
// CORRECT
interface StreamOutputProps {
  content: string;
  isStreaming: boolean;
}

export function StreamOutput({ content, isStreaming }: StreamOutputProps) {
  return <div className="font-mono text-sm">{content}</div>;
}

// INCORRECT — avoid default export for shared components
export default function StreamOutput() { ... }
```

### 4.5 Data Fetching Rules

- All **server state** (API data) is managed via **TanStack Query** hooks in `lib/api/`.
- All **SSE streaming** is handled via the native `EventSource` API in a custom hook (`useStream`).
- No `fetch()` or `axios` calls directly inside components — always through hooks.
- No global state manager (Zustand/Redux) in v1.0.

```tsx
// CORRECT — use TanStack Query hook
const { data: user, isLoading } = useCurrentUser();

// INCORRECT — do not call fetch directly in component
const user = await fetch("/user/me");
```

### 4.6 TypeScript Rules

- `strict: true` in `tsconfig.json` — no exceptions.
- No `any` types — use `unknown` and narrow properly.
- All API response types are defined in `lib/api/types.ts`.

```ts
// CORRECT
interface ApiResponse<T> {
  success: boolean;
  data: T | null;
  error: ApiError | null;
}

// INCORRECT
const response: any = await fetch(...);
```

### 4.7 Testing Conventions

| Test Type | Tool | Location | Scope |
|-----------|------|----------|-------|
| Unit / Component | Vitest + React Testing Library | co-located `*.test.tsx` | Utilities, hooks, components |
| End-to-End | Playwright | `e2e/` directory | Auth flow, dash, streaming, PWA |

---

## 5. Git Conventions

### 5.1 Branch Naming

```
feature/<short-description>     # New features
fix/<short-description>         # Bug fixes
chore/<short-description>       # Tooling, deps, config
docs/<short-description>        # Documentation only
```

### 5.2 Commit Message Format

Follow **Conventional Commits**:

```
<type>(<scope>): <short summary>

feat(auth): implement Google OAuth2 callback handler
fix(config): resolve Viper env override for nested keys
chore(deps): update GoFiber to v3.1.0
docs(arch): add SSE endpoint documentation
test(service): add unit tests for auth service
```

| Type | When to Use |
|------|------------|
| `feat` | New feature |
| `fix` | Bug fix |
| `chore` | Build, deps, tooling |
| `docs` | Documentation changes |
| `test` | Adding or fixing tests |
| `refactor` | Code change without feature/fix |

### 5.3 Pull Request Rules

- Every PR must reference a task (e.g., `Closes PHASE-2-T3`).
- PRs must pass CI (lint + tests) before merge.
- No direct commits to `main`.

---

## 6. Environment Variable Rules

- All environment variables consumed by `api/` are prefixed with `OPUS_`.
- Nested config keys map to `OPUS_<SECTION>_<KEY>` (e.g., `auth.secret` → `OPUS_AUTH_SECRET`).
- All variables are documented in the root `.env.example`.
- Never hardcode environment-specific values in source code.
- Never commit `.env` files — only `.env.example`.

---

## 7. Docker & Deployment Rules

- `docker-compose.yml` — production configuration.
- `docker-compose.dev.yml` — development overrides (live reload, exposed debug ports).
- All Docker images are published to `ghcr.io/opus/`.
- SQLite is a valid production database for single-user deployments.

---

## 8. AI Agent Usage Guidelines

When using this document with an AI agent (e.g., Gemini):

1. **Always provide this file** at the start of each session as context.
2. **Provide the relevant PHASE file** for the current task.
3. **One task at a time** — do not ask the agent to implement multiple tasks simultaneously.
4. **Reference specific section numbers** when asking for corrections (e.g., "Fix this to follow Section 3.4 Interface Definition Rule").
5. **Validate acceptance criteria** against the checklist in each PHASE file before proceeding to the next task.