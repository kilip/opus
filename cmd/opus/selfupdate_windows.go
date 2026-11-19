//go:build windows

package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func replaceBinary(exePath, tmpBinPath string) error {
	oldPath := exePath + ".old"
	if err := os.Remove(oldPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove stale %s: %w", oldPath, err)
	}
	if err := os.Rename(exePath, oldPath); err != nil {
		return fmt.Errorf("rename %s to %s: %w", exePath, oldPath, err)
	}
	if err := os.Rename(tmpBinPath, exePath); err != nil {
		_ = os.Rename(oldPath, exePath)
		return fmt.Errorf("replace %s: %w", exePath, err)
	}
	return nil
}

func cleanupOldBinary() {
	rawExe, err := os.Executable()
	if err != nil {
		return
	}
	exePath, err := filepath.EvalSymlinks(rawExe)
	if err != nil {
		exePath = rawExe
	}
	_ = os.Remove(exePath + ".old")
}
