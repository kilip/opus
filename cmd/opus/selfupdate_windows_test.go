//go:build windows

package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestSelfUpdate_WindowsWhileRunningAndOldCleanup(t *testing.T) {
	newBinDir := t.TempDir()
	newBinPath := buildTestOpusBinaryWithVersion(t, newBinDir, "0.2.0")
	archiveData := makeTestArchive(t, newBinPath, "windows")

	srv := newFakeReleaseServer(t, []fakeReleaseSpec{
		{tag: "v0.2.0", archiveData: archiveData},
	}, runtime.GOOS, runtime.GOARCH)

	installDir := t.TempDir()
	installedBin := buildTestOpusBinaryWithVersion(t, installDir, "0.1.0")

	// AC4: opus.exe self-update --yes replaces itself while running, leaving opus.exe.old.
	cmd := exec.Command(installedBin, "self-update", "--yes")
	cmd.Env = append(os.Environ(),
		"TEST_SELF_UPDATE_API_URL="+srv.URL,
		"APP_HOME="+filepath.Join(installDir, "nonexistent-home"),
		"APP_DATABASE_URL=",
		"KUBERNETES_SERVICE_HOST=",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("windows opus.exe self-update --yes failed: %v\n%s", err, out)
	}

	oldPath := installedBin + ".old"
	if _, err := os.Stat(oldPath); err != nil {
		t.Fatalf("expected %s to exist immediately after Windows self-update: %v", oldPath, err)
	}

	// AC4: Next start of opus.exe prints 0.2.0 and cleans up opus.exe.old.
	verOut, err := exec.Command(installedBin, "--version").CombinedOutput()
	if err != nil {
		t.Fatalf("updated opus.exe --version failed: %v\n%s", err, verOut)
	}
	if !strings.Contains(string(verOut), "0.2.0") {
		t.Fatalf("expected updated opus.exe to print 0.2.0, got:\n%s", verOut)
	}
	if _, err := os.Stat(oldPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected %s to be cleaned up on next startup, stat err=%v", oldPath, err)
	}
}
