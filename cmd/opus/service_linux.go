//go:build !darwin && !windows

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func systemdServiceName(role string) string {
	if role == serviceRoleWorker {
		return "opus-worker"
	}
	return "opus"
}

func systemdUnitPath(opts serviceOptions, deps serviceDeps) (string, error) {
	name := systemdServiceName(opts.role) + ".service"
	if opts.system {
		return filepath.Join("/etc/systemd/system", name), nil
	}
	cfgDir, err := deps.userConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config dir: %w", err)
	}
	return filepath.Join(cfgDir, "systemd", "user", name), nil
}

func systemctlArgs(system bool, args ...string) []string {
	if system {
		return args
	}
	return append([]string{"--user"}, args...)
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
	unitPath, err := systemdUnitPath(opts, deps)
	if err != nil {
		return err
	}

	envPath, created, err := ensureOpusEnvFile(appHome, opts.role)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(unitPath), 0o755); err != nil {
		return fmt.Errorf("create systemd unit dir: %w", err)
	}
	unitContent := renderSystemdUnit(exePath, opts.role, envPath, opts.system)
	if err := os.WriteFile(unitPath, []byte(unitContent), 0o644); err != nil {
		return fmt.Errorf("write systemd unit %s: %w", unitPath, err)
	}

	if out, err := deps.runCmd(ctx, "systemctl", systemctlArgs(opts.system, "daemon-reload")...); err != nil {
		return fmt.Errorf("systemctl daemon-reload: %w (%s)", err, string(out))
	}
	name := systemdServiceName(opts.role)
	if out, err := deps.runCmd(ctx, "systemctl", systemctlArgs(opts.system, "enable", name)...); err != nil {
		return fmt.Errorf("systemctl enable %s: %w (%s)", name, err, string(out))
	}

	_, _ = fmt.Fprintf(deps.out, "Installed systemd unit: %s\n", unitPath)
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
	unitPath, err := systemdUnitPath(opts, deps)
	if err != nil {
		return err
	}
	name := systemdServiceName(opts.role)

	_, _ = deps.runCmd(ctx, "systemctl", systemctlArgs(opts.system, "stop", name)...)
	_, _ = deps.runCmd(ctx, "systemctl", systemctlArgs(opts.system, "disable", name)...)

	if err := os.Remove(unitPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove systemd unit %s: %w", unitPath, err)
	}
	if out, err := deps.runCmd(ctx, "systemctl", systemctlArgs(opts.system, "daemon-reload")...); err != nil {
		return fmt.Errorf("systemctl daemon-reload: %w (%s)", err, string(out))
	}

	_, _ = fmt.Fprintf(deps.out, "Uninstalled systemd unit: %s\n", unitPath)
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
	name := systemdServiceName(opts.role)
	if out, err := deps.runCmd(ctx, "systemctl", systemctlArgs(opts.system, "start", name)...); err != nil {
		return fmt.Errorf("systemctl start %s: %w (%s)", name, err, string(out))
	}
	_, _ = fmt.Fprintf(deps.out, "Started %s.\n", name)
	return nil
}

func runServiceStop(ctx context.Context, opts serviceOptions, deps serviceDeps) error {
	if err := validateServiceRole(opts.role); err != nil {
		return err
	}
	name := systemdServiceName(opts.role)
	if out, err := deps.runCmd(ctx, "systemctl", systemctlArgs(opts.system, "stop", name)...); err != nil {
		return fmt.Errorf("systemctl stop %s: %w (%s)", name, err, string(out))
	}
	_, _ = fmt.Fprintf(deps.out, "Stopped %s.\n", name)
	return nil
}

func runServiceRestart(ctx context.Context, opts serviceOptions, deps serviceDeps) error {
	if err := validateServiceRole(opts.role); err != nil {
		return err
	}
	name := systemdServiceName(opts.role)
	if out, err := deps.runCmd(ctx, "systemctl", systemctlArgs(opts.system, "restart", name)...); err != nil {
		return fmt.Errorf("systemctl restart %s: %w (%s)", name, err, string(out))
	}
	_, _ = fmt.Fprintf(deps.out, "Restarted %s.\n", name)
	return nil
}

func runServiceStatus(ctx context.Context, opts serviceOptions, deps serviceDeps) (int, error) {
	if err := validateServiceRole(opts.role); err != nil {
		return 0, err
	}
	unitPath, err := systemdUnitPath(opts, deps)
	if err != nil {
		return 0, err
	}
	if _, err := deps.stat(unitPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			_, _ = fmt.Fprintln(deps.out, "Status: not installed")
			return serviceStatusNotInstalled, nil
		}
		return 0, fmt.Errorf("stat systemd unit %s: %w", unitPath, err)
	}

	name := systemdServiceName(opts.role)
	if _, err := deps.runCmd(ctx, "systemctl", systemctlArgs(opts.system, "is-active", "--quiet", name)...); err == nil {
		_, _ = fmt.Fprintln(deps.out, "Status: running")
		return serviceStatusRunning, nil
	}
	_, _ = fmt.Fprintln(deps.out, "Status: stopped")
	return serviceStatusStopped, nil
}
