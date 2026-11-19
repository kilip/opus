package main

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const defaultGitHubAPIBaseURL = "https://api.github.com"

type exitCodeError struct {
	code int
	err  error
}

func (e exitCodeError) Error() string {
	if e.err != nil {
		return e.err.Error()
	}
	return fmt.Sprintf("exit status %d", e.code)
}

func (e exitCodeError) Unwrap() error {
	return e.err
}

type selfUpdateOptions struct {
	check   bool
	version string
	yes     bool
}

type selfUpdateDeps struct {
	in             io.Reader
	out            io.Writer
	httpClient     *http.Client
	apiBaseURL     string
	currentVersion string
	goos           string
	goarch         string
	executable     func() (string, error)
	evalSymlinks   func(string) (string, error)
	homeDir        func() (string, error)
	getenv         func(string) string
	stat           func(string) (os.FileInfo, error)
	checkWritable  func(string) error
	replace        func(exePath, tmpBinPath string) error
	dockerEnvPath  string
}

func defaultSelfUpdateDeps(cmd *cobra.Command) selfUpdateDeps {
	apiBaseURL := defaultGitHubAPIBaseURL
	if v := os.Getenv("TEST_SELF_UPDATE_API_URL"); v != "" {
		apiBaseURL = v
	}
	return selfUpdateDeps{
		in:             cmd.InOrStdin(),
		out:            cmd.OutOrStdout(),
		httpClient:     &http.Client{Timeout: 2 * time.Minute},
		apiBaseURL:     apiBaseURL,
		currentVersion: Version,
		goos:           runtime.GOOS,
		goarch:         runtime.GOARCH,
		executable:     os.Executable,
		evalSymlinks:   filepath.EvalSymlinks,
		homeDir:        opusHomeDir,
		getenv:         os.Getenv,
		stat:           os.Stat,
		checkWritable:  checkBinaryWritable,
		replace:        replaceBinary,
		dockerEnvPath:  defaultDockerEnvPath,
	}
}

func newSelfUpdateCmd() *cobra.Command {
	var opts selfUpdateOptions
	cmd := &cobra.Command{
		Use:   "self-update",
		Short: "Update the opus binary to a new release",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			err := runSelfUpdate(cmd.Context(), opts, defaultSelfUpdateDeps(cmd))
			var exitErr exitCodeError
			if errors.As(err, &exitErr) {
				cmd.SilenceErrors = true
			}
			return err
		},
	}
	cmd.Flags().BoolVar(&opts.check, "check", false, "Check for updates without installing (exits 10 if behind, 0 if current)")
	cmd.Flags().StringVar(&opts.version, "version", "", "Target version to install (e.g. 0.3.0 or v0.3.0)")
	cmd.Flags().BoolVarP(&opts.yes, "yes", "y", false, "Skip confirmation prompts and allow downgrades")
	return cmd
}

