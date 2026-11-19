package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func buildTestOpusBinary(t *testing.T, dir string) string {
	t.Helper()
	binName := "opus"
	if runtime.GOOS == "windows" {
		binName = "opus.exe"
	}
	binPath := filepath.Join(dir, binName)
	cmd := exec.Command("go", "build", "-o", binPath, ".")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build opus binary: %v\n%s", err, out)
	}
	return binPath
}

func TestUninstallBinary_DefaultKeepsDataDir(t *testing.T) {
	installDir := t.TempDir()
	binPath := buildTestOpusBinary(t, installDir)

	appHome := filepath.Join(t.TempDir(), "opus-data")
	if err := os.MkdirAll(appHome, 0o755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(appHome, "node.json")
	if err := os.WriteFile(marker, []byte(`{"nodeId":"1"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(binPath, "uninstall", "--yes")
	cmd.Env = append(os.Environ(), "APP_HOME="+appHome, "KUBERNETES_SERVICE_HOST=")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("opus uninstall --yes failed: %v\n%s", err, out)
	}

	if _, err := os.Stat(binPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected binary %s to be removed, stat err=%v", binPath, err)
	}
	if runtime.GOOS == "windows" {
		if _, err := os.Stat(binPath + ".old"); err != nil {
			t.Fatalf("expected %s.old to exist on Windows: %v", binPath, err)
		}
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("expected data dir marker %s to be kept: %v", marker, err)
	}
	if !strings.Contains(string(out), appHome) {
		t.Fatalf("expected output to mention kept data dir %s, got:\n%s", appHome, out)
	}
}

func TestUninstallBinary_PurgeRemovesDataDir(t *testing.T) {
	installDir := t.TempDir()
	binPath := buildTestOpusBinary(t, installDir)

	appHome := filepath.Join(t.TempDir(), "opus-data")
	if err := os.MkdirAll(appHome, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(appHome, "opus.db"), []byte("db"), 0o600); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(binPath, "uninstall", "--purge", "--yes")
	cmd.Env = append(os.Environ(), "APP_HOME="+appHome, "KUBERNETES_SERVICE_HOST=")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("opus uninstall --purge --yes failed: %v\n%s", err, out)
	}

	if _, err := os.Stat(binPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected binary %s to be removed, stat err=%v", binPath, err)
	}
	if _, err := os.Stat(appHome); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected data dir %s to be purged, stat err=%v", appHome, err)
	}
}

func TestUninstallBinary_ContainerRefused(t *testing.T) {
	installDir := t.TempDir()
	binPath := buildTestOpusBinary(t, installDir)
	appHome := filepath.Join(t.TempDir(), "opus-data")
	if err := os.MkdirAll(appHome, 0o755); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(binPath, "uninstall", "--purge", "--yes")
	cmd.Env = append(os.Environ(), "APP_HOME="+appHome, "KUBERNETES_SERVICE_HOST=10.0.0.1")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected container uninstall to fail, got success:\n%s", out)
	}
	if !strings.Contains(strings.ToLower(string(out)), "container") || !strings.Contains(strings.ToLower(string(out)), "helm") {
		t.Fatalf("expected container/Helm message, got:\n%s", out)
	}
	if _, err := os.Stat(binPath); err != nil {
		t.Fatalf("expected binary %s to remain untouched: %v", binPath, err)
	}
	if _, err := os.Stat(appHome); err != nil {
		t.Fatalf("expected data dir %s to remain untouched: %v", appHome, err)
	}
}

func TestUninstallBinary_NonWritableDir(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("Unix non-root permission test")
	}

	installDir := t.TempDir()
	binPath := buildTestOpusBinary(t, installDir)
	appHome := filepath.Join(t.TempDir(), "opus-data")
	if err := os.MkdirAll(appHome, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.Chmod(installDir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(installDir, 0o755) })

	cmd := exec.Command(binPath, "uninstall", "--purge", "--yes")
	cmd.Env = append(os.Environ(), "APP_HOME="+appHome, "KUBERNETES_SERVICE_HOST=")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected non-writable install dir to fail, got:\n%s", out)
	}
	if !strings.Contains(string(out), binPath) || !strings.Contains(string(out), "sudo rm") {
		t.Fatalf("expected binary path and sudo rm hint, got:\n%s", out)
	}
	if _, err := os.Stat(binPath); err != nil {
		t.Fatalf("expected binary %s to remain untouched: %v", binPath, err)
	}
	if _, err := os.Stat(appHome); err != nil {
		t.Fatalf("expected data dir %s to remain untouched: %v", appHome, err)
	}
}

func TestUninstallPrompt_CancelAndPurgeConfirmation(t *testing.T) {
	t.Run("cancel on n", func(t *testing.T) {
		var out bytes.Buffer
		removed := false
		deps := testUninstallDeps(&out, "n\n")
		deps.removeBinary = func(string, io.Writer) error {
			removed = true
			return nil
		}

		if err := runUninstall(uninstallOptions{}, deps); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if removed {
			t.Fatal("expected binary not to be removed when user answers n")
		}
		if !strings.Contains(out.String(), "/tmp/bin/opus") {
			t.Fatalf("expected prompt to list binary path, got:\n%s", out.String())
		}
		if !strings.Contains(out.String(), "cancelled") {
			t.Fatalf("expected cancellation note, got:\n%s", out.String())
		}
	})

	t.Run("purge rejects y", func(t *testing.T) {
		var out bytes.Buffer
		removed := false
		deps := testUninstallDeps(&out, "y\n")
		deps.removeBinary = func(string, io.Writer) error {
			removed = true
			return nil
		}

		if err := runUninstall(uninstallOptions{purge: true}, deps); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if removed {
			t.Fatal("expected purge not to proceed on 'y'")
		}
		if !strings.Contains(out.String(), "/tmp/opus-home") {
			t.Fatalf("expected prompt to list data dir, got:\n%s", out.String())
		}
	})

	t.Run("purge proceeds on exact purge", func(t *testing.T) {
		var out bytes.Buffer
		binRemoved := false
		dataRemoved := false
		deps := testUninstallDeps(&out, "purge\n")
		deps.removeBinary = func(string, io.Writer) error {
			binRemoved = true
			return nil
		}
		deps.removeAll = func(string) error {
			dataRemoved = true
			return nil
		}

		if err := runUninstall(uninstallOptions{purge: true}, deps); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !binRemoved || !dataRemoved {
			t.Fatalf("expected both binary and data dir removed, got bin=%v data=%v", binRemoved, dataRemoved)
		}
	})
}

func TestUninstallSafetyGuards(t *testing.T) {
	t.Run("unsafe purge paths refused", func(t *testing.T) {
		unsafePaths := []string{"", "/", "\\", "C:\\", "/home/alice"}
		for _, p := range unsafePaths {
			var out bytes.Buffer
			deps := testUninstallDeps(&out, "")
			deps.homeDir = func() (string, error) { return p, nil }
			deps.userHomeDirs = func() []string { return []string{"/home/alice"} }
			deps.removeBinary = func(string, io.Writer) error {
				t.Fatalf("removeBinary must not be called for unsafe path %q", p)
				return nil
			}

			err := runUninstall(uninstallOptions{purge: true, yes: true}, deps)
			if err == nil {
				t.Fatalf("expected error for unsafe purge path %q", p)
			}
		}
	})

	t.Run("non-opus binary refused", func(t *testing.T) {
		var out bytes.Buffer
		deps := testUninstallDeps(&out, "")
		deps.executable = func() (string, error) { return "/tmp/bin/other-tool", nil }
		deps.evalSymlinks = func(s string) (string, error) { return s, nil }

		err := runUninstall(uninstallOptions{yes: true}, deps)
		if err == nil {
			t.Fatal("expected error for non-opus binary")
		}
	})

	t.Run("dockerenv refused", func(t *testing.T) {
		dockerEnv := filepath.Join(t.TempDir(), ".dockerenv")
		if err := os.WriteFile(dockerEnv, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		var out bytes.Buffer
		deps := testUninstallDeps(&out, "")
		deps.dockerEnvPath = dockerEnv
		deps.stat = os.Stat

		err := runUninstall(uninstallOptions{yes: true}, deps)
		if err == nil || !strings.Contains(strings.ToLower(err.Error()), "container") {
			t.Fatalf("expected container error, got: %v", err)
		}
	})

	t.Run("non-writable binary prints sudo hint without mutation", func(t *testing.T) {
		var out bytes.Buffer
		deps := testUninstallDeps(&out, "")
		deps.checkWritable = func(string) error { return os.ErrPermission }
		deps.removeBinary = func(string, io.Writer) error {
			t.Fatal("removeBinary must not be called when not writable")
			return nil
		}

		err := runUninstall(uninstallOptions{yes: true}, deps)
		if err == nil {
			t.Fatal("expected permission error")
		}
		if !strings.Contains(err.Error(), "/tmp/bin/opus") {
			t.Fatalf("expected binary path in error, got: %v", err)
		}
		if runtime.GOOS != "windows" && !strings.Contains(err.Error(), "sudo rm") {
			t.Fatalf("expected sudo rm hint on Unix, got: %v", err)
		}
	})
}

func TestUninstallExecutionOrder_ServiceBeforeBinary(t *testing.T) {
	var out bytes.Buffer
	var order []string
	deps := testUninstallDeps(&out, "")
	deps.hasServiceCmd = func() bool { return true }
	deps.stopAndRemoveService = func() error {
		order = append(order, "service")
		return nil
	}
	deps.removeBinary = func(string, io.Writer) error {
		order = append(order, "binary")
		return nil
	}
	deps.removeAll = func(string) error {
		order = append(order, "purge")
		return nil
	}

	if err := runUninstall(uninstallOptions{purge: true, yes: true}, deps); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "service,binary,purge"
	if got := strings.Join(order, ","); got != want {
		t.Fatalf("execution order = %q, want %q", got, want)
	}
}

func TestRemoveDirFromPath(t *testing.T) {
	cases := []struct {
		name      string
		pathVal   string
		target    string
		wantPath  string
		wantFound bool
	}{
		{
			name:      "removes matching entry case-insensitively with trailing slash",
			pathVal:   `C:\Windows\System32;c:\users\dev\AppData\Local\opus\bin\;C:\Tools`,
			target:    `C:\Users\dev\AppData\Local\opus\bin`,
			wantPath:  `C:\Windows\System32;C:\Tools`,
			wantFound: true,
		},
		{
			name:      "removes duplicate occurrences",
			pathVal:   `C:\opus\bin;C:\Other;C:\opus\bin\`,
			target:    `C:\opus\bin`,
			wantPath:  `C:\Other`,
			wantFound: true,
		},
		{
			name:      "leaves unrelated entries untouched when absent",
			pathVal:   `C:\Windows\System32;C:\Tools`,
			target:    `C:\Users\dev\AppData\Local\opus\bin`,
			wantPath:  `C:\Windows\System32;C:\Tools`,
			wantFound: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotPath, gotFound := removeDirFromPath(tc.pathVal, tc.target)
			if gotFound != tc.wantFound || gotPath != tc.wantPath {
				t.Fatalf("removeDirFromPath(%q, %q) = (%q, %v), want (%q, %v)",
					tc.pathVal, tc.target, gotPath, gotFound, tc.wantPath, tc.wantFound)
			}
		})
	}
}

func testUninstallDeps(out *bytes.Buffer, stdin string) uninstallDeps {
	return uninstallDeps{
		in:           strings.NewReader(stdin),
		out:          out,
		executable:   func() (string, error) { return "/tmp/bin/opus", nil },
		evalSymlinks: func(s string) (string, error) { return s, nil },
		homeDir:      func() (string, error) { return "/tmp/opus-home", nil },
		userHomeDirs: func() []string { return []string{"/home/testuser"} },
		getenv:       func(string) string { return "" },
		stat:         func(string) (os.FileInfo, error) { return nil, os.ErrNotExist },
		checkWritable: func(string) error {
			return nil
		},
		removeBinary: func(string, io.Writer) error {
			return nil
		},
		removeWindowsPath: func(string) (bool, error) {
			return false, nil
		},
		removeAll: func(string) error {
			return nil
		},
		hasServiceCmd: func() bool {
			return false
		},
		dockerEnvPath: "",
	}
}
