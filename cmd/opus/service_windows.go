//go:build windows

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func windowsTaskName(role string) string {
	if role == serviceRoleWorker {
		return "OpusWorker"
	}
	return "Opus"
}

func windowsLogPath(role string, deps serviceDeps) string {
	localAppData := deps.getenv("LOCALAPPDATA")
	if localAppData == "" {
		localAppData = os.TempDir()
	}
	name := "opus.log"
	if role == serviceRoleWorker {
		name = "opus-worker.log"
	}
	return filepath.Join(localAppData, "opus", name)
}

func runServiceInstall(ctx context.Context, opts serviceOptions, deps serviceDeps) error {
	if err := validateServiceRole(opts.role); err != nil {
		return err
	}
	appHome, err := deps.homeDir()
	if err != nil {
		return err
	}
	if opts.role == serviceRoleWorker {
		if err := checkWorkerNodeConfig(appHome, deps.stat); err != nil {
			return err
		}
	}
	exePath, err := resolveServiceExePath(deps)
	if err != nil {
		return err
	}
	envPath, created, err := ensureOpusEnvFile(appHome, opts.role)
	if err != nil {
		return err
	}

	logPath := windowsLogPath(opts.role, deps)
	taskXML := renderWindowsTaskXML(exePath, opts.role, appHome, envPath, logPath)

	tmpFile, err := os.CreateTemp("", "opus-task-*.xml")
	if err != nil {
		return fmt.Errorf("create temp task xml: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	if _, err := tmpFile.WriteString(taskXML); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("write task xml: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close task xml: %w", err)
	}

	taskName := windowsTaskName(opts.role)
	if out, err := deps.runCmd(ctx, "schtasks", "/Create", "/TN", taskName, "/XML", tmpPath, "/F"); err != nil {
		return fmt.Errorf("schtasks /Create %s: %w (%s)", taskName, err, string(out))
	}

	_, _ = fmt.Fprintf(deps.out, "Installed scheduled task: %s\n", taskName)
	if created {
		_, _ = fmt.Fprintf(deps.out, "Created environment file: %s\n", envPath)
	} else {
		_, _ = fmt.Fprintf(deps.out, "Kept existing environment file: %s\n", envPath)
	}
	return nil
}

func runServiceUninstall(ctx context.Context, opts serviceOptions, deps serviceDeps) error {
	if err := validateServiceRole(opts.role); err != nil {
		return err
	}
	taskName := windowsTaskName(opts.role)
	_, _ = deps.runCmd(ctx, "schtasks", "/End", "/TN", taskName)
	if out, err := deps.runCmd(ctx, "schtasks", "/Delete", "/TN", taskName, "/F"); err != nil {
		return fmt.Errorf("schtasks /Delete %s: %w (%s)", taskName, err, string(out))
	}
	_, _ = fmt.Fprintf(deps.out, "Uninstalled scheduled task: %s\n", taskName)
	return nil
}

func runServiceStart(ctx context.Context, opts serviceOptions, deps serviceDeps) error {
	if err := validateServiceRole(opts.role); err != nil {
		return err
	}
	if opts.role == serviceRoleWorker {
		appHome, err := deps.homeDir()
		if err != nil {
			return err
		}
		if err := checkWorkerNodeConfig(appHome, deps.stat); err != nil {
			return err
		}
	}
	taskName := windowsTaskName(opts.role)
	if out, err := deps.runCmd(ctx, "schtasks", "/Run", "/TN", taskName); err != nil {
		return fmt.Errorf("schtasks /Run %s: %w (%s)", taskName, err, string(out))
	}
	_, _ = fmt.Fprintf(deps.out, "Started %s.\n", taskName)
	return nil
}

func runServiceStop(ctx context.Context, opts serviceOptions, deps serviceDeps) error {
	if err := validateServiceRole(opts.role); err != nil {
		return err
	}
	taskName := windowsTaskName(opts.role)
	if out, err := deps.runCmd(ctx, "schtasks", "/End", "/TN", taskName); err != nil {
		return fmt.Errorf("schtasks /End %s: %w (%s)", taskName, err, string(out))
	}
	_, _ = fmt.Fprintf(deps.out, "Stopped %s.\n", taskName)
	return nil
}

func runServiceRestart(ctx context.Context, opts serviceOptions, deps serviceDeps) error {
	if err := validateServiceRole(opts.role); err != nil {
		return err
	}
	taskName := windowsTaskName(opts.role)
	_, _ = deps.runCmd(ctx, "schtasks", "/End", "/TN", taskName)
	return runServiceStart(ctx, opts, deps)
}

func runServiceStatus(ctx context.Context, opts serviceOptions, deps serviceDeps) (int, error) {
	if err := validateServiceRole(opts.role); err != nil {
		return 0, err
	}
	taskName := windowsTaskName(opts.role)
	out, err := deps.runCmd(ctx, "schtasks", "/Query", "/TN", taskName, "/FO", "LIST")
	if err != nil {
		_, _ = fmt.Fprintln(deps.out, "Status: not installed")
		return serviceStatusNotInstalled, nil
	}
	if strings.Contains(string(out), "Running") {
		_, _ = fmt.Fprintln(deps.out, "Status: running")
		return serviceStatusRunning, nil
	}
	_, _ = fmt.Fprintln(deps.out, "Status: stopped")
	return serviceStatusStopped, nil
}
