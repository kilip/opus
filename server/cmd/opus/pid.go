package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

var pidFileOverride string

// resolveConfigDir returns the path to the active Opus configuration directory.
// This matches the resolution order used by the Viper configuration loader in internal/config/loader.go.
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

// getPIDFilePath returns the path to the active opus.pid file.
func getPIDFilePath() string {
	if pidFileOverride != "" {
		return pidFileOverride
	}
	return filepath.Join(resolveConfigDir(), "opus.pid")
}

// getLogFilePath returns the path to the active logs/opus.log file.
func getLogFilePath() string {
	return filepath.Join(resolveConfigDir(), "logs", "opus.log")
}

// readPID reads the process ID from the PID file.
func readPID() (int, error) {
	path := getPIDFilePath()
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	pidStr := strings.TrimSpace(string(data))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return 0, fmt.Errorf("invalid PID in file: %w", err)
	}
	return pid, nil
}

// isProcessRunning checks if a process with the given PID is currently active in the OS.
func isProcessRunning(pid int) bool {
	if pid <= 0 {
		return false
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// On Unix/Linux, FindProcess always succeeds. We send signal 0 to check if the process is alive.
	err = process.Signal(syscall.Signal(0))
	if err == nil {
		return true
	}
	// If permission is denied, the process exists but is owned by another user (e.g. root), meaning it is running.
	if errors.Is(err, syscall.EPERM) {
		return true
	}
	return false
}

// writePID writes the current process ID to the PID file.
func writePID() error {
	path := getPIDFilePath()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create PID directory: %w", err)
	}
	return os.WriteFile(path, []byte(strconv.Itoa(os.Getpid())), 0644)
}

// removePID removes the PID file from the filesystem.
func removePID() error {
	path := getPIDFilePath()
	err := os.Remove(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
