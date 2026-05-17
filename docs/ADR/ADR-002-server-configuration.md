# ADR-002: Configuration Management

**Status:** Accepted  
**Date:** 2026-05-17  
**Deciders:** Chief Architect  
**Context:** Opus Server (`opus/server/`)

---

## Context and Problem Statement

Opus Server requires a structured, type-safe configuration system that supports multiple deployment environments (development, production), hot-reload without server restarts, IDE autocompletion via JSON Schema, and secret injection through environment variables without embedding credentials in config files.

The configuration system must be discoverable across standard locations, extensible as new domains are added, and validated at startup to surface misconfiguration early.

Additionally, as established in ADR-001, each feature domain must remain self-contained. This requires a config strategy that allows feature packages to own the shape of their configuration without coupling them to the central config loader.

---

## Decision Drivers

- Config file must be human-editable JSON with IDE autocompletion support
- Hot-reload required — configuration changes must apply without a server restart
- Secrets (API keys, credentials) must be injectable via environment variables only
- JSON Schema must be generated at build time, not runtime
- Config resolution must follow a deterministic precedence order across multiple lookup paths
- Zero mandatory cloud dependency — fully functional offline
- Feature packages must own their own config struct definitions (aligns with ADR-001 feature-based clean architecture)
- Feature packages must not import the central `config` package (dependency rule)

---

## Considered Options

1. **Viper + JSON + `invopop/jsonschema` build-time generation + hybrid composition** *(chosen)*
2. Viper + YAML + manual schema documentation
3. Custom config loader with `encoding/json` only
4. `koanf` + TOML

---

## Decision Outcome

**Chosen option: Option 1 — Viper + JSON + build-time JSON Schema generation + hybrid composition**

Viper provides production-grade config resolution with environment variable override, file watching, and multi-source merging. JSON is chosen over YAML as the config format because it has first-class JSON Schema tooling support, enabling IDE autocompletion and validation without additional plugins. JSON Schema is generated at build time via `go generate` and committed to the repository for distribution.

A **hybrid config composition pattern** is adopted: each feature domain defines its own config struct in `internal/[feature]/config.go`, and the root `internal/config/model.go` composes these structs into the top-level `Config`. Feature packages receive their own config struct via constructor injection and have no knowledge of Viper or the central config loader.

### Positive Consequences

- IDE autocompletion and inline validation via `$schema` field in `config.json`
- Hot-reload via Viper's `WatchConfig` + `OnConfigChange` callback
- Secrets never written to disk — env vars override any config file value
- Single source of truth for config structure: the Go struct hierarchy
- Schema can be distributed alongside releases for self-hosted user tooling
- Feature packages remain fully self-contained — config shape is co-located with domain logic
- Microservice extraction is clean — a feature carries its own `config.go` without modification

### Negative Consequences / Trade-offs

- JSON does not support comments — operators cannot annotate config values inline; mitigated by JSON Schema descriptions surfaced in IDE tooltips
- `invopop/jsonschema` requires struct tags to be kept in sync; schema drift is possible if tags are neglected
- Viper's `WatchConfig` uses `fsnotify` which has known edge cases on network-mounted filesystems (NFS, Docker volumes); documented as a known limitation
- Root `internal/config/model.go` must import each feature package to compose their config structs; this is the only permitted direction of cross-domain imports for config

---

## Implementation Specification

### 2.1 Config File Resolution Order

Viper resolves configuration from the following locations in **descending precedence**:

| Priority | Source | Notes |
|---|---|---|
| 1 | Environment variables | Prefixed `OPUS_`, override all file values |
| 2 | `OPUS_HOME/config.json` | Explicit override via env var |
| 3 | `~/.opus/config.json` | User-level config (production default) |
| 4 | `opus/.opus/config.json` | Project-level config (development default) |

Resolution stops at the first file found. Layered merging across multiple files is not used in MVP.

