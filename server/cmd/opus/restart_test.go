package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestRestartCommandNotRunning(t *testing.T) {
	tempDir := t.TempDir()
	pidFileOverride = filepath.Join(tempDir, "restart.pid")
	t.Setenv("OPUS_HOME", tempDir)
	t.Cleanup(func() {
		pidFileOverride = ""
	})

	cmd, _, err := RootCmd().Find([]string{"restart"})
	if err != nil {
		t.Fatalf("Restart command not found: %v", err)
	}

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	t.Cleanup(func() {
		cmd.SetOut(nil)
	})

	// Set daemonBool to true so that it attempts background spawn and exits immediately instead of hanging
	daemonBool = true
	t.Cleanup(func() {
		daemonBool = false
	})

	err = cmd.RunE(cmd, []string{})
	if err != nil && !strings.Contains(err.Error(), "background process started but exited immediately") {
		t.Fatalf("Unexpected error from restart run: %v", err)
	}

	if !strings.Contains(buf.String(), "Opus server is not running. Starting fresh...") {
		t.Errorf("Expected 'starting fresh' output, got: %q", buf.String())
	}
}
