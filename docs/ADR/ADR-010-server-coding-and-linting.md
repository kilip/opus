# ADR-010: Server Coding Conventions & Linting

**Status:** Accepted
**Date:** 2026-05-17
**Deciders:** Chief Architect
**Context:** Opus Server (`opus/server/`)

---

## 1. Context

Opus Server is a modular monolith written in Go, structured around Clean Architecture (ADR-001).
While ADR-001 through ADR-009 define architectural boundaries, technology choices, and testing
strategy, no ADR establishes the day-to-day coding conventions that govern how Go code is
written within those boundaries.

Without explicit conventions, teams and AI agents risk producing inconsistent code — mismatched
error handling styles, missing GoDoc comments, undisciplined use of `panic`, implicit context
propagation, and ad-hoc linting configurations. These inconsistencies accumulate into technical
debt and make the codebase harder to navigate, review, and maintain.

This ADR establishes the canonical coding conventions and linting configuration for all Go code
under `opus/server/`. It is the authoritative reference for code style, error handling, naming,
interface definition, context propagation, and static analysis tooling.

> **Note for AI agents and automated tooling:** This ADR is the authoritative specification for
> all Go coding conventions in `opus/server/`. Do not invent patterns, naming conventions, or
> error handling styles beyond what is defined here. When generating code, verify compliance with
> each section of this ADR before emitting output.

---

## 2. Decision

Opus Server adopts a **strict, explicitly documented coding convention** enforced by
`golangci-lint` in CI. All conventions are Go-idiomatic and aligned with the Clean Architecture
boundaries established in ADR-001.

---

### 2.1 GoDoc Comments

Every exported symbol — function, method, type, constant, variable, and interface — **must** have
a GoDoc comment. This is a non-negotiable requirement enforced by the `revive` linter.

**Rules:**

- Comments must begin with the name of the symbol they describe.
- Comments must be complete sentences ending with a period.
- Package-level comments are required for every package.

```go
// Package agent provides the business logic and domain types for the Agent lifecycle domain.
package agent

// Agent represents a single autonomous agent instance managed by Opus.
type Agent struct {
    ID     string
    Name   string
    Status Status
}

// Status represents the lifecycle state of an Agent.
type Status string

const (
    // StatusIdle indicates the agent is not currently executing a task.
    StatusIdle Status = "idle"

    // StatusRunning indicates the agent is actively executing a task.
    StatusRunning Status = "running"
)

// Service handles all business logic for the Agent domain.
type Service struct {
    repo   Repository
    logger logger.Logger
}

// NewService constructs a new Service with the provided repository and logger.
func NewService(repo Repository, log logger.Logger) *Service {
    return &Service{repo: repo, logger: log}
}

// FindByID retrieves an Agent by its unique identifier.
// Returns ErrNotFound if no agent with the given ID exists.
func (s *Service) FindByID(ctx context.Context, id string) (*Agent, error) {
    // ...
}
```

---

### 2.2 Error Handling

#### 2.2.1 Sentinel Errors

Domain-level errors are defined as package-level sentinel variables using `errors.New`. They are
defined in the domain package (`internal/[feature]/`) and are the only errors that cross layer
boundaries.

```go
// internal/agent/errors.go
package agent

import "errors"

// ErrNotFound is returned when a requested agent does not exist.
var ErrNotFound = errors.New("agent: not found")

// ErrInvalidStatus is returned when a status transition is not permitted.
var ErrInvalidStatus = errors.New("agent: invalid status transition")
```

**Rules:**

- Sentinel errors are defined in `internal/[feature]/errors.go`.
- Sentinel error variable names use the `Err` prefix.
- Sentinel errors must have a GoDoc comment.
- Callers check sentinel errors using `errors.Is`, never string comparison.

```go
// Correct
if errors.Is(err, agent.ErrNotFound) { ... }

// Incorrect — never compare error strings
if err.Error() == "agent: not found" { ... }
```

#### 2.2.2 Error Wrapping

All errors returned from infrastructure calls (repository, queue, external API) **must** be
wrapped with context using `fmt.Errorf` and the `%w` verb. This preserves the error chain for
`errors.Is` and `errors.As`.