type githubRelease struct {
	TagName    string        `json:"tag_name"`
	Draft      bool          `json:"draft"`
	Prerelease bool          `json:"prerelease"`
	Assets     []githubAsset `json:"assets"`
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type semver struct {
	major int
	minor int
	patch int
	pre   []string
}

func (s semver) String() string {
	base := fmt.Sprintf("%d.%d.%d", s.major, s.minor, s.patch)
	if len(s.pre) > 0 {
		return base + "-" + strings.Join(s.pre, ".")
	}
	return base
}

func parseSemver(raw string) (semver, error) {
	s := strings.TrimSpace(raw)
	s = strings.TrimPrefix(s, "v")
	s = strings.TrimPrefix(s, "V")
	if core, _, ok := strings.Cut(s, "+"); ok {
		s = core
	}
	core, preStr, hasPre := strings.Cut(s, "-")
	parts := strings.Split(core, ".")
	if len(parts) != 3 {
		return semver{}, fmt.Errorf("invalid semver %q", raw)
	}
	major, err := parseNonNegInt(parts[0])
	if err != nil {
		return semver{}, fmt.Errorf("invalid semver %q: %w", raw, err)
	}
	minor, err := parseNonNegInt(parts[1])
	if err != nil {
		return semver{}, fmt.Errorf("invalid semver %q: %w", raw, err)
	}
	patch, err := parseNonNegInt(parts[2])
	if err != nil {
		return semver{}, fmt.Errorf("invalid semver %q: %w", raw, err)
	}

	var pre []string
	if hasPre {
		if preStr == "" {
			return semver{}, fmt.Errorf("invalid semver %q: empty prerelease", raw)
		}
		pre = strings.Split(preStr, ".")
		for _, id := range pre {
			if id == "" {
				return semver{}, fmt.Errorf("invalid semver %q: empty prerelease identifier", raw)
			}
		}
	}

	return semver{major: major, minor: minor, patch: patch, pre: pre}, nil
}

func parseNonNegInt(s string) (int, error) {
	if s == "" {
		return 0, errors.New("empty number")
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 {
		return 0, fmt.Errorf("invalid integer %q", s)
	}
	return n, nil
}

func compareSemver(a, b semver) int {
	if a.major != b.major {
		return cmpInt(a.major, b.major)
	}
	if a.minor != b.minor {
		return cmpInt(a.minor, b.minor)
	}
	if a.patch != b.patch {
		return cmpInt(a.patch, b.patch)
	}
	if len(a.pre) == 0 && len(b.pre) == 0 {
		return 0
	}
	if len(a.pre) == 0 {
		return 1
	}
	if len(b.pre) == 0 {
		return -1
	}

	n := min(len(a.pre), len(b.pre))
	for i := range n {
		if a.pre[i] == b.pre[i] {
			continue
		}
		aNum, aIsNum := parsePreInt(a.pre[i])
		bNum, bIsNum := parsePreInt(b.pre[i])
		switch {
		case aIsNum && bIsNum:
			return cmpInt(aNum, bNum)
		case aIsNum:
			return -1
		case bIsNum:
			return 1
		default:
			return strings.Compare(a.pre[i], b.pre[i])
		}
	}
	return cmpInt(len(a.pre), len(b.pre))
}

func parsePreInt(s string) (int, bool) {
	if s == "" {
		return 0, false
	}
	for i := range len(s) {
		if s[i] < '0' || s[i] > '9' {
			return 0, false
		}
	}
	n, err := strconv.Atoi(s)
	return n, err == nil
}

func cmpInt(a, b int) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

func runSelfUpdate(ctx context.Context, opts selfUpdateOptions, deps selfUpdateDeps) error {
	if checkInsideContainer(deps.dockerEnvPath, deps.stat, deps.getenv) {
		return errors.New("cannot self-update opus inside a container; pull a new container image instead")
	}

	curVer, err := parseSemver(deps.currentVersion)
	if err != nil {
		return fmt.Errorf("parse current version %q: %w", deps.currentVersion, err)
	}

	releases, err := fetchReleases(ctx, deps.httpClient, deps.apiBaseURL)
	if err != nil {
		return err
	}

	targetRelease, targetVer, err := selectRelease(releases, opts.version)
	if err != nil {
		return err
	}

	cmp := compareSemver(curVer, targetVer)
	if opts.check {
		_, _ = fmt.Fprintf(deps.out, "Current version: %s\n", curVer.String())
		_, _ = fmt.Fprintf(deps.out, "Latest version:  %s\n", targetVer.String())
		if cmp < 0 {
			_, _ = fmt.Fprintln(deps.out, "Update available. Run 'opus self-update' to upgrade.")
			return exitCodeError{code: 10}
		}
		_, _ = fmt.Fprintln(deps.out, "Opus is up to date.")
		return nil
	}

	rawExe, err := deps.executable()
	if err != nil {
		return fmt.Errorf("resolve executable path: %w", err)
	}
	exePath, err := deps.evalSymlinks(rawExe)
	if err != nil {
		return fmt.Errorf("resolve executable symlinks: %w", err)
	}
	if err := deps.checkWritable(exePath); err != nil {
		return fmt.Errorf("binary path %s is not writable (%w); re-run the install script with appropriate permissions (https://github.com/kilip/opus#quick-install)", exePath, err)
	}

	if cmp == 0 {
		_, _ = fmt.Fprintf(deps.out, "Opus is already at version %s.\n", curVer.String())
		return nil
	}
	if cmp > 0 && !opts.yes {
		return fmt.Errorf("target version %s is older than current version %s; pass --yes to confirm downgrade", targetVer.String(), curVer.String())
	}

	if dataPath, ok := existingDataPath(deps); ok {
		_, _ = fmt.Fprintf(deps.out, "Warning: existing Opus data detected at %s. Back up your database before updating (see docs/backup.md).\n", dataPath)
		if !opts.yes {
			confirmed, err := promptUpdateConfirmation(targetVer.String(), deps)
			if err != nil {
				return err
			}
			if !confirmed {
				_, _ = fmt.Fprintln(deps.out, "Update cancelled.")
				return nil
			}
		}
	}

	archiveName, binName := releaseAssetNames(targetVer.String(), deps.goos, deps.goarch)
	archiveAsset, ok := findAsset(targetRelease.Assets, archiveName)
	if !ok {
		return fmt.Errorf("release %s does not contain asset %s", targetRelease.TagName, archiveName)
	}
	checksumsAsset, ok := findAsset(targetRelease.Assets, "checksums.txt")
	if !ok {
		return fmt.Errorf("release %s does not contain checksums.txt", targetRelease.TagName)
	}

	expectedSum, err := fetchExpectedChecksum(ctx, deps.httpClient, checksumsAsset.BrowserDownloadURL, archiveName)
	if err != nil {
		return err
	}

	tmpArchive, err := downloadAndVerifyArchive(ctx, deps.httpClient, archiveAsset.BrowserDownloadURL, archiveName, expectedSum)
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmpArchive) }()

	tmpBin, err := extractBinaryToSameDir(tmpArchive, filepath.Dir(exePath), binName, deps.goos)
	if err != nil {
		return err
	}

	if err := deps.replace(exePath, tmpBin); err != nil {
		_ = os.Remove(tmpBin)
		return fmt.Errorf("replace binary %s: %w", exePath, err)
	}

	_, _ = fmt.Fprintf(deps.out, "Updated opus to %s.\n", targetVer.String())
	_, _ = fmt.Fprintln(deps.out, "restart opus to use the new version")
	return nil
}

