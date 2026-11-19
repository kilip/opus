package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestCopyAgyCredentialsDir(t *testing.T) {
	src := t.TempDir()
	dst := filepath.Join(t.TempDir(), ".gemini")

	// Regular nested credential file
	sub := filepath.Join(src, "antigravity")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "state.json"), []byte(`{"ok":true}`), 0o644); err != nil {
		t.Fatal(err)
	}

	// Simulate Kubernetes Secret ..data symlink structure
	k8sData := filepath.Join(src, "..2026_11_19")
	if err := os.MkdirAll(k8sData, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(k8sData, "creds.json"), []byte(`{"k8s":true}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join("..2026_11_19", "creds.json"), filepath.Join(src, "creds.json")); err != nil {
		t.Fatal(err)
	}

	if err := copyAgyCredentialsDir(src, dst); err != nil {
		t.Fatalf("copyAgyCredentialsDir: %v", err)
	}

	copied, err := os.ReadFile(filepath.Join(dst, "antigravity", "state.json"))
	if err != nil || string(copied) != `{"ok":true}` {
		t.Fatalf("unexpected state.json: %q, err=%v", copied, err)
	}

	k8sCopied, err := os.ReadFile(filepath.Join(dst, "creds.json"))
	if err != nil || string(k8sCopied) != `{"k8s":true}` {
		t.Fatalf("unexpected k8s creds.json: %q, err=%v", k8sCopied, err)
	}

	if _, err := os.Stat(filepath.Join(dst, "..2026_11_19")); !os.IsNotExist(err) {
		t.Fatalf("expected kubernetes .. metadata dir to be skipped, stat err=%v", err)
	}

	if runtime.GOOS != "windows" {
		dirInfo, err := os.Stat(filepath.Join(dst, "antigravity"))
		if err != nil {
			t.Fatal(err)
		}
		if dirInfo.Mode().Perm() != 0o700 {
			t.Fatalf("dir perm = %o, want 0700", dirInfo.Mode().Perm())
		}
		fileInfo, err := os.Stat(filepath.Join(dst, "antigravity", "state.json"))
		if err != nil {
			t.Fatal(err)
		}
		if fileInfo.Mode().Perm() != 0o600 {
			t.Fatalf("file perm = %o, want 0600", fileInfo.Mode().Perm())
		}
	}
}