```go
// Correct — wraps the underlying error with context
func (s *Service) FindByID(ctx context.Context, id string) (*Agent, error) {
    a, err := s.repo.FindByID(ctx, id)
    if err != nil {
        return nil, fmt.Errorf("agent.Service.FindByID: %w", err)
    }
    return a, nil
}

// Incorrect — discards the error chain
return nil, errors.New("failed to find agent")

// Incorrect — loses the original error type
return nil, fmt.Errorf("failed to find agent: %s", err.Error())
```

**Wrapping convention:** `"<package>.<Type>.<Method>: %w"` — e.g.
`"agent.Service.FindByID: %w"`.

#### 2.2.3 `panic` Usage

`panic` is **prohibited** in all domain, service, adapter, and handler code. The only permitted
uses of `panic` are:

| Location | Permitted Use |
|---|---|
| `main.go` | Unrecoverable startup failure (e.g. config load, DB connection) |
| `queue.RegisterHandler` | Called after `Start()` — documented in ADR-008 |
| `internal/testutil/` | Test setup failure (via `t.Fatal`, not `panic`) |

```go
// Correct — return an error from application code
func (s *Service) FindByID(ctx context.Context, id string) (*Agent, error) {
    if id == "" {
        return nil, fmt.Errorf("agent.Service.FindByID: %w", ErrInvalidID)
    }
    // ...
}

// Incorrect — panic in application code
func (s *Service) FindByID(ctx context.Context, id string) (*Agent, error) {
    if id == "" {
        panic("id must not be empty")
    }
    // ...
}
```

#### 2.2.4 `log.Fatal` and `os.Exit`

`log.Fatal` and `os.Exit` are **prohibited** outside of `main.go`. Application code must return
errors; the decision to terminate the process is made exclusively in `main.go`.

---

### 2.3 Package and File Naming

| Entity | Convention | Example |
|---|---|---|
| Package | Lowercase, single word, no underscores | `agent`, `entgo`, `testutil` |
| File | Snake case, descriptive noun or noun phrase | `agent_handler.go`, `auth_repo.go` |
| Test file | Source file name + `_test` suffix | `agent_handler_test.go` |
| Integration test file | Source file name + `_integration_test` suffix | `agent_repo_integration_test.go` |
| Error file | `errors.go` per feature package | `internal/agent/errors.go` |
| Mock file | `mock_<interface_name>.go` | `mock_repository.go` |
| Generated file | Standard generated header comment | `// Code generated ... DO NOT EDIT.` |

**Package name must match the directory name.** The sole exception is the `main` package in
`main.go`.

---

### 2.4 Struct Conventions

#### 2.4.1 Constructors

Every struct that has dependencies (repository, logger, config, queue) **must** have a `New*`
constructor function. Direct struct literal instantiation of such types is not permitted outside
of the defining package.

```go
// Correct — constructor
func NewService(repo Repository, log logger.Logger, cfg Config) *Service {
    return &Service{
        repo:   repo,
        logger: log.With(logger.String("component", "agent_service")),
        cfg:    cfg,
    }
}

// Incorrect — direct instantiation from outside the package
svc := &agent.Service{} // compilation error — fields are unexported; this is enforced structurally
```

#### 2.4.2 Field Visibility

Struct fields are **unexported by default**. Exported fields are only permitted on:

- Config structs (e.g. `agent.Config`, `logger.Config`)
- Domain model structs (e.g. `agent.Agent`, `queue.Job`)
- API request/response structs in the delivery layer

```go
// Correct — service struct with unexported fields
type Service struct {
    repo   Repository
    logger logger.Logger
    cfg    Config
}

// Correct — domain model with exported fields
type Agent struct {
    ID        string
    Name      string
    Status    Status
    CreatedAt time.Time
}
```

---

### 2.5 Interface Conventions

#### 2.5.1 Definition Location

Interfaces are defined in the package that **uses** them (the consumer), not in the package that
implements them. This is the standard Go idiom and is already established in ADR-001 via the
repository pattern.

