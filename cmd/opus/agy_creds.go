package main

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const defaultAgyCredentialsDir = "/agy-credentials"

// syncAgyCredentials copies mounted read-only agy credentials (e.g. /agy-credentials)
// into $HOME/.gemini with 0700 directories and 0600 files so containers with
// readOnlyRootFilesystem can authenticate agy non-interactively.
func syncAgyCredentials() {
	srcDir := os.Getenv("AGY_CREDENTIALS_DIR")
	if srcDir == "" {
		srcDir = defaultAgyCredentialsDir
	}
	home := os.Getenv("HOME")
	if home == "" {
		var err error
		home, err = os.UserHomeDir()
		if err != nil || home == "" {
			return
		}
	}
	if err := copyAgyCredentialsDir(srcDir, filepath.Join(home, ".gemini")); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "warning: sync agy credentials from %s: %v\n", srcDir, err)
	}
}

func copyAgyCredentialsDir(srcDir, dstDir string) error {
	info, err := os.Stat(srcDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if !info.IsDir() {
		return nil
	}

	if err := os.MkdirAll(dstDir, 0o700); err != nil {
		return err
	}
	_ = os.Chmod(dstDir, 0o700)

	return filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == srcDir {
			return nil
		}
		// Skip Kubernetes Secret/ConfigMap internal metadata entries (..data, ..YYYY_MM_DD...).
		if strings.HasPrefix(d.Name(), "..") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dstDir, rel)

		// Follow symlinks (e.g. Kubernetes Secret volume mounts) via os.Stat.
		targetInfo, err := os.Stat(path)
		if err != nil {
			return err
		}
		if targetInfo.IsDir() {
			if err := os.MkdirAll(target, 0o700); err != nil {
				return err
			}
			return os.Chmod(target, 0o700)
		}

		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			return err
		}
		return copyFileMode(path, target, 0o600)
	})
}

func copyFileMode(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	if err := out.Chmod(mode); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}
