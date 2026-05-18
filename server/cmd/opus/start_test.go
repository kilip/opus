// White-box testing is used to access package-level unexported variables like rootCmd.
package main

import (
	"path/filepath"
	"testing"
)

func TestStartCommandRegistered(t *testing.T) {
	cmd, _, err := RootCmd().Find([]string{"start"})
	if err != nil {
		t.Fatalf("Start command not found: %v", err)
	}
	if cmd.Use != "start" {
		t.Errorf("Expected command use 'start', got %s", cmd.Use)
	}
}

func TestRootDefaultRun(t *testing.T) {
	root := RootCmd()
	if root.RunE == nil {
		t.Error("Expected rootCmd to have a default RunE function for backward compatibility")
	}
}

func TestCommandsRegistered(t *testing.T) {
	cmds := []string{"stop", "status", "logs", "restart"}
	for _, name := range cmds {
		cmd, _, err := RootCmd().Find([]string{name})
		if err != nil {
			t.Fatalf("Command '%s' not found: %v", name, err)
		}
		if cmd.Use != name {
			t.Errorf("Expected command use '%s', got %s", name, cmd.Use)
		}
	}
}

func TestStartDoubleRunPrevention(t *testing.T) {
	tempDir := t.TempDir()
	pidFileOverride = filepath.Join(tempDir, "start.pid")
	t.Cleanup(func() {
		pidFileOverride = ""
	})

	// Write a mock active process PID (ourselves)
	err := writePID()
	if err != nil {
		t.Fatalf("Failed to write mock PID: %v", err)
	}

	// Attempting to run start command should fail immediately
	cmd, _, err := RootCmd().Find([]string{"start"})
	if err != nil {
		t.Fatalf("Start command not found: %v", err)
	}

	err = cmd.RunE(cmd, []string{})
	if err == nil {
		t.Error("Expected error when starting server while PID file claims it is already running, got nil")
	}
}