```go
// internal/agent/repository.go — defined in the consumer package (agent domain)
package agent

// Repository defines the persistence contract for the Agent domain.
// The concrete implementation is in adapter/entgo/agent_repo.go.
type Repository interface {
    FindByID(ctx context.Context, id string) (*Agent, error)
    FindAll(ctx context.Context, cursor string, limit int) ([]*Agent, string, error)
    Create(ctx context.Context, agent *Agent) (*Agent, error)
    UpdateStatus(ctx context.Context, id string, status Status) error
    Delete(ctx context.Context, id string) error
}
```

> **Note for AI agents:** Interfaces are always defined in `internal/[feature]/` — never in
> `adapter/` or `delivery/`. The adapter and delivery layers implement interfaces; they do not
> define them.

#### 2.5.2 Naming

- Interface names are nouns or noun phrases (`Repository`, `Logger`, `Queue`, `EventBus`).
- The `I` prefix is **prohibited** (`IRepository` is invalid).
- Single-method interfaces should be named with the method name plus the `-er` suffix where
  idiomatic (`Reloadable`, `Stringer`).

#### 2.5.3 Interface Size

Interfaces should be as small as the consumer requires. Prefer multiple small interfaces over a
single large interface. If a function only needs `FindByID`, it should accept an interface with
only that method, not the full `Repository`.

```go
// Preferred for a function that only reads
type AgentFinder interface {
    FindByID(ctx context.Context, id string) (*Agent, error)
}

// Acceptable for a service that requires full CRUD
type Repository interface {
    FindByID(ctx context.Context, id string) (*Agent, error)
    Create(ctx context.Context, agent *Agent) (*Agent, error)
    // ...
}
```

#### 2.5.4 Mock Generation Directive

Every interface that requires mocking in tests **must** carry a `//go:generate` directive
immediately above the interface declaration.

```go
//go:generate mockgen -destination=mock_repository.go -package=agent . Repository

// Repository defines the persistence contract for the Agent domain.
type Repository interface {
    // ...
}
```

---

### 2.6 Context Propagation

#### 2.6.1 `context.Context` as First Parameter

Every function or method that performs I/O, calls another service, or may block **must** accept
`context.Context` as its first parameter, named `ctx`.

```go
// Correct
func (s *Service) FindByID(ctx context.Context, id string) (*Agent, error)

// Incorrect — context omitted
func (s *Service) FindByID(id string) (*Agent, error)

// Incorrect — context not first
func (s *Service) FindByID(id string, ctx context.Context) (*Agent, error)
```

#### 2.6.2 No Context in Structs

`context.Context` **must never** be stored in a struct field. A context must be passed explicitly
to each function call.

```go
// Incorrect — context stored in struct
type Service struct {
    ctx  context.Context
    repo Repository
}

// Correct — context passed per call
func (s *Service) FindByID(ctx context.Context, id string) (*Agent, error) {
    return s.repo.FindByID(ctx, id)
}
```

#### 2.6.3 Context Enrichment

Context enrichment (adding tracing metadata) is performed exclusively in the delivery layer
middleware, following ADR-006. Domain and adapter layer code extracts values from context but
never adds new keys.

```go
// Correct — delivery layer middleware enriches context
ctx := logger.WithRequestID(context.Background(), requestID)

// Incorrect — domain service adds arbitrary context values
ctx = context.WithValue(ctx, "my_key", "my_value") // prohibited in internal/
```

---

### 2.7 Logging Conventions

All logging in `opus/server/` follows ADR-006. The conventions below complement ADR-006 with
specific usage rules.

- **Direct use of `fmt.Println`, `log.Print*`, or any concrete logging library is prohibited**
  outside of `main.go`. All logging goes through the injected `logger.Logger` interface.
- Services bind a `component` field in their constructor via `logger.With`.
- Sensitive values (passwords, tokens, API keys, email addresses) **must** use `logger.Redact`.
- Log messages are lowercase, present tense, without trailing punctuation.
  - Correct: `"login attempt started"`
  - Incorrect: `"Login attempt started."` / `"Started login attempt"`

