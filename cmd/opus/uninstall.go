package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

const defaultDockerEnvPath = "/.dockerenv"

type uninstallOptions struct {
	purge bool
	yes   bool
}

type uninstallDeps struct {
	in                   io.Reader
	out                  io.Writer
	executable           func() (string, error)
	evalSymlinks         func(string) (string, error)
	homeDir              func() (string, error)
	userHomeDirs         func() []string
	getenv               func(string) string
	stat                 func(string) (os.FileInfo, error)
	checkWritable        func(string) error
	removeBinary         func(string, io.Writer) error
	removeWindowsPath    func(string) (bool, error)
	removeAll            func(string) error
	hasServiceCmd        func() bool
	stopAndRemoveService func() error
	dockerEnvPath        string
}

func defaultUninstallDeps(cmd *cobra.Command) uninstallDeps {
	return uninstallDeps{
		in:            cmd.InOrStdin(),
		out:           cmd.OutOrStdout(),
		executable:    os.Executable,
		evalSymlinks:  filepath.EvalSymlinks,
		homeDir:       opusHomeDir,
		userHomeDirs:  defaultUserHomeDirs,
		getenv:        os.Getenv,
		stat:          os.Stat,
		checkWritable: checkBinaryWritable,
		removeBinary: func(exePath string, out io.Writer) error {
			return removeExecutable(exePath, out)
		},
		removeWindowsPath: removeInstallDirFromUserPath,
		removeAll:         os.RemoveAll,
		hasServiceCmd: func() bool {
			root := cmd.Root()
			if root == nil {
				return false
			}
			for _, sub := range root.Commands() {
				if sub.Name() == "service" {
					return true
				}
			}
			return false
		},
		stopAndRemoveService: func() error {
			return nil
		},
		dockerEnvPath: defaultDockerEnvPath,
	}
}

func defaultUserHomeDirs() []string {
	var dirs []string
	if h, err := os.UserHomeDir(); err == nil && h != "" {
		dirs = append(dirs, h)
	}
	if h := os.Getenv("HOME"); h != "" {
		dirs = append(dirs, h)
	}
	if h := os.Getenv("USERPROFILE"); h != "" {
		dirs = append(dirs, h)
	}
	return dirs
}

func newUninstallCmd() *cobra.Command {
	var opts uninstallOptions
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall the opus binary and optionally purge data",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runUninstall(opts, defaultUninstallDeps(cmd))
		},
	}
	cmd.Flags().BoolVar(&opts.purge, "purge", false, "Also remove the Opus data directory ($APP_HOME)")
	cmd.Flags().BoolVarP(&opts.yes, "yes", "y", false, "Skip confirmation prompt")
	return cmd
}

func runUninstall(opts uninstallOptions, deps uninstallDeps) error {
	if isInsideContainer(deps) {
		return errors.New("cannot uninstall opus inside a container; remove the container or Helm release instead")
	}

	rawExe, err := deps.executable()
	if err != nil {
		return fmt.Errorf("resolve executable path: %w", err)
	}
	exePath, err := deps.evalSymlinks(rawExe)
	if err != nil {
		return fmt.Errorf("resolve executable symlinks: %w", err)
	}
	if !isValidOpusBinary(exePath) {
		return fmt.Errorf("refusing to uninstall non-opus binary: %s", exePath)
	}

	dataDir, err := deps.homeDir()
	if err != nil {
		return fmt.Errorf("resolve data directory: %w", err)
	}
	if opts.purge {
		if isUnsafePurgePath(dataDir, deps.userHomeDirs(), deps.evalSymlinks) {
			return fmt.Errorf("refusing to purge unsafe data directory: %q", dataDir)
		}
	}

	if err := deps.checkWritable(exePath); err != nil {
		return formatNotWritableError(exePath, err)
	}

	if !opts.yes {
		confirmed, err := promptConfirmation(opts, exePath, dataDir, deps)
		if err != nil {
			return err
		}
		if !confirmed {
			_, _ = fmt.Fprintln(deps.out, "Uninstall cancelled.")
			return nil
		}
	}

	// (a) Stop and remove service if opus service command exists.
	if deps.hasServiceCmd != nil && deps.hasServiceCmd() && deps.stopAndRemoveService != nil {
		if err := deps.stopAndRemoveService(); err != nil {
			return fmt.Errorf("stop/remove service: %w", err)
		}
	}

	// (b) Remove binary.
	if err := deps.removeBinary(exePath, deps.out); err != nil {
		return formatNotWritableError(exePath, err)
	}

	// (c) On Windows, clean up user PATH if matching install.ps1 directory.
	pathRemoved := false
	installDir := filepath.Dir(exePath)
	if deps.removeWindowsPath != nil {
		removed, err := deps.removeWindowsPath(installDir)
		if err != nil {
			_, _ = fmt.Fprintf(deps.out, "Note: could not update user PATH: %v\n", err)
		} else {
			pathRemoved = removed
		}
	}

	// (d) Purge data directory if requested.
	if opts.purge {
		if err := deps.removeAll(dataDir); err != nil {
			return fmt.Errorf("remove data directory %s: %w", dataDir, err)
		}
	}

	// Summary.
	_, _ = fmt.Fprintf(deps.out, "Removed binary: %s\n", exePath)
	if pathRemoved {
		_, _ = fmt.Fprintf(deps.out, "Removed from user PATH: %s\n", installDir)
	}
	if opts.purge {
		_, _ = fmt.Fprintf(deps.out, "Removed data directory: %s\n", dataDir)
	} else {
		_, _ = fmt.Fprintf(deps.out, "Kept data directory: %s\n", dataDir)
	}
	return nil
}

