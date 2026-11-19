//go:build darwin

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func launchdLabel(role string) string {
	if role == serviceRoleWorker {
		return "com.kilip.opus.worker"
	}
	return "com.kilip.opus"
}

func launchdPlistPath(opts serviceOptions, deps serviceDeps) (string, error) {
	label := launchdLabel(opts.role) + ".plist"
	if opts.system {
		return filepath.Join("/Library/LaunchDaemons", label), nil
	}
	home, err := deps.userHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home dir: %w", err)
	}
	return filepath.Join(home, "Library", "LaunchAgents", label), nil
}

func launchdLogPath(opts serviceOptions, deps serviceDeps) (string, error) {
	name := "opus.log"
	if opts.role == serviceRoleWorker {
		name = "opus-worker.log"
	}
	if opts.system {
		return filepath.Join("/Library/Logs", name), nil
	}
	home, err := deps.userHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home dir: %w", err)
	}
	return filepath.Join(home, "Library", "Logs", name), nil
}

func launchdDomain(opts serviceOptions, deps serviceDeps) string {
	if opts.system {
		return "system"
	}
	return fmt.Sprintf("gui/%d", deps.uid())
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
	plistPath, err := launchdPlistPath(opts, deps)
	if err != nil {
		return err
	}
	logPath, err := launchdLogPath(opts, deps)
	if err != nil {
		return err
	}

	envPath, created, err := ensureOpusEnvFile(appHome, opts.role)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(plistPath), 0o755); err != nil {
		return fmt.Errorf("create LaunchAgents dir: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		return fmt.Errorf("create Logs dir: %w", err)
	}

	label := launchdLabel(opts.role)
	plist := renderLaunchdPlist(label, exePath, opts.role, appHome, envPath, logPath)
	domain := launchdDomain(opts, deps)
	_, _ = deps.runCmd(ctx, "launchctl", "bootout", domain+"/"+label)
	if err := os.WriteFile(plistPath, []byte(plist), 0o644); err != nil {
		return fmt.Errorf("write plist %s: %w", plistPath, err)
	}

	_, _ = fmt.Fprintf(deps.out, "Installed launchd plist: %s\n", plistPath)
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
	plistPath, err := launchdPlistPath(opts, deps)
	if err != nil {
		return err
	}
	domain := launchdDomain(opts, deps)
	label := launchdLabel(opts.role)
	_, _ = deps.runCmd(ctx, "launchctl", "bootout", domain+"/"+label)

	if err := os.Remove(plistPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove plist %s: %w", plistPath, err)
	}
	_, _ = fmt.Fprintf(deps.out, "Uninstalled launchd plist: %s\n", plistPath)
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
	plistPath, err := launchdPlistPath(opts, deps)
	if err != nil {
		return err
	}
	domain := launchdDomain(opts, deps)
	label := launchdLabel(opts.role)
	if out, err := deps.runCmd(ctx, "launchctl", "print", domain+"/"+label); err == nil && strings.Contains(string(out), "state = running") {
		_, _ = fmt.Fprintf(deps.out, "Started %s.\n", label)
		return nil
	}
	if out, err := deps.runCmd(ctx, "launchctl", "bootstrap", domain, plistPath); err != nil {
		if kickOut, kickErr := deps.runCmd(ctx, "launchctl", "kickstart", domain+"/"+label); kickErr != nil {
			return fmt.Errorf("launchctl start %s: %w (%s / %s)", label, kickErr, string(out), string(kickOut))
		}
	}
	_, _ = fmt.Fprintf(deps.out, "Started %s.\n", label)
	return nil
}

func runServiceStop(ctx context.Context, opts serviceOptions, deps serviceDeps) error {
	if err := validateServiceRole(opts.role); err != nil {
		return err
	}
	domain := launchdDomain(opts, deps)
	label := launchdLabel(opts.role)
	if out, err := deps.runCmd(ctx, "launchctl", "bootout", domain+"/"+label); err != nil {
		return fmt.Errorf("launchctl bootout %s: %w (%s)", label, err, string(out))
	}
	_, _ = fmt.Fprintf(deps.out, "Stopped %s.\n", label)
	return nil
}

func runServiceRestart(ctx context.Context, opts serviceOptions, deps serviceDeps) error {
	if err := validateServiceRole(opts.role); err != nil {
		return err
	}
	domain := launchdDomain(opts, deps)
	label := launchdLabel(opts.role)
	_, _ = deps.runCmd(ctx, "launchctl", "bootout", domain+"/"+label)
	return runServiceStart(ctx, opts, deps)
}

func runServiceStatus(ctx context.Context, opts serviceOptions, deps serviceDeps) (int, error) {
	if err := validateServiceRole(opts.role); err != nil {
		return 0, err
	}
	plistPath, err := launchdPlistPath(opts, deps)
	if err != nil {
		return 0, err
	}
	if _, err := deps.stat(plistPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			_, _ = fmt.Fprintln(deps.out, "Status: not installed")
			return serviceStatusNotInstalled, nil
		}
		return 0, fmt.Errorf("stat plist %s: %w", plistPath, err)
	}

	domain := launchdDomain(opts, deps)
	label := launchdLabel(opts.role)
	out, err := deps.runCmd(ctx, "launchctl", "print", domain+"/"+label)
	if err == nil && strings.Contains(string(out), "state = running") {
		_, _ = fmt.Fprintln(deps.out, "Status: running")
		return serviceStatusRunning, nil
	}
	_, _ = fmt.Fprintln(deps.out, "Status: stopped")
	return serviceStatusStopped, nil
}