```go
// Correct
s.logger.InfoCtx(ctx, "agent evaluation started",
    logger.String("agent_id", id),
)

// Incorrect — direct stdlib log
log.Printf("agent %s evaluation started", id)

// Incorrect — uppercase / trailing punctuation
s.logger.InfoCtx(ctx, "Agent evaluation started.", logger.String("agent_id", id))
```

---

### 2.8 `go generate` Conventions

- All `//go:generate` directives are placed in the source file that defines the type or interface
  being generated for (e.g. `repository.go`, not `service.go`).
- Generated files **must not** be edited manually.
- Generated files **must** carry the standard Go generated file header on line 1:
  `// Code generated by <tool>. DO NOT EDIT.`
- `go generate ./...` must be run from `server/` before committing any interface or schema
  change.
- CI validates that generated files are up-to-date by running `go generate ./...` and checking
  for uncommitted changes.

---

### 2.9 Linting — `golangci-lint`

#### 2.9.1 Configuration File

The linting configuration lives at `server/.golangci.yml` and is the single source of truth for
all static analysis rules. It is committed to the repository and applied identically in CI and
local development.

#### 2.9.2 Enabled Linters

```yaml
# server/.golangci.yml
version: "2"

linters:
  enable:
    - errcheck        # Ensures all error return values are handled
    - govet           # Reports suspicious constructs (go vet)
    - staticcheck     # Comprehensive static analysis (SA*, S1*, ST1*)
    - revive          # Replaces golint; enforces GoDoc, naming, and style rules
    - gofmt           # Enforces gofmt formatting
    - goimports       # Enforces goimports (gofmt + import grouping)
    - godot           # Enforces period at end of GoDoc comments
    - wrapcheck       # Ensures errors from external packages are wrapped
    - exhaustive      # Ensures switch statements on enums are exhaustive
    - noctx           # Disallows http.Request without context
    - contextcheck    # Ensures functions that accept context pass it to callees
    - unparam         # Reports unused function parameters
    - misspell        # Detects common spelling mistakes in comments and strings
    - whitespace      # Detects unnecessary whitespace

linters-settings:
  revive:
    rules:
      - name: exported
        severity: error
        arguments:
          - "checkPrivateReceivers"
          - "sayRepetitiveInsteadOfStutters"
      - name: var-naming
        severity: error
      - name: error-return
        severity: error
      - name: unused-parameter
        severity: warning

  wrapcheck:
    ignorePackageGlobs:
      - "github.com/kilip/opus/server/internal/*"

  exhaustive:
    default-signifies-exhaustive: true

issues:
  exclude-dirs:
    - ent              # Exclude generated Ent code
    - vendor
  exclude-files:
    - ".*_mock\\.go$"  # Exclude generated mock files
    - ".*\\.pb\\.go$"  # Exclude generated protobuf files
  exclude-rules:
    - path: "_test\\.go"
      linters:
        - wrapcheck    # Test files are exempt from error wrapping requirement
        - unparam      # Test helpers may have fixed parameter signatures
```

#### 2.9.3 Zero Warning Policy

CI enforces a **zero warning policy** — any linter finding, warning or error, causes the CI
pipeline to fail. Warnings are not suppressed without explicit justification.

#### 2.9.4 Per-Line Suppression

Linter suppression is permitted only for specific, documented reasons. Suppression must include
a comment explaining why.

```go
// Correct — suppression with justification
_ = result //nolint:errcheck // result is always nil for this implementation; verified by unit test

// Incorrect — suppression without justification
_ = result //nolint:errcheck
```

Blanket `//nolint` (without a specific linter name) is **prohibited**.

#### 2.9.5 Local Execution

```bash
# Install golangci-lint (from server/ directory)
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run linter
cd server && golangci-lint run ./...

# Run linter with auto-fix where possible
cd server && golangci-lint run --fix ./...
```

---

### 2.10 Import Grouping

Imports are organised into three groups, separated by blank lines, enforced by `goimports`:

1. Standard library
2. Third-party packages
3. Internal packages (`github.com/kilip/opus/server/...`)

