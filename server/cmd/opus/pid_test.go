package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPIDLifecycle(t *testing.T) {
	tempDir := t.TempDir()
	pidFileOverride = filepath.Join(tempDir, "test.pid")
	t.Cleanup(func() {
		pidFileOverride = ""
	})

	// 1. Initially should fail to read PID
	_, err := readPID()
	if err == nil {
		t.Error("Expected error reading non-existent PID file, got nil")
	}

	// 2. Write current PID
	err = writePID()
	if err != nil {
		t.Fatalf("Failed to write PID file: %v", err)
	}

	// 3. Read back PID and verify match
	pid, err := readPID()
	if err != nil {
		t.Fatalf("Failed to read PID file: %v", err)
	}
	if pid != os.Getpid() {
		t.Errorf("Expected PID %d, got %d", os.Getpid(), pid)
	}

	// 4. Verify process is running
	if !isProcessRunning(pid) {
		t.Error("Expected current process to be reported as running")
	}

	// 5. Remove PID file
	err = removePID()
	if err != nil {
		t.Fatalf("Failed to remove PID file: %v", err)
	}

	// 6. Should fail to read PID after deletion
	_, err = readPID()
	if err == nil {
		t.Error("Expected error reading deleted PID file, got nil")
	}
}