```go
// internal/config/loader.go
package config

import (
    "os"
    "path/filepath"

    "github.com/fsnotify/fsnotify"
    "github.com/spf13/viper"
)

func Load() (*Config, error) {
    v := viper.New()
    v.SetConfigName("config")
    v.SetConfigType("json")

    // Resolution order: lowest to highest priority
    v.AddConfigPath(filepath.Join("opus", ".opus"))            // development
    v.AddConfigPath(filepath.Join(os.Getenv("HOME"), ".opus")) // user home

    if opusHome := os.Getenv("OPUS_HOME"); opusHome != "" {
        v.AddConfigPath(opusHome) // explicit override
    }

    v.SetEnvPrefix("OPUS")
    v.AutomaticEnv()

    if err := v.ReadInConfig(); err != nil {
        return nil, err
    }

    var cfg Config
    if err := v.Unmarshal(&cfg); err != nil {
        return nil, err
    }

    return &cfg, nil
}
```

---

### 2.2 Hybrid Config Composition Pattern

Each feature domain defines the shape of its own configuration in `internal/[feature]/config.go`. The root `internal/config/model.go` composes these feature config structs into the top-level `Config`. Viper unmarshals directly into the composed struct.

#### Dependency Rule

```
internal/config  →  internal/[feature]  ✅  (root config imports feature config)
internal/[feature]  →  internal/config  ❌  (features must never import root config)
```

Feature services receive only their own config struct via constructor injection. They have no dependency on Viper, the loader, or the root `Config` struct.

#### Feature Config Definition

Each feature defines its config struct independently:

```go
// internal/agent/config.go
package agent

type Config struct {
    TickInterval string `mapstructure:"tick_interval" json:"tick_interval" jsonschema:"default=60s,description=Interval between autonomous agent evaluation cycles (Go duration string)"`
    MaxRetries   int    `mapstructure:"max_retries"   json:"max_retries"   jsonschema:"default=3,description=Maximum retries for a failed agent task before marking it as errored"`
}
```

```go
// internal/vault/config.go
package vault

type Config struct {
    Path string `mapstructure:"path" json:"path" jsonschema:"description=Absolute or relative path to the vault root directory"`
}
```

```go
// internal/llm/config.go
package llm

type Config struct {
    Provider  string `mapstructure:"provider"   json:"provider"   jsonschema:"enum=anthropic,enum=openai,enum=ollama,description=Active LLM provider"`
    BaseURL   string `mapstructure:"base_url"   json:"base_url"   jsonschema:"description=Override provider base URL (e.g. for Ollama local endpoint)"`
    APIKey    string `mapstructure:"api_key"    json:"-"          jsonschema:"-"`
    Model     string `mapstructure:"model"      json:"model"      jsonschema:"description=Model identifier passed to provider"`
    MaxTokens int    `mapstructure:"max_tokens" json:"max_tokens" jsonschema:"default=4096,description=Maximum tokens per completion request"`
}
```

#### Root Config Composition

The root config struct imports and embeds each feature config struct. It owns no business-level field definitions beyond infrastructure concerns (server, database, log):

```go
// internal/config/model.go
//go:generate go run generate.go
package config

import (
    "opus/server/internal/agent"
    "opus/server/internal/delivery/gofiber"
    "opus/server/internal/llm"
    "opus/server/internal/vault"
    "opus/server/internal/workflow"
)

type Config struct {
    Server   gofiber.Config `mapstructure:"server"   json:"server"   jsonschema:"required"`
    Database DatabaseConfig `mapstructure:"database" json:"database" jsonschema:"required"`
    Log      LogConfig      `mapstructure:"log"      json:"log"`
    LLM      llm.Config     `mapstructure:"llm"      json:"llm"      jsonschema:"required"`
    Agent    agent.Config   `mapstructure:"agent"    json:"agent"`
    Vault    vault.Config   `mapstructure:"vault"    json:"vault"`
    Workflow workflow.Config `mapstructure:"workflow" json:"workflow"`
}

type DatabaseConfig struct {
    Driver string `mapstructure:"driver" json:"driver" jsonschema:"enum=sqlite3,enum=postgres,default=sqlite3,description=Database driver"`
    DSN    string `mapstructure:"dsn"    json:"dsn"    jsonschema:"description=Data source name. Use env var OPUS_DATABASE_DSN for secrets"`
}

type LogConfig struct {
    Level  string `mapstructure:"level"  json:"level"  jsonschema:"enum=debug,enum=info,enum=warn,enum=error,default=info"`
    Format string `mapstructure:"format" json:"format" jsonschema:"enum=json,enum=text,default=json"`
}
```

