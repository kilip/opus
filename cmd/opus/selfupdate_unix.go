//go:build !windows

package main

import "os"

func replaceBinary(exePath, tmpBinPath string) error {
	return os.Rename(tmpBinPath, exePath)
}

func cleanupOldBinary() {}
