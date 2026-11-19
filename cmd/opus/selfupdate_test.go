package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func buildTestOpusBinaryWithVersion(t *testing.T, dir, version string) string {
	t.Helper()
	binName := "opus"
	if runtime.GOOS == "windows" {
		binName = "opus.exe"
	}
	binPath := filepath.Join(dir, binName)
	ldflags := fmt.Sprintf("-X main.Version=%s", version)
	cmd := exec.Command("go", "build", "-ldflags", ldflags, "-o", binPath, ".")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build opus binary (%s): %v\n%s", version, err, out)
	}
	return binPath
}

func makeTestArchive(t *testing.T, binPath, goos string) []byte {
	t.Helper()
	data, err := os.ReadFile(binPath)
	if err != nil {
		t.Fatalf("read binary %s: %v", binPath, err)
	}

	var buf bytes.Buffer
	if goos == "windows" {
		zw := zip.NewWriter(&buf)
		fw, err := zw.Create("opus.exe")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := fw.Write(data); err != nil {
			t.Fatal(err)
		}
		if err := zw.Close(); err != nil {
			t.Fatal(err)
		}
		return buf.Bytes()
	}

	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)
	hdr := &tar.Header{
		Name: "opus",
		Mode: 0o755,
		Size: int64(len(data)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

type fakeReleaseSpec struct {
	tag         string
	prerelease  bool
	draft       bool
	archiveData []byte
	checksum    string
}

func newFakeReleaseServer(t *testing.T, specs []fakeReleaseSpec, goos, goarch string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		baseURL := "http://" + r.Host
		switch r.URL.Path {
		case "/repos/kilip/opus/releases":
			var releases []githubRelease
			for _, s := range specs {
				ver := strings.TrimPrefix(s.tag, "v")
				archiveName, _ := releaseAssetNames(ver, goos, goarch)
				releases = append(releases, githubRelease{
					TagName:    s.tag,
					Draft:      s.draft,
					Prerelease: s.prerelease,
					Assets: []githubAsset{
						{
							Name:               archiveName,
							BrowserDownloadURL: baseURL + "/download/" + s.tag + "/" + archiveName,
						},
						{
							Name:               "checksums.txt",
							BrowserDownloadURL: baseURL + "/download/" + s.tag + "/checksums.txt",
						},
					},
				})
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(releases)
		default:
			for _, s := range specs {
				ver := strings.TrimPrefix(s.tag, "v")
				archiveName, _ := releaseAssetNames(ver, goos, goarch)
				if r.URL.Path == "/download/"+s.tag+"/"+archiveName {
					_, _ = w.Write(s.archiveData)
					return
				}
				if r.URL.Path == "/download/"+s.tag+"/checksums.txt" {
					sum := s.checksum
					if sum == "" {
						h := sha256.Sum256(s.archiveData)
						sum = hex.EncodeToString(h[:])
					}
					_, _ = fmt.Fprintf(w, "%s  %s\n", sum, archiveName)
					return
				}
			}
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestSelfUpdate_CheckExitCodes(t *testing.T) {
	srv := newFakeReleaseServer(t, []fakeReleaseSpec{
		{tag: "v0.2.0", archiveData: []byte("dummy")},
	}, runtime.GOOS, runtime.GOARCH)

	installDir := t.TempDir()
	binPath := buildTestOpusBinaryWithVersion(t, installDir, "0.1.0")

	// AC1: behind -> exit code 10
	cmd := exec.Command(binPath, "self-update", "--check")
	cmd.Env = append(os.Environ(),
		"TEST_SELF_UPDATE_API_URL="+srv.URL,
		"KUBERNETES_SERVICE_HOST=",
	)
	out, err := cmd.CombinedOutput()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 10 {
		t.Fatalf("expected exit code 10 when behind, got err=%v, out:\n%s", err, out)
	}
	if !strings.Contains(string(out), "0.1.0") || !strings.Contains(string(out), "0.2.0") {
		t.Fatalf("expected output to report current and latest version, got:\n%s", out)
	}

	// AC1: current -> exit code 0
	binCurrent := buildTestOpusBinaryWithVersion(t, t.TempDir(), "0.2.0")
	cmd2 := exec.Command(binCurrent, "self-update", "--check")
	cmd2.Env = append(os.Environ(),
		"TEST_SELF_UPDATE_API_URL="+srv.URL,
		"KUBERNETES_SERVICE_HOST=",
	)
	out2, err := cmd2.CombinedOutput()
	if err != nil {
		t.Fatalf("expected exit code 0 when current, got err=%v, out:\n%s", err, out2)
	}
	if !strings.Contains(string(out2), "0.2.0") {
		t.Fatalf("expected output to report 0.2.0, got:\n%s", out2)
	}
}

func TestSelfUpdate_TamperedArchiveAborts(t *testing.T) {
	archiveData := []byte("tampered-payload")
	srv := newFakeReleaseServer(t, []fakeReleaseSpec{
		{
			tag:         "v0.2.0",
			archiveData: archiveData,
			checksum:    strings.Repeat("0", 64),
		},
	}, runtime.GOOS, runtime.GOARCH)

	dir := t.TempDir()
	binName := "opus"
	if runtime.GOOS == "windows" {
		binName = "opus.exe"
	}
	binPath := filepath.Join(dir, binName)
	origContent := []byte("original-binary-content")
	if err := os.WriteFile(binPath, origContent, 0o755); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	deps := selfUpdateDeps{
		in:             strings.NewReader(""),
		out:            &out,
		httpClient:     srv.Client(),
		apiBaseURL:     srv.URL,
		currentVersion: "0.1.0",
		goos:           runtime.GOOS,
		goarch:         runtime.GOARCH,
		executable:     func() (string, error) { return binPath, nil },
		evalSymlinks:   filepath.EvalSymlinks,
		homeDir:        func() (string, error) { return filepath.Join(dir, "nonexistent-home"), nil },
		getenv:         func(string) string { return "" },
		stat:           os.Stat,
		checkWritable:  checkBinaryWritable,
		replace:        replaceBinary,
		dockerEnvPath:  filepath.Join(dir, "nonexistent-dockerenv"),
	}

	err := runSelfUpdate(context.Background(), selfUpdateOptions{yes: true}, deps)
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("expected checksum mismatch error, got: %v", err)
	}

	got, readErr := os.ReadFile(binPath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !bytes.Equal(got, origContent) {
		t.Fatalf("expected original binary to remain untouched")
	}
}

func TestSelfUpdate_SuccessAndMidpointExtractionFailure(t *testing.T) {
	newBinDir := t.TempDir()
	newBinPath := buildTestOpusBinaryWithVersion(t, newBinDir, "0.2.0")
	validArchive := makeTestArchive(t, newBinPath, runtime.GOOS)

	// AC3 part 1: midpoint extraction failure (corrupt archive with matching checksum) leaves old binary intact.
	corruptArchive := validArchive[:len(validArchive)/2]
	corruptSrv := newFakeReleaseServer(t, []fakeReleaseSpec{
		{tag: "v0.2.0", archiveData: corruptArchive},
	}, runtime.GOOS, runtime.GOARCH)

	installDir := t.TempDir()
	installedBin := buildTestOpusBinaryWithVersion(t, installDir, "0.1.0")
	origBytes, err := os.ReadFile(installedBin)
	if err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	deps := selfUpdateDeps{
		in:             strings.NewReader(""),
		out:            &out,
		httpClient:     corruptSrv.Client(),
		apiBaseURL:     corruptSrv.URL,
		currentVersion: "0.1.0",
		goos:           runtime.GOOS,
		goarch:         runtime.GOARCH,
		executable:     func() (string, error) { return installedBin, nil },
		evalSymlinks:   filepath.EvalSymlinks,
		homeDir:        func() (string, error) { return filepath.Join(installDir, "nonexistent-home"), nil },
		getenv:         func(string) string { return "" },
		stat:           os.Stat,
		checkWritable:  checkBinaryWritable,
		replace:        replaceBinary,
		dockerEnvPath:  filepath.Join(installDir, "nonexistent-dockerenv"),
	}
	if err := runSelfUpdate(context.Background(), selfUpdateOptions{yes: true}, deps); err == nil {
		t.Fatal("expected extraction failure on truncated archive")
	}
	afterCorruptBytes, err := os.ReadFile(installedBin)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(origBytes, afterCorruptBytes) {
		t.Fatal("expected old binary to remain intact after midpoint extraction failure")
	}

	// AC3 part 2: successful binary replacement -> executing new binary --version prints 0.2.0.
	validSrv := newFakeReleaseServer(t, []fakeReleaseSpec{
		{tag: "v0.2.0", archiveData: validArchive},
	}, runtime.GOOS, runtime.GOARCH)

	cmd := exec.Command(installedBin, "self-update", "--yes")
	cmd.Env = append(os.Environ(),
		"TEST_SELF_UPDATE_API_URL="+validSrv.URL,
		"APP_HOME="+filepath.Join(installDir, "nonexistent-home"),
		"APP_DATABASE_URL=",
		"KUBERNETES_SERVICE_HOST=",
	)
	updateOut, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("opus self-update --yes failed: %v\n%s", err, updateOut)
	}
	if !strings.Contains(string(updateOut), "restart opus to use the new version") {
		t.Fatalf("expected restart message, got:\n%s", updateOut)
	}

	verOut, err := exec.Command(installedBin, "--version").CombinedOutput()
	if err != nil {
		t.Fatalf("replaced binary --version failed: %v\n%s", err, verOut)
	}
	if !strings.Contains(string(verOut), "0.2.0") {
		t.Fatalf("expected replaced binary to print 0.2.0, got:\n%s", verOut)
	}
}

func TestSelfUpdate_DowngradeRequiresYes(t *testing.T) {
	oldBinPath := buildTestOpusBinaryWithVersion(t, t.TempDir(), "0.1.0")
	oldArchive := makeTestArchive(t, oldBinPath, runtime.GOOS)

	srv := newFakeReleaseServer(t, []fakeReleaseSpec{
		{tag: "v0.1.0", archiveData: oldArchive},
		{tag: "v0.2.0", archiveData: []byte("newer")},
	}, runtime.GOOS, runtime.GOARCH)

	installDir := t.TempDir()
	installedBin := buildTestOpusBinaryWithVersion(t, installDir, "0.2.0")

	var out bytes.Buffer
	deps := selfUpdateDeps{
		in:             strings.NewReader(""),
		out:            &out,
		httpClient:     srv.Client(),
		apiBaseURL:     srv.URL,
		currentVersion: "0.2.0",
		goos:           runtime.GOOS,
		goarch:         runtime.GOARCH,
		executable:     func() (string, error) { return installedBin, nil },
		evalSymlinks:   filepath.EvalSymlinks,
		homeDir:        func() (string, error) { return filepath.Join(installDir, "nonexistent-home"), nil },
		getenv:         func(string) string { return "" },
		stat:           os.Stat,
		checkWritable:  checkBinaryWritable,
		replace:        replaceBinary,
		dockerEnvPath:  filepath.Join(installDir, "nonexistent-dockerenv"),
	}

	// AC5: without --yes, downgrade must be refused.
	err := runSelfUpdate(context.Background(), selfUpdateOptions{version: "0.1.0", yes: false}, deps)
	if err == nil || !strings.Contains(err.Error(), "--yes") {
		t.Fatalf("expected downgrade refusal requiring --yes, got: %v", err)
	}

	// AC5: with --yes, downgrade succeeds.
	if err := runSelfUpdate(context.Background(), selfUpdateOptions{version: "0.1.0", yes: true}, deps); err != nil {
		t.Fatalf("expected downgrade with --yes to succeed, got: %v", err)
	}
	verOut, err := exec.Command(installedBin, "--version").CombinedOutput()
	if err != nil {
		t.Fatalf("downgraded binary --version failed: %v\n%s", err, verOut)
	}
	if !strings.Contains(string(verOut), "0.1.0") {
		t.Fatalf("expected downgraded binary to print 0.1.0, got:\n%s", verOut)
	}
}

func TestSelfUpdate_PrereleaseHandling(t *testing.T) {
	// AC6: 0.3.0-rc2 -> 0.3.0-rc3, and 0.3.0 > 0.3.0-rc*
	vRC2, err := parseSemver("0.3.0-rc2")
	if err != nil {
		t.Fatal(err)
	}
	vRC3, err := parseSemver("v0.3.0-rc3")
	if err != nil {
		t.Fatal(err)
	}
	vStable, err := parseSemver("0.3.0")
	if err != nil {
		t.Fatal(err)
	}
	if compareSemver(vRC2, vRC3) >= 0 {
		t.Fatalf("expected 0.3.0-rc2 < 0.3.0-rc3")
	}
	if compareSemver(vRC3, vStable) >= 0 {
		t.Fatalf("expected 0.3.0-rc3 < 0.3.0")
	}

	// Server has 0.3.0-rc2 and 0.3.0-rc3; current is 0.3.0-rc2 -> --check picks 0.3.0-rc3 and exits 10.
	srv := newFakeReleaseServer(t, []fakeReleaseSpec{
		{tag: "v0.3.0-rc2", prerelease: true, archiveData: []byte("rc2")},
		{tag: "v0.3.0-rc3", prerelease: true, archiveData: []byte("rc3")},
	}, runtime.GOOS, runtime.GOARCH)

	var out bytes.Buffer
	deps := selfUpdateDeps{
		out:            &out,
		httpClient:     srv.Client(),
		apiBaseURL:     srv.URL,
		currentVersion: "0.3.0-rc2",
		goos:           runtime.GOOS,
		goarch:         runtime.GOARCH,
		getenv:         func(string) string { return "" },
		stat:           os.Stat,
		dockerEnvPath:  filepath.Join(t.TempDir(), "nonexistent-dockerenv"),
	}
	err = runSelfUpdate(context.Background(), selfUpdateOptions{check: true}, deps)
	var exitErr exitCodeError
	if !errors.As(err, &exitErr) || exitErr.code != 10 {
		t.Fatalf("expected exit code 10 for 0.3.0-rc2 -> 0.3.0-rc3, got: %v", err)
	}
	if !strings.Contains(out.String(), "0.3.0-rc3") {
		t.Fatalf("expected 0.3.0-rc3 in output, got:\n%s", out.String())
	}
}

func TestSelfUpdate_ContainerRefused(t *testing.T) {
	dockerEnv := filepath.Join(t.TempDir(), ".dockerenv")
	if err := os.WriteFile(dockerEnv, []byte(""), 0o600); err != nil {
		t.Fatal(err)
	}

	// AC7: /.dockerenv present
	deps := selfUpdateDeps{
		out:           &bytes.Buffer{},
		getenv:        func(string) string { return "" },
		stat:          os.Stat,
		dockerEnvPath: dockerEnv,
	}
	err := runSelfUpdate(context.Background(), selfUpdateOptions{yes: true}, deps)
	if err == nil || !strings.Contains(err.Error(), "pull a new container image") {
		t.Fatalf("expected container refusal message, got: %v", err)
	}

	// AC7: KUBERNETES_SERVICE_HOST set
	deps2 := selfUpdateDeps{
		out:           &bytes.Buffer{},
		getenv:        func(k string) string { return map[string]string{"KUBERNETES_SERVICE_HOST": "10.0.0.1"}[k] },
		stat:          os.Stat,
		dockerEnvPath: filepath.Join(t.TempDir(), "nonexistent-dockerenv"),
	}
	err2 := runSelfUpdate(context.Background(), selfUpdateOptions{yes: true}, deps2)
	if err2 == nil || !strings.Contains(err2.Error(), "pull a new container image") {
		t.Fatalf("expected k8s container refusal message, got: %v", err2)
	}
}

func TestSelfUpdate_NotWritableAndBackupPrompt(t *testing.T) {
	srv := newFakeReleaseServer(t, []fakeReleaseSpec{
		{tag: "v0.2.0", archiveData: []byte("dummy")},
	}, runtime.GOOS, runtime.GOARCH)

	dir := t.TempDir()
	binPath := filepath.Join(dir, "opus")

	// Not writable binary path
	depsNotWritable := selfUpdateDeps{
		out:            &bytes.Buffer{},
		httpClient:     srv.Client(),
		apiBaseURL:     srv.URL,
		currentVersion: "0.1.0",
		goos:           runtime.GOOS,
		goarch:         runtime.GOARCH,
		executable:     func() (string, error) { return binPath, nil },
		evalSymlinks:   func(p string) (string, error) { return p, nil },
		getenv:         func(string) string { return "" },
		stat:           os.Stat,
		checkWritable:  func(string) error { return os.ErrPermission },
		dockerEnvPath:  filepath.Join(dir, "nonexistent-dockerenv"),
	}
	err := runSelfUpdate(context.Background(), selfUpdateOptions{yes: true}, depsNotWritable)
	if err == nil || !strings.Contains(err.Error(), binPath) || !strings.Contains(err.Error(), "install script") {
		t.Fatalf("expected not-writable error with path and install script hint, got: %v", err)
	}

	// Existing SQLite DB triggers backup reminder and cancellation when user answers "n"
	dbPath := filepath.Join(dir, "opus.db")
	if err := os.WriteFile(dbPath, []byte("sqlite"), 0o600); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	depsBackup := selfUpdateDeps{
		in:             strings.NewReader("n\n"),
		out:            &out,
		httpClient:     srv.Client(),
		apiBaseURL:     srv.URL,
		currentVersion: "0.1.0",
		goos:           runtime.GOOS,
		goarch:         runtime.GOARCH,
		executable:     func() (string, error) { return binPath, nil },
		evalSymlinks:   func(p string) (string, error) { return p, nil },
		homeDir:        func() (string, error) { return dir, nil },
		getenv:         func(string) string { return "" },
		stat:           os.Stat,
		checkWritable:  func(string) error { return nil },
		dockerEnvPath:  filepath.Join(dir, "nonexistent-dockerenv"),
	}
	if err := runSelfUpdate(context.Background(), selfUpdateOptions{yes: false}, depsBackup); err != nil {
		t.Fatalf("expected clean cancellation, got: %v", err)
	}
	if !strings.Contains(out.String(), "docs/backup.md") || !strings.Contains(out.String(), "Update cancelled.") {
		t.Fatalf("expected backup reminder and cancellation message, got:\n%s", out.String())
	}
}