func fetchReleases(ctx context.Context, client *http.Client, apiBaseURL string) ([]githubRelease, error) {
	url := strings.TrimRight(apiBaseURL, "/") + "/repos/kilip/opus/releases?per_page=100"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build releases request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "opus-self-update")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch releases: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch releases: unexpected status %s", resp.Status)
	}

	var releases []githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, fmt.Errorf("decode releases: %w", err)
	}
	return releases, nil
}

func selectRelease(releases []githubRelease, targetVersion string) (githubRelease, semver, error) {
	if targetVersion != "" {
		want, err := parseSemver(targetVersion)
		if err != nil {
			return githubRelease{}, semver{}, err
		}
		for _, r := range releases {
			if r.Draft {
				continue
			}
			v, err := parseSemver(r.TagName)
			if err != nil {
				continue
			}
			if compareSemver(v, want) == 0 {
				return r, v, nil
			}
		}
		return githubRelease{}, semver{}, fmt.Errorf("release %s not found", targetVersion)
	}

	var (
		found   bool
		bestRel githubRelease
		bestVer semver
	)
	for _, r := range releases {
		if r.Draft {
			continue
		}
		v, err := parseSemver(r.TagName)
		if err != nil {
			continue
		}
		if !found || compareSemver(v, bestVer) > 0 {
			found = true
			bestRel = r
			bestVer = v
		}
	}
	if !found {
		return githubRelease{}, semver{}, errors.New("no valid releases found")
	}
	return bestRel, bestVer, nil
}

func existingDataPath(deps selfUpdateDeps) (string, bool) {
	if deps.stat == nil {
		return "", false
	}
	if deps.getenv != nil {
		if dbURL := deps.getenv("APP_DATABASE_URL"); dbURL != "" {
			dbPath := strings.TrimPrefix(dbURL, "file:")
			if before, _, ok := strings.Cut(dbPath, "?"); ok {
				dbPath = before
			}
			if dbPath != "" {
				if _, err := deps.stat(dbPath); err == nil {
					return dbPath, true
				}
			}
		}
	}
	if deps.homeDir != nil {
		if dataDir, err := deps.homeDir(); err == nil && dataDir != "" {
			dbPath := filepath.Join(dataDir, "opus.db")
			if _, err := deps.stat(dbPath); err == nil {
				return dbPath, true
			}
			if _, err := deps.stat(dataDir); err == nil {
				return dataDir, true
			}
		}
	}
	return "", false
}

func promptUpdateConfirmation(targetVer string, deps selfUpdateDeps) (bool, error) {
	_, _ = fmt.Fprintf(deps.out, "Proceed with update to %s? [y/N]: ", targetVer)
	reader := bufio.NewReader(deps.in)
	line, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return false, fmt.Errorf("read confirmation: %w", err)
	}
	answer := strings.TrimSpace(line)
	return strings.EqualFold(answer, "y") || strings.EqualFold(answer, "yes"), nil
}

