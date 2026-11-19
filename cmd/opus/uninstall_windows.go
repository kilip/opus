//go:build windows

package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

func checkBinaryWritable(exePath string) error {
	f, err := os.CreateTemp(filepath.Dir(exePath), ".opus-uninstall-*")
	if err != nil {
		return err
	}
	name := f.Name()
	_ = f.Close()
	_ = os.Remove(name)
	return nil
}

func removeExecutable(exePath string, out io.Writer) error {
	oldPath := exePath + ".old"
	if err := os.Remove(oldPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove stale %s: %w", oldPath, err)
	}
	if err := os.Rename(exePath, oldPath); err != nil {
		return fmt.Errorf("rename %s to %s: %w", exePath, oldPath, err)
	}

	scheduleDeleteOnReboot(oldPath, out)

	installDir := filepath.Dir(exePath)
	if isInstallDirEmptyExcept(installDir, filepath.Base(oldPath)) {
		scheduleDeleteOnReboot(installDir, out)
	}
	return nil
}

func isInstallDirEmptyExcept(dir, allowedName string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if !strings.EqualFold(entry.Name(), allowedName) {
			return false
		}
	}
	return true
}

func scheduleDeleteOnReboot(path string, out io.Writer) {
	ptr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		_, _ = fmt.Fprintf(out, "Note: could not schedule %s for deletion on reboot (%v); delete it manually after exit.\n", path, err)
		return
	}
	if err := windows.MoveFileEx(ptr, nil, windows.MOVEFILE_DELAY_UNTIL_REBOOT); err != nil {
		_, _ = fmt.Fprintf(out, "Note: could not schedule %s for deletion on reboot (%v); delete it manually after exit.\n", path, err)
	}
}

func removeInstallDirFromUserPath(installDir string) (bool, error) {
	if !isManagedInstallDir(installDir) {
		return false, nil
	}

	k, err := registry.OpenKey(registry.CURRENT_USER, `Environment`, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		if errors.Is(err, registry.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	defer func() { _ = k.Close() }()

	val, valType, err := k.GetStringValue("Path")
	if err != nil {
		if errors.Is(err, registry.ErrNotExist) {
			return false, nil
		}
		return false, err
	}

	newPath, changed := removeDirFromPath(val, installDir)
	if !changed {
		return false, nil
	}

	if valType == registry.EXPAND_SZ {
		if err := k.SetExpandStringValue("Path", newPath); err != nil {
			return false, err
		}
		return true, nil
	}
	if err := k.SetStringValue("Path", newPath); err != nil {
		return false, err
	}
	return true, nil
}

func isManagedInstallDir(installDir string) bool {
	norm := strings.TrimRight(strings.TrimSpace(installDir), "\\/")
	if norm == "" {
		return false
	}
	if custom := strings.TrimRight(strings.TrimSpace(os.Getenv("OPUS_INSTALL_DIR")), "\\/"); custom != "" {
		if strings.EqualFold(norm, custom) {
			return true
		}
	}
	if localAppData := strings.TrimSpace(os.Getenv("LOCALAPPDATA")); localAppData != "" {
		defaultDir := strings.TrimRight(filepath.Join(localAppData, "opus", "bin"), "\\/")
		if strings.EqualFold(norm, defaultDir) {
			return true
		}
	}
	return false
}
