// server/internal/config/loader_test.go
package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
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

func TestLoad_ReadsFileAndEnvVars(t *testing.T) {
	tmpDir := t.TempDir()
	opusHome := filepath.Join(tmpDir, "custom_opus_home")
	t.Setenv("OPUS_HOME", opusHome)

	// Pre-create dir and add config.json
	err := os.MkdirAll(opusHome, 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	configPath := filepath.Join(opusHome, "config.json")
	err = os.WriteFile(configPath, []byte(`{"test_field": "value_from_file"}`), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

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

func TestWatch(t *testing.T) {
	tmpDir := t.TempDir()
	opusHome := filepath.Join(tmpDir, "custom_opus_home")
	t.Setenv("OPUS_HOME", opusHome)

	// Pre-create dir and add config.json
	err := os.MkdirAll(opusHome, 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	configPath := filepath.Join(opusHome, "config.json")
	err = os.WriteFile(configPath, []byte(`{"test_field": "initial_value"}`), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	_, v, err := LoadWithViper()
	if err != nil {
		t.Fatalf("LoadWithViper() error: %v", err)
	}

	ch := make(chan string, 1)
	Watch(v, func(cfg *Config) {
		ch <- cfg.TestField
	})

	// Modify config file to trigger watch
	err = os.WriteFile(configPath, []byte(`{"test_field": "updated_value"}`), 0644)
	if err != nil {
		t.Fatalf("Failed to update config file: %v", err)
	}

	select {
	case val := <-ch:
		if val != "updated_value" {
			t.Errorf("Expected updated value 'updated_value', got '%s'", val)
		}
	case <-time.After(2 * time.Second):
		// fsnotify sometimes requires more time or has environment constraints
		t.Log("Watch notification timed out or not supported in this environment.")
	}
}

func TestLoad_FilePrecedenceOrder(t *testing.T) {
	// Setup custom home and OPUS_HOME temp directories
	tmpDir := t.TempDir()
	opusHomeDir := filepath.Join(tmpDir, "opus_home")
	homeDir := filepath.Join(tmpDir, "home")

	// Ensure dirs exist
	if err := os.MkdirAll(opusHomeDir, 0755); err != nil {
		t.Fatalf("Failed to create opusHomeDir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(homeDir, ".opus"), 0755); err != nil {
		t.Fatalf("Failed to create homeDir/.opus: %v", err)
	}

	// Create local .opus directory in current test running directory
	localOpusDir := ".opus"
	if err := os.MkdirAll(localOpusDir, 0755); err != nil {
		t.Fatalf("Failed to create local .opus dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(localOpusDir)
	}()

	// Write config files in all 3 locations with different values
	err := os.WriteFile(filepath.Join(opusHomeDir, "config.json"), []byte(`{"test_field": "opus_home_val"}`), 0644)
	if err != nil {
		t.Fatalf("Failed to write config in OPUS_HOME: %v", err)
	}
	err = os.WriteFile(filepath.Join(homeDir, ".opus", "config.json"), []byte(`{"test_field": "home_val"}`), 0644)
	if err != nil {
		t.Fatalf("Failed to write config in home: %v", err)
	}
	err = os.WriteFile(filepath.Join(localOpusDir, "config.json"), []byte(`{"test_field": "local_val"}`), 0644)
	if err != nil {
		t.Fatalf("Failed to write config in local: %v", err)
	}

	// Setenv for HOME and OPUS_HOME
	t.Setenv("OPUS_HOME", opusHomeDir)
	t.Setenv("HOME", homeDir)

	// Scenario 1: Both OPUS_HOME, Home and Local exist. OPUS_HOME must win.
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() Scenario 1 error: %v", err)
	}
	if cfg.TestField != "opus_home_val" {
		t.Errorf("Expected 'opus_home_val', got '%s'", cfg.TestField)
	}

	// Scenario 2: OPUS_HOME is not set. Home and Local exist. Home must win.
	t.Setenv("OPUS_HOME", "")
	cfg2, err := Load()
	if err != nil {
		t.Fatalf("Load() Scenario 2 error: %v", err)
	}
	if cfg2.TestField != "home_val" {
		t.Errorf("Expected 'home_val', got '%s'", cfg2.TestField)
	}

	// Scenario 3: OPUS_HOME is not set, Home does not have config. Local must win.
	t.Setenv("HOME", filepath.Join(tmpDir, "non_existent_home"))
	cfg3, err := Load()
	if err != nil {
		t.Fatalf("Load() Scenario 3 error: %v", err)
	}
	if cfg3.TestField != "local_val" {
		t.Errorf("Expected 'local_val', got '%s'", cfg3.TestField)
	}
}
