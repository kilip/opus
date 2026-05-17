# Design Spec: Base Viper Configuration Integration

## Objective
Implement the foundational configuration management system for Opus Server using Viper, adhering to ADR-002. This scope is strictly limited to path resolution, auto-creation of configuration directories, and basic loading mechanics, with a minimal configuration struct.

## Architecture & Components

1.  **Configuration Model (`server/internal/config/model.go`)**
    *   A minimal `Config` struct to serve as the foundation.
    *   (Future expansions will add fields like Database, Server, etc.)

2.  **Configuration Loader (`server/internal/config/loader.go`)**
    *   A `Load()` function that initializes a new Viper instance.
    *   **Path Resolution Order (Lowest to Highest Priority):**
        1.  `opus/.opus/` (Development fallback)
        2.  `~/.opus/` (User home directory)
        3.  `OPUS_HOME` (Environment variable override)
    *   **Auto-Creation:** The loader will ensure that the resolved directory (either `OPUS_HOME` or `~/.opus/`) exists before Viper attempts to search for configurations. If it does not exist, it will be created using `os.MkdirAll` (e.g., with `0755` permissions).
    *   **Environment Variables:** Enable Viper's `AutomaticEnv` with the prefix `OPUS`. Environment variables will override any file-based configurations.
    *   Reads `config.json` via Viper.

3.  **Reloadable Interface (`server/internal/config/reloadable.go`)**
    *   Defines the `Reloadable` interface with a `Reload(cfg *Config)` method as specified in ADR-002. (Implementation of the file watcher is deferred to future scopes if not strictly necessary for basic loading, though defining the interface sets the foundation).

## Testing Strategy (Test-Driven Development)

As requested, this implementation will strictly follow TDD principles.

*   **Test Cases (`loader_test.go`):**
    *   **Test Env Override:** Verify that if `OPUS_HOME` is set, it becomes the highest priority path and is created if it doesn't exist.
    *   **Test Auto Creation:** Verify `os.MkdirAll` is called successfully when the target config directory is missing.
    *   **Test Fallbacks:** Verify that Viper checks the correct fallback paths (`~/.opus`, `opus/.opus`) if no explicit override is given.
    *   **Test File Loading:** Verify that if a `config.json` exists in the resolved path, it is parsed correctly into the minimal `Config` struct.

## Next Steps
Once this spec is approved, we will transition to creating a detailed implementation plan using the `writing-plans` skill.