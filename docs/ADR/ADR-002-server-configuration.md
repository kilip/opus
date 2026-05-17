# ADR-002: Configuration Management

**Status:** Accepted
**Date:** 2026-05-17
**Deciders:** Chief Architect
**Context:** Opus Server (`opus/server/`)

---

## Context and Problem Statement

Opus Server requires a structured, type-safe configuration system that supports multiple deployment environments (development, production), hot-reload without server restarts, IDE autocompletion via JSON Schema, and secret injection through environment variables without embedding credentials in config files.

The configuration system must be discoverable across standard locations, extensible as new domains are added, and validated at startup to surface misconfiguration early.

---

## Decision Drivers

- Config file must be human-editable JSON with IDE autocompletion support
- Hot-reload required — configuration changes must apply without a server restart
- Secrets (API keys, credentials) must be injectable via environment variables only
- JSON Schema must be generated at build time, not runtime
- Config resolution must follow a deterministic precedence order across multiple lookup paths
- Zero mandatory cloud dependency — fully functional offline

---

## Considered Options

1. **Viper + JSON + `invopop/jsonschema` build-time generation** *(chosen)*
2. Viper + YAML + manual schema documentation
3. Custom config loader with `encoding/json` only
4. `koanf` + TOML

---

## Decision Outcome

**Chosen option: Option 1 — Viper + JSON + build-time JSON Schema generation**

Viper provides production-grade config resolution with environment variable override, file watching, and multi-source merging. JSON is chosen over YAML as the config format because it has first-class JSON Schema tooling support, enabling IDE autocompletion and validation without additional plugins. JSON Schema is generated at build time via `go generate` and committed to the repository for distribution.

### Positive Consequences

- IDE autocompletion and inline validation via `$schema` field in `config.json`
- Hot-reload via Viper's `WatchConfig` + `OnConfigChange` callback
- Secrets never written to disk — env vars override any config file value
- Single source of truth for config structure: the Go struct with struct tags
- Schema can be distributed alongside releases for self-hosted user tooling

### Negative Consequences / Trade-offs

- JSON does not support comments — operators cannot annotate config values inline; mitigated by JSON Schema descriptions surfaced in IDE tooltips
- `invopop/jsonschema` requires struct tags to be kept in sync; schema drift is possible if tags are neglected
- Viper's `WatchConfig` uses `fsnotify` which has known edge cases on network-mounted filesystems (NFS, Docker volumes); documented as a known limitation

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

### 2.2 Config Struct Definition

The Go struct is the single source of truth. All fields carry `mapstructure` tags (Viper), `json` tags (serialization), and `jsonschema` tags (schema generation).

```go
// internal/config/model.go
//go:generate go run generate.go
package config

type Config struct {
    Server   ServerConfig   `mapstructure:"server"   json:"server"   jsonschema:"required"`
    Database DatabaseConfig `mapstructure:"database" json:"database" jsonschema:"required"`
    LLM      LLMConfig      `mapstructure:"llm"      json:"llm"      jsonschema:"required"`
    Agent    AgentConfig    `mapstructure:"agent"    json:"agent"`
    Vault    VaultConfig    `mapstructure:"vault"    json:"vault"`
    Log      LogConfig      `mapstructure:"log"      json:"log"`
}

type ServerConfig struct {
    Address string `mapstructure:"address" json:"address" jsonschema:"default=:8080,description=TCP address the HTTP server listens on"`
    Debug   bool   `mapstructure:"debug"   json:"debug"   jsonschema:"description=Enable debug mode and verbose request logging"`
}

type DatabaseConfig struct {
    Driver string `mapstructure:"driver" json:"driver" jsonschema:"enum=sqlite3,enum=postgres,default=sqlite3,description=Database driver"`
    DSN    string `mapstructure:"dsn"    json:"dsn"    jsonschema:"description=Data source name. Use env var OPUS_DATABASE_DSN for secrets"`
}

type LLMConfig struct {
    Provider  string `mapstructure:"provider"   json:"provider"   jsonschema:"enum=anthropic,enum=openai,enum=ollama,description=Active LLM provider"`
    BaseURL   string `mapstructure:"base_url"   json:"base_url"   jsonschema:"description=Override provider base URL (e.g. for Ollama local endpoint)"`
    // APIKey must be injected via OPUS_LLM_APIKEY env var — never written to config file
    APIKey    string `mapstructure:"api_key"    json:"-"          jsonschema:"-"`
    Model     string `mapstructure:"model"      json:"model"      jsonschema:"description=Model identifier passed to provider"`
    MaxTokens int    `mapstructure:"max_tokens" json:"max_tokens" jsonschema:"default=4096,description=Maximum tokens per completion request"`
}

type AgentConfig struct {
    TickInterval string `mapstructure:"tick_interval" json:"tick_interval" jsonschema:"default=60s,description=Interval between autonomous agent evaluation cycles (Go duration string)"`
    MaxRetries   int    `mapstructure:"max_retries"   json:"max_retries"   jsonschema:"default=3,description=Maximum retries for a failed agent task before marking it as errored"`
}

type VaultConfig struct {
    Path string `mapstructure:"path" json:"path" jsonschema:"description=Absolute or relative path to the vault root directory"`
}

type LogConfig struct {
    Level  string `mapstructure:"level"  json:"level"  jsonschema:"enum=debug,enum=info,enum=warn,enum=error,default=info"`
    Format string `mapstructure:"format" json:"format" jsonschema:"enum=json,enum=text,default=json"`
}
```

