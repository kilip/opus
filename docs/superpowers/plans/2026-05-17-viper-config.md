# Viper Config Integration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [x]`) syntax for tracking.

**Goal:** Implement the foundational configuration management system for Opus Server using Viper, focusing on path resolution and directory auto-creation as per ADR-002.

**Architecture:** We will create a minimal `Config` struct, a `Reloadable` interface, and a `loader.go` that initializes Viper. The loader logic ensures configuration directories (either `OPUS_HOME` or fallback `~/.opus/`) are created if missing before Viper reads the configuration. 

**Tech Stack:** Go 1.26, Viper, Test-Driven Development (testing standard library).

---

### Task 1: Setup Models and Interface

**Files:**
- Create: `server/internal/config/model.go`
- Create: `server/internal/config/reloadable.go`

- [x] **Step 1: Write minimal struct and interface**

```go
// server/internal/config/model.go
package config

// Config is the top-level configuration structure.
// Future fields (Server, Database, etc.) will be added here.
type Config struct {
	// Minimal struct for now
}
```

```go
// server/internal/config/reloadable.go
package config

// Reloadable defines an interface for services that need to react to config changes.
type Reloadable interface {
	Reload(cfg *Config)
}
```

- [x] **Step 2: Check compilation**

Run: `cd server && go build ./internal/config/...`
Expected: Passes without errors.

- [x] **Step 3: Commit**

```bash
cd server && git add internal/config/model.go internal/config/reloadable.go
git commit -m "feat: add base config model and reloadable interface"
```

### Task 2: Write tests for Path Resolution & Auto-Create

**Files:**
- Create: `server/internal/config/loader_test.go`

- [x] **Step 1: Write failing test**

```go
// server/internal/config/loader_test.go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_AutoCreateAndResolveDir(t *testing.T) {
	// Setup temporary directory for test isolation
	tmpDir := t.TempDir()
	opusHome := filepath.Join(tmpDir, "custom_opus_home")
	
	// Set OPUS_HOME override
	t.Setenv("OPUS_HOME", opusHome)

	_, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	// Verify the directory was created
	info, err := os.Stat(opusHome)
	if err != nil {
		t.Fatalf("Directory was not created: %v", err)
	}
	if !info.IsDir() {
		t.Errorf("Expected path %s to be a directory", opusHome)
	}
}
```

- [x] **Step 2: Run test to verify it fails**

Run: `cd server && go test ./internal/config/... -v`
Expected: FAIL with "undefined: Load"

- [x] **Step 3: Write minimal placeholder loader**

**Files:**
- Create: `server/internal/config/loader.go`

```go
// server/internal/config/loader.go
package config

func Load() (*Config, error) {
	return &Config{}, nil
}
```

- [x] **Step 4: Run test to verify it fails on logic**

Run: `cd server && go test ./internal/config/... -v`
Expected: FAIL with "Directory was not created"

- [x] **Step 5: Commit**

```bash
cd server && git add internal/config/loader_test.go internal/config/loader.go
git commit -m "test: add failing test for config dir auto-creation"
```

### Task 3: Implement Directory Resolution and Auto-Create

**Files:**
- Modify: `server/internal/config/loader.go`

- [x] **Step 1: Write minimal implementation to pass the test**

```go
// server/internal/config/loader.go
package config

import (
	"os"
	"path/filepath"
)

func resolveConfigDir() string {
	if opusHome := os.Getenv("OPUS_HOME"); opusHome != "" {
		return opusHome
	}
	// Fallback to user home
	home, err := os.UserHomeDir()
	if err != nil {
		return "." // fallback if home cannot be determined
	}
	return filepath.Join(home, ".opus")
}

func Load() (*Config, error) {
	configDir := resolveConfigDir()

	// Auto-create directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, err
	}

	return &Config{}, nil
}
```

- [x] **Step 2: Run test to verify it passes**

Run: `cd server && go test ./internal/config/... -v`
Expected: PASS

- [x] **Step 3: Commit**

```bash
cd server && git add internal/config/loader.go
git commit -m "feat: implement config directory auto-creation"
```

### Task 4: Write Tests for Viper Config Loading

**Files:**
- Modify: `server/internal/config/loader_test.go`
- Modify: `server/internal/config/model.go` (add a temporary field to test loading)

- [x] **Step 1: Add a test field to Config model**

Replace `server/internal/config/model.go` content with:
```go
// server/internal/config/model.go
package config

// Config is the top-level configuration structure.
type Config struct {
	TestField string `mapstructure:"test_field" json:"test_field"`
}
```

- [x] **Step 2: Write test for Viper file loading and env override**

Append to `server/internal/config/loader_test.go`:
```go
func TestLoad_ReadsFileAndEnvVars(t *testing.T) {
	tmpDir := t.TempDir()
	opusHome := filepath.Join(tmpDir, "custom_opus_home")
	t.Setenv("OPUS_HOME", opusHome)

	// Pre-create dir and add config.json
	os.MkdirAll(opusHome, 0755)
	configPath := filepath.Join(opusHome, "config.json")
	os.WriteFile(configPath, []byte(`{"test_field": "value_from_file"}`), 0644)

	// Test 1: Load from file
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.TestField != "value_from_file" {
		t.Errorf("Expected test_field='value_from_file', got '%s'", cfg.TestField)
	}

	// Test 2: Env Var override
	t.Setenv("OPUS_TEST_FIELD", "value_from_env")
	cfg2, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg2.TestField != "value_from_env" {
		t.Errorf("Expected test_field='value_from_env', got '%s'", cfg2.TestField)
	}
}
```

- [x] **Step 3: Run test to verify it fails**

Run: `cd server && go test ./internal/config/... -v`
Expected: FAIL on `TestLoad_ReadsFileAndEnvVars` because `Load()` returns an empty struct.

- [x] **Step 4: Commit**

```bash
cd server && git add internal/config/loader_test.go internal/config/model.go
git commit -m "test: add failing tests for viper config loading and env override"
```

### Task 5: Implement Viper Loading Logic

**Files:**
- Modify: `server/internal/config/loader.go`

- [x] **Step 1: Implement full Viper loading**

Replace `server/internal/config/loader.go` content with:
```go
// server/internal/config/loader.go
package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

func resolveConfigDir() string {
	if opusHome := os.Getenv("OPUS_HOME"); opusHome != "" {
		return opusHome
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return filepath.Join(home, ".opus")
}

func Load() (*Config, error) {
	configDir := resolveConfigDir()

	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, err
	}

	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("json")

	// Resolution order: lowest to highest priority
	v.AddConfigPath(filepath.Join("opus", ".opus")) // development
	home, _ := os.UserHomeDir()
	if home != "" {
		v.AddConfigPath(filepath.Join(home, ".opus")) // user home
	}

	// Explicit override via env var
	if opusHome := os.Getenv("OPUS_HOME"); opusHome != "" {
		v.AddConfigPath(opusHome)
	}

	v.SetEnvPrefix("OPUS")
	v.AutomaticEnv()

	// It's okay if config file doesn't exist, we might just use env vars
	_ = v.ReadInConfig()

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
```

- [x] **Step 2: Run tests to verify they pass**

Run: `cd server && go test ./internal/config/... -v`
Expected: PASS for all tests.

- [x] **Step 3: Tidy and Commit**

```bash
cd server
go mod tidy
git add internal/config/loader.go go.mod go.sum
git commit -m "feat: implement viper loading with path resolution and fallbacks"
```
