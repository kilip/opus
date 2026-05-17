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
