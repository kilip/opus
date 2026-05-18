package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestStopReportingNotRunning(t *testing.T) {
	tempDir := t.TempDir()
	pidFileOverride = filepath.Join(tempDir, "stop.pid")
	t.Cleanup(func() {
		pidFileOverride = ""
	})

	cmd, _, err := RootCmd().Find([]string{"stop"})
	if err != nil {
		t.Fatalf("Stop command not found: %v", err)
	}

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	t.Cleanup(func() {
		cmd.SetOut(nil)
	})

	err = cmd.RunE(cmd, []string{})
	if err != nil {
		t.Fatalf("Failed to run stop command: %v", err)
	}

	if !strings.Contains(buf.String(), "Opus server is not running") {
		t.Errorf("Expected not running response, got: %q", buf.String())
	}
}