#### Constructor Injection in `main.go`

Feature services receive only their own config slice. They have no awareness of the root `Config`:

```go
// main.go
package main

import (
    "opus/server/adapter/entgo"
    "opus/server/internal/delivery/gofiber/handler"
    "opus/server/internal/delivery/gofiber"
    "opus/server/internal/agent"
    "opus/server/internal/auth"
    "opus/server/internal/config"
    "opus/server/internal/vault"
)

func main() {
    cfg, err := config.Load()
    if err != nil {
        panic(err)
    }

    // Adapter layer
    db := entgo.NewClient(cfg.Database)
    authRepo := entgo.NewAuthRepo(db)
    agentRepo := entgo.NewAgentRepo(db)
    vaultRepo := entgo.NewVaultRepo(db)

    // Service layer — each service receives only its own config slice
    authService := auth.NewService(authRepo, cfg.Server)
    agentService := agent.NewService(agentRepo, cfg.Agent)
    vaultService := vault.NewService(vaultRepo, cfg.Vault)

    // Delivery layer
    auth := handler.NewAuth(authService)
    agent := handler.NewAgent(agentService)
    vault := handler.NewVault(vaultService)

    // Bootstrap
    app := gofiber.New(auth, agent, vault)
    app.Listen(cfg.Server.Address)
}
```

---

### 2.3 Hot-Reload

Viper watches the resolved config file via `fsnotify`. On change, the updated values are unmarshalled and the new `Config` is delivered to registered subscribers via the global `Reloadable` interface.

Hot-reload is managed globally — individual feature services do not implement per-feature reload logic. Services that require runtime reconfiguration implement the `Reloadable` interface and register with the config watcher at startup.

```go
// internal/config/loader.go (continued)

func Watch(v *viper.Viper, onChange func(cfg *Config)) {
    v.WatchConfig()
    v.OnConfigChange(func(e fsnotify.Event) {
        var cfg Config
        if err := v.Unmarshal(&cfg); err != nil {
            // log error; do not apply partial config
            return
        }
        onChange(&cfg)
    })
}
```

```go
// internal/config/reloadable.go
package config

// Reloadable is implemented by services that support runtime reconfiguration
// without a server restart. Reload is called by the config watcher on each
// successful config file change.
type Reloadable interface {
    Reload(cfg *Config)
}
```

> **Note:** `Reloadable` accepts the root `*Config` to provide the full updated config to any service that needs it. This is intentional — the watcher operates at the infrastructure level, not the feature level.

---

### 2.4 Environment Variable Conventions

All secrets and environment-specific overrides follow the `OPUS_<SECTION>_<KEY>` naming convention.

| Environment Variable | Config Path | Notes |
|---|---|---|
| `OPUS_DATABASE_DSN` | `database.dsn` | Database connection string |
| `OPUS_LLM_APIKEY` | `llm.api_key` | LLM provider API key — never in config file |
| `OPUS_SERVER_ADDRESS` | `server.address` | Override listen address |
| `OPUS_HOME` | — | Config directory override (not a Viper key) |

The `APIKey` field in `llm.Config` carries `json:"-"` to prevent accidental serialization and `jsonschema:"-"` to exclude it from the generated schema.

---

### 2.5 JSON Schema Build-Time Generation

Schema generation is triggered via `go generate`. The generator reads the composed `Config` struct via reflection using `github.com/invopop/jsonschema` — which recursively processes embedded feature config structs — and writes the output to `docs/config.schema.json`. The generated file is committed to the repository.

```go
// internal/config/generate.go
//go:build ignore

package main

import (
    "encoding/json"
    "os"

    "github.com/invopop/jsonschema"
    "opus/server/internal/config"
)

func main() {
    r := new(jsonschema.Reflector)
    schema := r.Reflect(&config.Config{})

    out, err := json.MarshalIndent(schema, "", "  ")
    if err != nil {
        panic(err)
    }

    if err := os.WriteFile("docs/config.schema.json", out, 0644); err != nil {
        panic(err)
    }
}
```

