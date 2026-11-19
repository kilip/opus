package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

const (
	serviceRoleServer = "server"
	serviceRoleWorker = "worker"

	serviceStatusRunning      = 0
	serviceStatusStopped      = 3
	serviceStatusNotInstalled = 4
)

type serviceOptions struct {
	role   string
	system bool
}

type serviceCommandRunner func(ctx context.Context, name string, args ...string) ([]byte, error)

type serviceDeps struct {
	out           io.Writer
	executable    func() (string, error)
	evalSymlinks  func(string) (string, error)
	homeDir       func() (string, error)
	userConfigDir func() (string, error)
	userHomeDir   func() (string, error)
	getenv        func(string) string
	stat          func(string) (os.FileInfo, error)
	runCmd        serviceCommandRunner
	uid           func() int
}

func defaultServiceDeps(cmd *cobra.Command) serviceDeps {
	return serviceDeps{
		out:           cmd.OutOrStdout(),
		executable:    os.Executable,
		evalSymlinks:  filepath.EvalSymlinks,
		homeDir:       opusHomeDir,
		userConfigDir: os.UserConfigDir,
		userHomeDir:   os.UserHomeDir,
		getenv:        os.Getenv,
		stat:          os.Stat,
		runCmd: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return exec.CommandContext(ctx, name, args...).CombinedOutput()
		},
		uid: os.Getuid,
	}
}

func newServiceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "service",
		Short: "Manage opus as a background system/user service",
	}

	var opts serviceOptions
	addServiceFlags := func(sub *cobra.Command) {
		sub.Flags().StringVar(&opts.role, "role", serviceRoleServer, "Service role: server or worker")
		sub.Flags().BoolVar(&opts.system, "system", false, "Install/manage as a system-wide service (default: user service)")
	}

	installCmd := &cobra.Command{
		Use:   "install",
		Short: "Install the opus service",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			return runServiceInstall(c.Context(), opts, defaultServiceDeps(c))
		},
	}
	addServiceFlags(installCmd)

	uninstallCmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall the opus service (keeps $APP_HOME/opus.env)",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			return runServiceUninstall(c.Context(), opts, defaultServiceDeps(c))
		},
	}
	addServiceFlags(uninstallCmd)

	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Start the opus service",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			return runServiceStart(c.Context(), opts, defaultServiceDeps(c))
		},
	}
	addServiceFlags(startCmd)

	stopCmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop the opus service",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			return runServiceStop(c.Context(), opts, defaultServiceDeps(c))
		},
	}
	addServiceFlags(stopCmd)

	restartCmd := &cobra.Command{
		Use:   "restart",
		Short: "Restart the opus service",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			return runServiceRestart(c.Context(), opts, defaultServiceDeps(c))
		},
	}
	addServiceFlags(restartCmd)

	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Check opus service status (exits 0=running, 3=stopped, 4=not installed)",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			code, err := runServiceStatus(c.Context(), opts, defaultServiceDeps(c))
			if err != nil {
				return err
			}
			if code != serviceStatusRunning {
				c.SilenceErrors = true
				return exitCodeError{code: code}
			}
			return nil
		},
	}
	addServiceFlags(statusCmd)

	var (
		runRole    string
		runEnvFile string
		runLogFile string
	)
	runCmd := &cobra.Command{
		Use:    "run",
		Short:  "Internal service launcher that loads opus.env before starting",
		Hidden: true,
		Args:   cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			return runServiceExec(c.Context(), runRole, runEnvFile, runLogFile)
		},
	}
	runCmd.Flags().StringVar(&runRole, "role", serviceRoleServer, "Service role")
	runCmd.Flags().StringVar(&runEnvFile, "env-file", "", "Path to opus.env")
	runCmd.Flags().StringVar(&runLogFile, "log-file", "", "Optional log file path")

	cmd.AddCommand(installCmd, uninstallCmd, startCmd, stopCmd, restartCmd, statusCmd, runCmd)
	return cmd
}

func validateServiceRole(role string) error {
	if role != serviceRoleServer && role != serviceRoleWorker {
		return fmt.Errorf("invalid --role %q: must be %q or %q", role, serviceRoleServer, serviceRoleWorker)
	}
	return nil
}

func resolveServiceExePath(deps serviceDeps) (string, error) {
	raw, err := deps.executable()
	if err != nil {
		return "", fmt.Errorf("resolve executable path: %w", err)
	}
	if deps.evalSymlinks != nil {
		if resolved, err := deps.evalSymlinks(raw); err == nil && resolved != "" {
			return resolved, nil
		}
	}
	return raw, nil
}

func checkWorkerNodeConfig(appHome string, stat func(string) (os.FileInfo, error)) error {
	cfgPath := nodeConfigPath(appHome)
	if _, err := stat(cfgPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("node configuration not found at %s: run 'opus join <url>' first", cfgPath)
		}
		return fmt.Errorf("check node configuration %s: %w", cfgPath, err)
	}
	return nil
}

