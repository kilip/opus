package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/kilip/opus/internal/app"
	"github.com/kilip/opus/internal/worker"
	_ "modernc.org/sqlite"
)

func newHealthyDoctorDeps(t *testing.T) doctorDeps {
	t.Helper()
	appHome := t.TempDir()
	dbPath := filepath.Join(appHome, "opus.db")
	dbURL := "file:" + filepath.ToSlash(dbPath) + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"

	db, err := sql.Open("sqlite", dbURL)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`PRAGMA journal_mode=WAL; PRAGMA busy_timeout=5000; CREATE TABLE users (id TEXT PRIMARY KEY); INSERT INTO users (id) VALUES ('u1');`); err != nil {
		_ = db.Close()
		t.Fatal(err)
	}
	_ = db.Close()

	ghSrv := newFakeReleaseServer(t, []fakeReleaseSpec{
		{tag: "v0.1.0", archiveData: []byte("ok")},
	}, runtime.GOOS, runtime.GOARCH)

	tgSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/getMe") {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"ok":true}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(tgSrv.Close)

	return doctorDeps{
		out: &bytes.Buffer{},
		cfg: app.Config{
			AppHome:                appHome,
			Host:                   "127.0.0.1",
			Port:                   "0",
			DatabaseURL:            dbURL,
			BaseURL:                "http://localhost:7880",
			SessionSecret:          "0123456789abcdef0123456789abcdef",
			AuthCookieSecure:       "auto",
			NotifyTelegramBotToken: "123456:ABC-fake-tg-token-999",
		},
		currentVersion:  "0.1.0",
		goos:            runtime.GOOS,
		goarch:          runtime.GOARCH,
		apiBaseURL:      ghSrv.URL,
		telegramBaseURL: tgSrv.URL,
		httpClient:      &http.Client{Timeout: doctorTimeout},
		homeDir:         func() (string, error) { return appHome, nil },
		getenv:          func(string) string { return "" },
		stat:            os.Stat,
		lookPath:        func(bin string) (string, error) { return "/usr/bin/" + bin, nil },
		runCmd:          func(context.Context, string, ...string) ([]byte, error) { return []byte("logged in"), nil },
		probeAgy:        func(context.Context, string) worker.ToolStatus { return worker.ToolStatusReady },
		dialTimeout:     func(string, string, time.Duration) (net.Conn, error) { return nil, errors.New("unused") },
		listen: func(network, address string) (net.Listener, error) {
			return net.Listen(network, "127.0.0.1:0")
		},
		serviceStatus: func(context.Context) (int, error) { return serviceStatusNotInstalled, nil },
	}
}

func TestDoctor_HealthySetupPassesAndJSONValid(t *testing.T) {
	var buf bytes.Buffer
	deps := newHealthyDoctorDeps(t)
	deps.out = &buf

	// AC1: healthy setup exits 0 with PASS
	if err := runDoctor(context.Background(), doctorOptions{jsonOut: false}, deps); err != nil {
		t.Fatalf("expected healthy doctor to exit 0, got: %v\n%s", err, buf.String())
	}
	if strings.Contains(buf.String(), "[FAIL]") {
		t.Fatalf("expected no FAIL checks in healthy setup:\n%s", buf.String())
	}

	// AC3: --json outputs valid JSON with name, status, message, hint
	var jsonBuf bytes.Buffer
	deps.out = &jsonBuf
	if err := runDoctor(context.Background(), doctorOptions{jsonOut: true}, deps); err != nil {
		t.Fatalf("doctor --json failed: %v", err)
	}
	var results []doctorResult
	if err := json.Unmarshal(jsonBuf.Bytes(), &results); err != nil {
		t.Fatalf("invalid JSON output: %v\n%s", err, jsonBuf.String())
	}
	if len(results) < 10 {
		t.Fatalf("expected >= 10 checks, got %d", len(results))
	}
	for _, r := range results {
		if r.Name == "" || r.Status == "" || r.Message == "" {
			t.Fatalf("incomplete doctorResult: %+v", r)
		}
	}
}

func TestDoctor_MissingToolsAndShortSecret(t *testing.T) {
	const shortSecret = "8bytess!"
	const tgSecret = "999999:SUPER-SECRET-TG-TOKEN"
	const ghSecret = "ghp_SUPER_SECRET_GH_TOKEN_42"

	// Hanging HTTP server for Telegram to test error scrubbing & <= 3s timeout (AC4, AC5)
	hangCh := make(chan struct{})
	t.Cleanup(func() { close(hangCh) })
	hangSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-hangCh:
		case <-r.Context().Done():
		}
	}))
	t.Cleanup(hangSrv.Close)

	deps := newHealthyDoctorDeps(t)
	deps.cfg.SessionSecret = shortSecret
	deps.cfg.NotifyTelegramBotToken = tgSecret
	deps.telegramBaseURL = hangSrv.URL
	deps.getenv = func(k string) string {
		if k == "GH_TOKEN" {
			return ghSecret
		}
		return ""
	}
	deps.lookPath = func(bin string) (string, error) {
		if bin == "gh" {
			return "", errors.New("not found")
		}
		return "/usr/bin/" + bin, nil
	}
	deps.probeAgy = func(context.Context, string) worker.ToolStatus {
		return worker.ToolStatusMissing
	}

	var buf bytes.Buffer
	deps.out = &buf
	start := time.Now()
	err := runDoctor(context.Background(), doctorOptions{jsonOut: false}, deps)
	elapsed := time.Since(start)

	// AC5: network timeout must complete within <= 4s (3s timeout)
	if elapsed > 4*time.Second {
		t.Fatalf("doctor took too long (%v); expected <= 3s timeout", elapsed)
	}

	// AC2: exit code 1 for FAIL
	var exitErr exitCodeError
	if !errors.As(err, &exitErr) || exitErr.code != 1 {
		t.Fatalf("expected exitCodeError(1), got: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "[FAIL] gh:") || !strings.Contains(out, "[WARN] agy:") || !strings.Contains(out, "[FAIL] auth_secret:") {
		t.Fatalf("expected gh FAIL, agy WARN, auth_secret FAIL in output:\n%s", out)
	}

	// AC4: secrets MUST NOT appear anywhere in output even on error!
	for _, secret := range []string{shortSecret, tgSecret, ghSecret} {
		if strings.Contains(out, secret) {
			t.Fatalf("secret %q leaked into doctor output:\n%s", secret, out)
		}
	}
}
