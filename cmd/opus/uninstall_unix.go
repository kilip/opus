//go:build !windows

package main

import (
	"io"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

func checkBinaryWritable(exePath string) error {
	return unix.Access(filepath.Dir(exePath), unix.W_OK|unix.X_OK)
}

func removeExecutable(exePath string, _ io.Writer) error {
	return os.Remove(exePath)
}

func removeInstallDirFromUserPath(_ string) (bool, error) {
	return false, nil
}