// ensureOpusEnvFile creates $APP_HOME/opus.env with mode 0600 if it does not already exist.
// Existing files are never overwritten.
func ensureOpusEnvFile(appHome, role string) (string, bool, error) {
	if err := os.MkdirAll(appHome, 0o700); err != nil {
		return "", false, fmt.Errorf("create app home %s: %w", appHome, err)
	}
	envPath := filepath.Join(appHome, "opus.env")
	f, err := os.OpenFile(envPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return envPath, false, nil
		}
		return "", false, fmt.Errorf("create env file %s: %w", envPath, err)
	}
	defer func() { _ = f.Close() }()

	var sb strings.Builder
	fmt.Fprintf(&sb, "APP_HOME=%s\n", appHome)
	if role == serviceRoleServer {
		b := make([]byte, 32)
		if _, err := rand.Read(b); err != nil {
			_ = os.Remove(envPath)
			return "", false, fmt.Errorf("generate AUTH_SECRET: %w", err)
		}
		fmt.Fprintf(&sb, "AUTH_SECRET=%s\n", hex.EncodeToString(b))
	}

	if _, err := f.WriteString(sb.String()); err != nil {
		_ = os.Remove(envPath)
		return "", false, fmt.Errorf("write env file %s: %w", envPath, err)
	}
	return envPath, true, nil
}

func loadOpusEnvFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read env file %s: %w", path, err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		if k != "" && os.Getenv(k) == "" {
			_ = os.Setenv(k, v)
		}
	}
	return nil
}

func runServiceExec(ctx context.Context, role, envFile, logFile string) error {
	if err := validateServiceRole(role); err != nil {
		return err
	}
	if envFile == "" {
		home, err := opusHomeDir()
		if err != nil {
			return err
		}
		envFile = filepath.Join(home, "opus.env")
	}
	if err := loadOpusEnvFile(envFile); err != nil {
		return err
	}
	if logFile != "" {
		if err := os.MkdirAll(filepath.Dir(logFile), 0o755); err == nil {
			if f, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644); err == nil {
				os.Stdout = f
				os.Stderr = f
				defer func() { _ = f.Close() }()
			}
		}
	}

	if role == serviceRoleWorker {
		return runWorker(ctx)
	}
	serveCmd := newServeCmd()
	return serveCmd.RunE(serveCmd, nil)
}

func renderSystemdUnit(exePath, role, envPath string, system bool) string {
	subcmd := "serve"
	desc := "Opus Server"
	if role == serviceRoleWorker {
		subcmd = "worker"
		desc = "Opus Worker"
	}
	wantedBy := "default.target"
	if system {
		wantedBy = "multi-user.target"
	}
	return fmt.Sprintf(`[Unit]
Description=%s
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=%s %s
EnvironmentFile=%s
Restart=on-failure
RestartSec=2

[Install]
WantedBy=%s
`, desc, exePath, subcmd, envPath, wantedBy)
}

func renderLaunchdPlist(label, exePath, role, envPath, logPath string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>%s</string>
  <key>ProgramArguments</key>
  <array>
    <string>%s</string>
    <string>service</string>
    <string>run</string>
    <string>--role</string>
    <string>%s</string>
    <string>--env-file</string>
    <string>%s</string>
  </array>
  <key>RunAtLoad</key>
  <true/>
  <key>KeepAlive</key>
  <dict>
    <key>SuccessfulExit</key>
    <false/>
  </dict>
  <key>StandardOutPath</key>
  <string>%s</string>
  <key>StandardErrorPath</key>
  <string>%s</string>
</dict>
</plist>
`, xmlEscape(label), xmlEscape(exePath), xmlEscape(role), xmlEscape(envPath), xmlEscape(logPath), xmlEscape(logPath))
}

func renderWindowsTaskXML(exePath, role, envPath, logPath string) string {
	args := fmt.Sprintf(`service run --role %s --env-file "%s" --log-file "%s"`, role, envPath, logPath)
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-16"?>
<Task version="1.2" xmlns="http://schemas.microsoft.com/windows/2004/02/mit/task">
  <Triggers>
    <LogonTrigger>
      <Enabled>true</Enabled>
    </LogonTrigger>
  </Triggers>
  <Settings>
    <MultipleInstancesPolicy>IgnoreNew</MultipleInstancesPolicy>
    <DisallowStartIfOnBatteries>false</DisallowStartIfOnBatteries>
    <StopIfGoingOnBatteries>false</StopIfGoingOnBatteries>
    <AllowHardTerminate>true</AllowHardTerminate>
    <StartWhenAvailable>true</StartWhenAvailable>
    <ExecutionTimeLimit>PT0S</ExecutionTimeLimit>
    <RestartOnFailure>
      <Interval>PT1M</Interval>
      <Count>3</Count>
    </RestartOnFailure>
    <Enabled>true</Enabled>
  </Settings>
  <Actions Context="Author">
    <Exec>
      <Command>%s</Command>
      <Arguments>%s</Arguments>
    </Exec>
  </Actions>
</Task>
`, xmlEscape(exePath), xmlEscape(args))
}

func xmlEscape(s string) string {
	r := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
		"'", "&apos;",
	)
	return r.Replace(s)
}