### 2.3 Hot-Reload

Viper watches the resolved config file via `fsnotify`. On change, the updated values are unmarshalled and the new `Config` is delivered to registered subscribers. Services requiring reload awareness implement the `Reloadable` interface.

```go
// internal/config/loader.go (continued)

func Watch(v *viper.Viper, onChange func(cfg *Config)) {
    v.WatchConfig()
    v.OnConfigChange(func(e fsnotify.Event) {
        var cfg Config
        if err := v.Unmarshal(&cfg); err != nil {
            // log error, do not apply partial config
            return
        }
        onChange(&cfg)
    })
}
```

```go
// internal/config/reloadable.go
package config

type Reloadable interface {
    Reload(cfg *Config)
}
```

### 2.4 Environment Variable Conventions

All secrets and environment-specific overrides follow the `OPUS_<SECTION>_<KEY>` naming convention.

| Environment Variable | Config Path | Notes |
|---|---|---|
| `OPUS_DATABASE_DSN` | `database.dsn` | Database connection string |
| `OPUS_LLM_APIKEY` | `llm.api_key` | LLM provider API key — never in config file |
| `OPUS_SERVER_ADDRESS` | `server.address` | Override listen address |
| `OPUS_HOME` | — | Config directory override (not a Viper key) |

The `APIKey` field carries `json:"-"` to prevent accidental serialization and `jsonschema:"-"` to exclude it from the generated schema.

### 2.5 JSON Schema Build-Time Generation

Schema generation is triggered via `go generate`. The generator reads the `Config` struct via reflection using `github.com/invopop/jsonschema` and writes the output to `docs/config.schema.json`. The generated file is committed to the repository.

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

### 2.7 Directory Integration with ADR-001

This ADR extends the `internal/config/` package established in ADR-001. No new top-level directories are introduced.

```
opus/server/
└── internal/
    └── config/
        ├── model.go        # Config struct — single source of truth; go:generate directive
        ├── loader.go       # Viper setup, resolution order, Watch()
        ├── reloadable.go   # Reloadable interface
        └── generate.go     # go:build ignore — schema generator entrypoint

docs/
└── config.schema.json      # Generated — committed to repository

opus/.opus/
└── config.json             # Development config — committed with safe defaults only
```

---

## Pros and Cons of Options

### Option 1 — Viper + JSON + build-time JSON Schema *(chosen)*

| | |
|---|---|
| ✅ | First-class JSON Schema tooling; IDE autocompletion with zero plugin configuration |
| ✅ | Viper handles env var override, file watching, and unmarshalling with a single dependency |
| ✅ | Schema generated from struct — single source of truth, no manual documentation drift |
| ✅ | `json:"-"` pattern prevents accidental secret serialization |
| ❌ | JSON does not support inline comments |
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

- [Viper — spf13/viper](https://github.com/spf13/viper)
- [invopop/jsonschema — JSON Schema from Go structs](https://github.com/invopop/jsonschema)
- [JSON Schema — IDE Integration](https://json-schema.org/implementations#editors)
- [fsnotify — File system notifications](https://github.com/fsnotify/fsnotify)
- [ADR-001: Server Clean Architecture](./ADR-001-server-clean-architecture.md)