Run with:

```bash
cd opus/server/internal/config && go generate
```

---

### 2.6 Example `config.json` (Development)

```json
{
  "$schema": "../../docs/config.schema.json",
  "server": {
    "address": ":8080",
    "debug": true
  },
  "database": {
    "driver": "sqlite3",
    "dsn": "opus.db"
  },
  "llm": {
    "provider": "anthropic",
    "model": "claude-sonnet-4-20250514",
    "max_tokens": 4096
  },
  "agent": {
    "tick_interval": "60s",
    "max_retries": 3
  },
  "vault": {
    "path": "./vault"
  },
  "log": {
    "level": "debug",
    "format": "text"
  }
}
```

---

### 2.7 Directory Structure

This ADR extends the `internal/config/` package established in ADR-001. Each feature package gains a `config.go` file. No new top-level directories are introduced.

```
opus/server/
├── internal/
│   ├── config/
│   │   ├── model.go        # Root Config struct — composes feature config structs; go:generate directive
│   │   ├── loader.go       # Viper setup, resolution order, Watch()
│   │   ├── reloadable.go   # Reloadable interface (global)
│   │   └── generate.go     # go:build ignore — schema generator entrypoint
├── delivery/
│   └── gofiber/
│       ├── config.go       # gofiber.Config — owned by delivery layer
│       ├── router.go
│       └── response.go
├── agent/
│   ├── config.go       # agent.Config — owned by feature
│   ├── model.go
│   ├── repository.go
│   └── service.go
├── vault/
│   ├── config.go       # vault.Config — owned by feature
│   ├── model.go
│   ├── repository.go
│   └── service.go
├── workflow/
│   ├── config.go       # workflow.Config — owned by feature
│   ├── model.go
│   ├── repository.go
│   └── service.go
└── llm/
    ├── config.go       # llm.Config — owned by feature
    ├── model.go
    └── router.go
│
docs/
└── config.schema.json      # Generated — committed to repository

opus/.opus/
└── config.json             # Development config — committed with safe defaults only
```

---

## Pros and Cons of Options

### Option 1 — Viper + JSON + build-time JSON Schema + hybrid composition *(chosen)*

| | |
|---|---|
| ✅ | First-class JSON Schema tooling; IDE autocompletion with zero plugin configuration |
| ✅ | Viper handles env var override, file watching, and unmarshalling with a single dependency |
| ✅ | Schema generated recursively from composed struct — no manual documentation drift |
| ✅ | Feature config structs are self-contained and travel with the feature on microservice extraction |
| ✅ | `json:"-"` pattern prevents accidental secret serialization |
| ❌ | JSON does not support inline comments |
| ❌ | Root `config/model.go` must import all feature packages; adding a feature requires updating the root |
| ❌ | Viper `WatchConfig` has known edge cases on network-mounted filesystems |

### Option 2 — Viper + YAML + manual schema documentation

| | |
|---|---|
| ✅ | YAML supports comments — operators can annotate config inline |
| ❌ | No standardised JSON Schema tooling for YAML; IDE support is inconsistent across editors |
| ❌ | Manual schema documentation diverges from code over time |

### Option 3 — Custom loader with `encoding/json` only

| | |
|---|---|
| ✅ | Zero additional dependencies |
| ❌ | No env var override or file watching — must be implemented from scratch |
| ❌ | Not justified for MVP given Viper's maturity |

### Option 4 — `koanf` + TOML

| | |
|---|---|
| ✅ | `koanf` is more modular than Viper; TOML supports comments |
| ❌ | Smaller ecosystem; less community precedent in Go server projects |
| ❌ | TOML has no JSON Schema equivalent for IDE autocompletion |

---

## References

- [ADR-001: Server Clean Architecture](./ADR-001-server-clean-architecture.md)
- [Viper — spf13/viper](https://github.com/spf13/viper)
- [invopop/jsonschema — JSON Schema from Go structs](https://github.com/invopop/jsonschema)
- [JSON Schema — IDE Integration](https://json-schema.org/implementations#editors)
- [fsnotify — File system notifications](https://github.com/fsnotify/fsnotify)