```go
import (
    "context"
    "fmt"
    "time"

    "github.com/gofiber/fiber/v3"
    "go.uber.org/mock/gomock"

    "github.com/kilip/opus/server/internal/agent"
    "github.com/kilip/opus/server/internal/shared/logger"
)
```

---

### 2.11 Alignment with Existing ADRs

| Convention | This ADR | Related ADR |
|---|---|---|
| GoDoc on all exported symbols | §2.1 | ADR-001 (code quality non-negotiables) |
| Sentinel errors in domain layer | §2.2.1 | ADR-001 (feature package isolation) |
| No `log.Fatal` outside `main.go` | §2.2.4 | ADR-006 (Logger interface) |
| Interface defined in consumer package | §2.5.1 | ADR-001 (repository pattern) |
| `ctx` as first parameter | §2.6.1 | ADR-006 (context-aware logger) |
| No direct logging library calls | §2.7 | ADR-006 (Logger interface) |
| `//go:generate` for mocks | §2.8 | ADR-009 (mock generation) |
| `golangci-lint` in CI | §2.9 | ADR-009 (CI enforcement) |

---

## 3. Alternatives Considered

### 3.1 `golint` (deprecated)

The original Go linting tool. Rejected because it has been officially deprecated in favour of
`revive`, which is a drop-in replacement with additional configurability.

### 3.2 `staticcheck` Only

Using `staticcheck` as the sole linter without `golangci-lint`. Rejected because `golangci-lint`
aggregates multiple linters under a single configuration file and execution command, reducing CI
complexity and allowing fine-grained per-linter configuration.

### 3.3 `uber-go/guide` Style Guide Adoption

Adopting the Uber Go Style Guide wholesale. Not adopted because it contains conventions that
conflict with standard Go idioms (e.g. Uber's preference for `errors.New` with `fmt.Errorf`
wrapping differs subtly from the convention defined here). Opus's conventions are derived from
the Go standard library, `effective_go`, and the specific architectural constraints of ADR-001.

---

## 4. Consequences

### 4.1 Positive

- **AI agent compliance** — Explicit, unambiguous conventions prevent AI-generated code from
  introducing inconsistent patterns. Each section is written to be directly consumable by
  automated tooling.
- **Consistent error traceability** — Mandatory error wrapping with the
  `<package>.<Type>.<Method>: %w` convention creates a traceable, unwrappable error chain across
  all layers.
- **Self-documenting codebase** — Mandatory GoDoc on all exported symbols ensures the codebase
  remains navigable without external documentation.
- **Zero linting debt** — Zero warning CI policy prevents gradual accumulation of suppressed
  warnings.
- **Interface clarity** — Explicit interface definition location (consumer package) and mock
  generation directives prevent ambiguity for contributors and AI agents alike.

### 4.2 Negative / Trade-offs

- **Onboarding friction** — New contributors must read this ADR before writing code; the
  conventions are stricter than a typical Go project.
- **`wrapcheck` verbosity** — Mandatory error wrapping adds boilerplate in adapter layer code;
  mitigated by the `ignorePackageGlobs` exclusion for internal packages.
- **`exhaustive` linter maintenance** — Exhaustive switch enforcement requires updating switch
  statements whenever a new enum value is added to a domain type; this is intentional but adds a
  small maintenance cost.
- **GoDoc discipline** — The `godot` linter enforces periods at the end of GoDoc comments; this
  is a minor but occasionally surprising requirement for contributors from other Go projects.

---

## 5. References

- [ADR-001: Server Clean Architecture](./ADR-001-server-clean-architecture.md)
- [ADR-006: Server Logger Architecture](./ADR-006-server-logger.md)
- [ADR-008: Server Queue Architecture](./ADR-008-server-queue.md)
- [ADR-009: Server Testing Strategy](./ADR-009-server-testing-strategy.md)
- [Effective Go](https://go.dev/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [golangci-lint](https://golangci-lint.run)
- [revive linter](https://github.com/mgechev/revive)
- [go.uber.org/mock](https://github.com/uber-go/mock)
- [wrapcheck linter](https://github.com/tomarrell/wrapcheck)