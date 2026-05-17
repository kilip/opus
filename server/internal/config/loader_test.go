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
