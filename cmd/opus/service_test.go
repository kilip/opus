package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestService_EnvFileCreationAndNoOverwrite(t *testing.T) {
	appHome := t.TempDir()

	envPath, created, err := ensureOpusEnvFile(appHome, serviceRoleServer)
	if err != nil {
		t.Fatalf("ensureOpusEnvFile: %v", err)
	}
	if !created {
		t.Fatal("expected created=true on first call")
	}

	info, err := os.Stat(envPath)
	if err != nil {
		t.Fatalf("stat %s: %v", envPath, err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("env file perm = %o, want 0600", info.Mode().Perm())
	}

	raw, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "AUTH_SECRET=") {
		t.Fatalf("expected AUTH_SECRET in %s, got:\n%s", envPath, raw)
	}
	for _, line := range strings.Split(string(raw), "\n") {
		if after, ok := strings.CutPrefix(line, "AUTH_SECRET="); ok {
			if len(strings.TrimSpace(after)) < 32 {
				t.Fatalf("expected AUTH_SECRET >= 32 chars, got %d", len(strings.TrimSpace(after)))
			}
		}
	}

	// Customise opus.env and verify second call does NOT overwrite it.
	const customLine = "\nCUSTOM_KEY=keep-me\n"
	if err := os.WriteFile(envPath, append(raw, customLine...), 0o600); err != nil {
		t.Fatal(err)
	}
	_, created2, err := ensureOpusEnvFile(appHome, serviceRoleServer)
	if err != nil {
		t.Fatal(err)
	}
	if created2 {
		t.Fatal("expected created=false when opus.env already exists")
	}
	afterRaw, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(afterRaw), "CUSTOM_KEY=keep-me") {
		t.Fatalf("expected existing opus.env to be preserved, got:\n%s", afterRaw)
	}
}

func TestService_RenderedUnitFilesExcludeSecrets(t *testing.T) {
	const secret = "super-secret-auth-value-0123456789"
	envPath := "/tmp/opus/opus.env"

	unit := renderSystemdUnit("/usr/local/bin/opus", serviceRoleServer, envPath, false)
	if strings.Contains(unit, secret) || !strings.Contains(unit, "EnvironmentFile="+envPath) {
		t.Fatalf("unexpected systemd unit:\n%s", unit)
	}
	if !strings.Contains(unit, "Restart=on-failure") || !strings.Contains(unit, "RestartSec=2") {
		t.Fatalf("expected Restart=on-failure and RestartSec=2 in systemd unit:\n%s", unit)
	}
	if !strings.Contains(unit, "ExecStart=/usr/local/bin/opus serve") {
		t.Fatalf("expected ExecStart serve in systemd unit:\n%s", unit)
	}

	workerUnit := renderSystemdUnit("/usr/local/bin/opus", serviceRoleWorker, envPath, true)
	if !strings.Contains(workerUnit, "ExecStart=/usr/local/bin/opus worker") || !strings.Contains(workerUnit, "WantedBy=multi-user.target") {
		t.Fatalf("unexpected system worker unit:\n%s", workerUnit)
	}

	plist := renderLaunchdPlist("com.kilip.opus", "/usr/local/bin/opus", serviceRoleServer, envPath, "/tmp/opus.log")
	if strings.Contains(plist, secret) || !strings.Contains(plist, "<string>--env-file</string>") {
		t.Fatalf("unexpected launchd plist:\n%s", plist)
	}

	winXML := renderWindowsTaskXML(`C:\opus\opus.exe`, serviceRoleServer, envPath, `C:\opus\opus.log`)
	if strings.Contains(winXML, secret) || !strings.Contains(winXML, "<RestartOnFailure>") || !strings.Contains(winXML, "<ExecutionTimeLimit>PT0S</ExecutionTimeLimit>") {
		t.Fatalf("unexpected windows task XML:\n%s", winXML)
	}
}

