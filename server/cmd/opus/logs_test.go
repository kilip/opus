package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLogsCommand(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "logs", "opus.log")

	t.Setenv("OPUS_HOME", tempDir)

	err := os.MkdirAll(filepath.Dir(logPath), 0755)
	if err != nil {
		t.Fatalf("Failed to create log directory: %v", err)
	}

	// 1. Verify response when log file is missing
	cmd, _, err := RootCmd().Find([]string{"logs"})
	if err != nil {
		t.Fatalf("Logs command not found: %v", err)
	}

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	t.Cleanup(func() {
		cmd.SetOut(nil)
	})

	err = cmd.RunE(cmd, []string{})
	if err != nil {
		t.Fatalf("Failed to run logs command: %v", err)
	}

	if !strings.Contains(buf.String(), "No logs found at") {
		t.Errorf("Expected 'No logs found' output, got: %q", buf.String())
	}

	// 2. Write mock logs to the file
	buf.Reset()
	logLines := []string{
		"line 1: init server",
		"line 2: database connected",
		"line 3: starting workers",
		"line 4: server listening on :8080",
	}
	err = os.WriteFile(logPath, []byte(strings.Join(logLines, "\n")+"\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to write mock log: %v", err)
	}

	// 3. Test reading last N lines
	linesCount = 2
	defer func() { linesCount = 100 }()

	err = cmd.RunE(cmd, []string{})
	if err != nil {
		t.Fatalf("Failed to run logs tail command: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "line 3: starting workers") || !strings.Contains(output, "line 4: server listening on :8080") {
		t.Errorf("Expected last 2 lines, got: %q", output)
	}
	if strings.Contains(output, "line 1:") {
		t.Errorf("Should not contain line 1, got: %q", output)
	}

	// 4. Test follow mode (stream logs)
	buf.Reset()
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	cmdFollow, _, err := RootCmd().Find([]string{"logs"})
	if err != nil {
		t.Fatalf("Logs command not found: %v", err)
	}
	followBool = true
	linesCount = 0
	defer func() {
		followBool = false
		linesCount = 100
	}()

	cmdFollow.SetOut(buf)
	cmdFollow.SetContext(ctx)

	done := make(chan error, 1)
	go func() {
		done <- cmdFollow.RunE(cmdFollow, []string{})
	}()

	time.Sleep(50 * time.Millisecond)
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("Failed to open log file for append: %v", err)
	}
	_, _ = f.WriteString("line 5: follow active\n")
	if cerr := f.Close(); cerr != nil {
		t.Fatalf("Failed to close log file: %v", cerr)
	}
	err = <-done
	if err != nil && !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Logf("Follow exit status: %v", err)
	}

	followOutput := buf.String()
	if !strings.Contains(followOutput, "line 5: follow active") {
		t.Errorf("Expected followed output to contain 'line 5: follow active', got: %q", followOutput)
	}
}
