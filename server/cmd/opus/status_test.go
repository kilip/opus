package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestStatusReporting(t *testing.T) {
	tempDir := t.TempDir()
	pidFileOverride = filepath.Join(tempDir, "status.pid")
	t.Cleanup(func() {
		pidFileOverride = ""
	})

	// 1. Status when not running
	cmd, _, err := RootCmd().Find([]string{"status"})
	if err != nil {
		t.Fatalf("Status command not found: %v", err)
	}

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	t.Cleanup(func() {
		cmd.SetOut(nil)
	})

	err = cmd.RunE(cmd, []string{})
	if err != nil {
		t.Fatalf("Failed to run status command: %v", err)
	}

	if !strings.Contains(buf.String(), "Opus server is not running") {
		t.Errorf("Expected 'not running' message, got: %q", buf.String())
	}

	// 2. Status when running (mock running PID)
	buf.Reset()
	err = writePID()
	if err != nil {
		t.Fatalf("Failed to write mock PID: %v", err)
	}

	err = cmd.RunE(cmd, []string{})
	if err != nil {
		t.Fatalf("Failed to run status command: %v", err)
	}

	if !strings.Contains(buf.String(), "Opus server is running with PID") {
		t.Errorf("Expected 'running' message, got: %q", buf.String())
	}
}