func TestService_WorkerRequiresNodeJSON(t *testing.T) {
	appHome := t.TempDir()
	deps := serviceDeps{
		out:           &bytes.Buffer{},
		homeDir:       func() (string, error) { return appHome, nil },
		userConfigDir: func() (string, error) { return t.TempDir(), nil },
		stat:          os.Stat,
		executable:    func() (string, error) { return "/usr/local/bin/opus", nil },
		runCmd:        func(context.Context, string, ...string) ([]byte, error) { return nil, nil },
	}

	// AC5: install --role worker fails if node.json is missing.
	err := runServiceInstall(context.Background(), serviceOptions{role: serviceRoleWorker}, deps)
	if err == nil || !strings.Contains(err.Error(), "opus join") {
		t.Fatalf("expected opus join error on install without node.json, got: %v", err)
	}

	// AC5: start --role worker fails if node.json is missing.
	err = runServiceStart(context.Background(), serviceOptions{role: serviceRoleWorker}, deps)
	if err == nil || !strings.Contains(err.Error(), "opus join") {
		t.Fatalf("expected opus join error on start without node.json, got: %v", err)
	}
}

func TestService_LifecycleAndStatusExitCodes(t *testing.T) {
	appHome := t.TempDir()
	cfgDir := t.TempDir()
	var cmds [][]string
	active := false

	deps := serviceDeps{
		out:           &bytes.Buffer{},
		executable:    func() (string, error) { return "/usr/local/bin/opus", nil },
		evalSymlinks:  func(p string) (string, error) { return p, nil },
		homeDir:       func() (string, error) { return appHome, nil },
		userConfigDir: func() (string, error) { return cfgDir, nil },
		stat:          os.Stat,
		runCmd: func(_ context.Context, name string, args ...string) ([]byte, error) {
			cmds = append(cmds, append([]string{name}, args...))
			if len(args) >= 2 && args[1] == "is-active" {
				if active {
					return []byte("active\n"), nil
				}
				return []byte("inactive\n"), errors.New("exit status 3")
			}
			return nil, nil
		},
	}

	opts := serviceOptions{role: serviceRoleServer}

	// Not installed -> exit code 4
	code, err := runServiceStatus(context.Background(), opts, deps)
	if err != nil || code != serviceStatusNotInstalled {
		t.Fatalf("status before install = (%d, %v), want (4, nil)", code, err)
	}

	// Install creates unit and opus.env
	if err := runServiceInstall(context.Background(), opts, deps); err != nil {
		t.Fatalf("install: %v", err)
	}
	unitPath := filepath.Join(cfgDir, "systemd", "user", "opus.service")
	if _, err := os.Stat(unitPath); err != nil {
		t.Fatalf("expected unit file %s: %v", unitPath, err)
	}
	envPath := filepath.Join(appHome, "opus.env")
	if _, err := os.Stat(envPath); err != nil {
		t.Fatalf("expected env file %s: %v", envPath, err)
	}

	// Installed but stopped -> exit code 3
	code, err = runServiceStatus(context.Background(), opts, deps)
	if err != nil || code != serviceStatusStopped {
		t.Fatalf("status when stopped = (%d, %v), want (3, nil)", code, err)
	}

	// Start -> running -> exit code 0
	if err := runServiceStart(context.Background(), opts, deps); err != nil {
		t.Fatalf("start: %v", err)
	}
	active = true
	code, err = runServiceStatus(context.Background(), opts, deps)
	if err != nil || code != serviceStatusRunning {
		t.Fatalf("status when running = (%d, %v), want (0, nil)", code, err)
	}

	// Restart & Stop
	if err := runServiceRestart(context.Background(), opts, deps); err != nil {
		t.Fatalf("restart: %v", err)
	}
	if err := runServiceStop(context.Background(), opts, deps); err != nil {
		t.Fatalf("stop: %v", err)
	}

	// AC3: Uninstall removes unit file, keeps opus.env
	if err := runServiceUninstall(context.Background(), opts, deps); err != nil {
		t.Fatalf("uninstall: %v", err)
	}
	if _, err := os.Stat(unitPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected unit file removed, stat err=%v", err)
	}
	if _, err := os.Stat(envPath); err != nil {
		t.Fatalf("expected opus.env to be kept after uninstall: %v", err)
	}
	if len(cmds) == 0 {
		t.Fatal("expected systemctl commands to be invoked")
	}
}