func releaseAssetNames(version, goos, goarch string) (archiveName, binName string) {
	if goos == "windows" {
		return fmt.Sprintf("opus_%s_%s_%s.zip", version, goos, goarch), "opus.exe"
	}
	return fmt.Sprintf("opus_%s_%s_%s.tar.gz", version, goos, goarch), "opus"
}

func findAsset(assets []githubAsset, name string) (githubAsset, bool) {
	for _, a := range assets {
		if a.Name == name {
			return a, true
		}
	}
	return githubAsset{}, false
}

func fetchExpectedChecksum(ctx context.Context, client *http.Client, url, archiveName string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("build checksums request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("download checksums.txt: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download checksums.txt: unexpected status %s", resp.Status)
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 2 && fields[1] == archiveName {
			return fields[0], nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("read checksums.txt: %w", err)
	}
	return "", fmt.Errorf("checksum for %s not found in checksums.txt", archiveName)
}

func downloadAndVerifyArchive(ctx context.Context, client *http.Client, url, archiveName, expectedSum string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("build archive request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("download %s: %w", archiveName, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download %s: unexpected status %s", archiveName, resp.Status)
	}

	tmpFile, err := os.CreateTemp("", "opus-archive-*")
	if err != nil {
		return "", fmt.Errorf("create temp archive file: %w", err)
	}
	tmpPath := tmpFile.Name()

	hasher := sha256.New()
	if _, err := io.Copy(io.MultiWriter(tmpFile, hasher), resp.Body); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
		return "", fmt.Errorf("download %s: %w", archiveName, err)
	}
	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return "", fmt.Errorf("close temp archive %s: %w", tmpPath, err)
	}

	actualSum := hex.EncodeToString(hasher.Sum(nil))
	if !strings.EqualFold(actualSum, expectedSum) {
		_ = os.Remove(tmpPath)
		return "", fmt.Errorf("checksum mismatch for %s: expected %s, got %s", archiveName, expectedSum, actualSum)
	}
	return tmpPath, nil
}

func extractBinaryToSameDir(archivePath, targetDir, binName, goos string) (string, error) {
	tmpBin, err := os.CreateTemp(targetDir, ".opus-update-*")
	if err != nil {
		return "", fmt.Errorf("create temp binary in %s: %w", targetDir, err)
	}
	tmpBinPath := tmpBin.Name()

	cleanup := func() {
		_ = tmpBin.Close()
		_ = os.Remove(tmpBinPath)
	}

	if goos == "windows" {
		err = extractFromZip(archivePath, binName, tmpBin)
	} else {
		err = extractFromTarGz(archivePath, binName, tmpBin)
	}
	if err != nil {
		cleanup()
		return "", err
	}

	if goos != "windows" {
		if err := tmpBin.Chmod(0o755); err != nil {
			cleanup()
			return "", fmt.Errorf("chmod temp binary: %w", err)
		}
	}
	if err := tmpBin.Close(); err != nil {
		_ = os.Remove(tmpBinPath)
		return "", fmt.Errorf("close temp binary: %w", err)
	}
	return tmpBinPath, nil
}

func extractFromTarGz(archivePath, binName string, dst io.Writer) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("open archive %s: %w", archivePath, err)
	}
	defer func() { _ = f.Close() }()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("open gzip stream: %w", err)
	}
	defer func() { _ = gzr.Close() }()

	tr := tar.NewReader(gzr)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("read tar archive: %w", err)
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		if filepath.Base(hdr.Name) == binName {
			if _, err := io.Copy(dst, tr); err != nil {
				return fmt.Errorf("extract %s from tar: %w", binName, err)
			}
			return nil
		}
	}
	return fmt.Errorf("binary %s not found in archive", binName)
}

func extractFromZip(archivePath, binName string, dst io.Writer) error {
	zr, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("open zip archive %s: %w", archivePath, err)
	}
	defer func() { _ = zr.Close() }()

	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		if filepath.Base(f.Name) == binName {
			rc, err := f.Open()
			if err != nil {
				return fmt.Errorf("open %s in zip: %w", binName, err)
			}
			_, copyErr := io.Copy(dst, rc)
			closeErr := rc.Close()
			if copyErr != nil {
				return fmt.Errorf("extract %s from zip: %w", binName, copyErr)
			}
			if closeErr != nil {
				return fmt.Errorf("close %s in zip: %w", binName, closeErr)
			}
			return nil
		}
	}
	return fmt.Errorf("binary %s not found in archive", binName)
}