func isInsideContainer(deps uninstallDeps) bool {
	return checkInsideContainer(deps.dockerEnvPath, deps.stat, deps.getenv)
}

func checkInsideContainer(dockerEnvPath string, stat func(string) (os.FileInfo, error), getenv func(string) string) bool {
	if dockerEnvPath != "" && stat != nil {
		if _, err := stat(dockerEnvPath); err == nil {
			return true
		}
	}
	if getenv != nil && getenv("KUBERNETES_SERVICE_HOST") != "" {
		return true
	}
	return false
}

func isValidOpusBinary(exePath string) bool {
	base := filepath.Base(exePath)
	return base == "opus" || strings.EqualFold(base, "opus.exe")
}

func isUnsafePurgePath(dataDir string, homeDirs []string, evalSymlinks func(string) (string, error)) bool {
	trimmed := strings.TrimSpace(dataDir)
	if trimmed == "" {
		return true
	}
	if isRootOrEmptyPath(trimmed) {
		return true
	}
	cleaned := filepath.Clean(trimmed)
	if isRootOrEmptyPath(cleaned) {
		return true
	}

	resolved := cleaned
	if evalSymlinks != nil {
		if r, err := evalSymlinks(cleaned); err == nil && r != "" {
			resolved = filepath.Clean(r)
			if isRootOrEmptyPath(resolved) {
				return true
			}
		}
	}

	for _, h := range homeDirs {
		if h == "" {
			continue
		}
		cleanH := filepath.Clean(h)
		if pathsEqual(cleaned, cleanH) || pathsEqual(resolved, cleanH) {
			return true
		}
		if evalSymlinks != nil {
			if rh, err := evalSymlinks(cleanH); err == nil && rh != "" {
				cleanRH := filepath.Clean(rh)
				if pathsEqual(cleaned, cleanRH) || pathsEqual(resolved, cleanRH) {
					return true
				}
			}
		}
	}
	return false
}

func isRootOrEmptyPath(p string) bool {
	if p == "" || p == "." || p == "/" || p == "\\" {
		return true
	}
	if p == filepath.Dir(p) {
		return true
	}
	// Explicit Windows drive root check (e.g. "C:", "C:\", "C:/") for cross-platform safety.
	if len(p) >= 2 && ((p[0] >= 'A' && p[0] <= 'Z') || (p[0] >= 'a' && p[0] <= 'z')) && p[1] == ':' {
		rest := strings.Trim(p[2:], "\\/")
		if rest == "" {
			return true
		}
	}
	return false
}

func pathsEqual(a, b string) bool {
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}

func promptConfirmation(opts uninstallOptions, exePath, dataDir string, deps uninstallDeps) (bool, error) {
	_, _ = fmt.Fprintln(deps.out, "The following paths will be removed:")
	_, _ = fmt.Fprintf(deps.out, "  - %s\n", exePath)
	if opts.purge {
		_, _ = fmt.Fprintf(deps.out, "  - %s\n", dataDir)
		_, _ = fmt.Fprint(deps.out, "Type 'purge' to confirm removal: ")
	} else {
		_, _ = fmt.Fprint(deps.out, "Proceed with uninstall? [y/N]: ")
	}

	reader := bufio.NewReader(deps.in)
	line, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return false, fmt.Errorf("read confirmation: %w", err)
	}
	answer := strings.TrimSpace(line)
	if opts.purge {
		return answer == "purge", nil
	}
	return strings.EqualFold(answer, "y") || strings.EqualFold(answer, "yes"), nil
}

func formatNotWritableError(exePath string, err error) error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("binary path %s is not writable (%w)\nHint: run 'sudo rm -f %s'", exePath, err, exePath)
	}
	return fmt.Errorf("binary path %s is not writable: %w", exePath, err)
}

// removeDirFromPath removes all occurrences of targetDir (case-insensitive,
// ignoring trailing slashes) from a semicolon-delimited Windows PATH string.
func removeDirFromPath(pathValue, targetDir string) (string, bool) {
	normTarget := strings.TrimRight(strings.TrimSpace(targetDir), "\\/")
	if normTarget == "" {
		return pathValue, false
	}

	parts := strings.Split(pathValue, ";")
	kept := make([]string, 0, len(parts))
	removed := false
	for _, p := range parts {
		if p == "" {
			continue
		}
		normPart := strings.TrimRight(strings.TrimSpace(p), "\\/")
		if strings.EqualFold(normPart, normTarget) {
			removed = true
			continue
		}
		kept = append(kept, p)
	}
	if !removed {
		return pathValue, false
	}
	return strings.Join(kept, ";"), true
}
