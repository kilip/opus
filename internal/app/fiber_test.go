package app

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"testing/fstest"
	"time"

	"github.com/kilip/opus/internal/platform/eventbus"
)

func testAppConfig(t *testing.T) Config {
	t.Helper()
	return Config{
		AppName:         "opus-test",
		AppHome:         t.TempDir(),
		Host:            "127.0.0.1",
		Port:            "0",
		ReadTimeout:     5 * time.Second,
		WriteTimeout:    5 * time.Second,
		IdleTimeout:     5 * time.Second,
		ShutdownTimeout: 100 * time.Millisecond,
		BodyLimit:       4 * 1024 * 1024,
		BaseURL:         "http://localhost:7880",
		SessionSecret:   "0123456789abcdef0123456789abcdef",
		SessionTTL:      time.Hour,
	}
}

func TestReadyz_AndLoggerSkip(t *testing.T) {
	ctx := context.Background()
	dbURL := "file:" + filepath.Join(t.TempDir(), "readyz.db")
	db, err := OpenDBWithVersion(ctx, dbURL, "0.1.0", false)
	if err != nil {
		t.Fatalf("OpenDBWithVersion: %v", err)
	}

	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, nil))
	shuttingDown := &atomic.Bool{}
	bus := eventbus.New(eventbus.WithLogger(logger))
	defer func() { _ = bus.Close(ctx) }()

	app, err := GetFiber(testAppConfig(t), logger, db, bus, WithShuttingDown(shuttingDown))
	if err != nil {
		t.Fatalf("GetFiber: %v", err)
	}

	// /api/readyz returns 200 and skips request log.
	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/readyz", nil))
	if err != nil {
		t.Fatalf("GET /api/readyz: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("readyz status = %d, want 200", resp.StatusCode)
	}

	// /api/health returns 200 and skips request log.
	respH, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/health", nil))
	if err != nil {
		t.Fatalf("GET /api/health: %v", err)
	}
	_ = respH.Body.Close()
	if respH.StatusCode != http.StatusOK {
		t.Fatalf("health status = %d, want 200", respH.StatusCode)
	}

	if strings.Contains(logBuf.String(), "/api/health") || strings.Contains(logBuf.String(), "/api/readyz") {
		t.Fatalf("expected /api/health and /api/readyz to be skipped in logger, got:\n%s", logBuf.String())
	}

	// Non-health request is logged.
	respNotFound, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/auth/me", nil))
	if err != nil {
		t.Fatalf("GET /api/auth/me: %v", err)
	}
	_ = respNotFound.Body.Close()
	if !strings.Contains(logBuf.String(), "/api/auth/me") {
		t.Fatalf("expected /api/auth/me to be logged, got:\n%s", logBuf.String())
	}

	// Shutting down flag -> /api/readyz returns 503, /api/health still returns 200.
	shuttingDown.Store(true)
	respDown, _ := app.Test(httptest.NewRequest(http.MethodGet, "/api/readyz", nil))
	_ = respDown.Body.Close()
	if respDown.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("readyz while shutting down = %d, want 503", respDown.StatusCode)
	}

	// Closed DB -> /api/readyz returns 503.
	shuttingDown.Store(false)
	_ = db.Close()
	respClosed, _ := app.Test(httptest.NewRequest(http.MethodGet, "/api/readyz", nil))
	_ = respClosed.Body.Close()
	if respClosed.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("readyz with closed DB = %d, want 503", respClosed.StatusCode)
	}
}

func TestSecurityHeaders_AndSPAFallback(t *testing.T) {
	ctx := context.Background()
	dbURL := "file:" + filepath.Join(t.TempDir(), "headers.db")
	db, err := OpenDBWithVersion(ctx, dbURL, "0.1.0", false)
	if err != nil {
		t.Fatalf("OpenDBWithVersion: %v", err)
	}
	defer func() { _ = db.Close() }()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	bus := eventbus.New(eventbus.WithLogger(logger))
	defer func() { _ = bus.Close(ctx) }()

	// HTTP BaseURL -> no HSTS header.
	mockDist := fstest.MapFS{
		"index.html":    {Data: []byte("<!doctype html><html><body>Opus</body></html>")},
		"assets/app.js": {Data: []byte("console.log('ok')")},
	}
	appHTTP, err := GetFiber(testAppConfig(t), logger, db, bus, WithDistFS(mockDist))
	if err != nil {
		t.Fatalf("GetFiber HTTP: %v", err)
	}
	resp, err := appHTTP.Test(httptest.NewRequest(http.MethodGet, "/config.js", nil))
	if err != nil {
		t.Fatalf("GET /config.js: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("/config.js status = %d, want 200", resp.StatusCode)
	}
	if got := resp.Header.Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("X-Content-Type-Options = %q, want nosniff", got)
	}
	if got := resp.Header.Get("X-Frame-Options"); got != "DENY" {
		t.Fatalf("X-Frame-Options = %q, want DENY", got)
	}
	if got := resp.Header.Get("Referrer-Policy"); got != "strict-origin-when-cross-origin" {
		t.Fatalf("Referrer-Policy = %q", got)
	}
	if got := resp.Header.Get("Content-Security-Policy"); !strings.Contains(got, "script-src 'self'") {
		t.Fatalf("CSP missing script-src 'self': %q", got)
	}
	if got := resp.Header.Get("Strict-Transport-Security"); got != "" {
		t.Fatalf("expected no HSTS on http://, got %q", got)
	}

	// Missing asset with extension returns 404.
	respMissing, err := appHTTP.Test(httptest.NewRequest(http.MethodGet, "/assets/does-not-exist.js", nil))
	if err != nil {
		t.Fatalf("GET missing asset: %v", err)
	}
	_ = respMissing.Body.Close()
	if respMissing.StatusCode != http.StatusNotFound {
		t.Fatalf("missing asset status = %d, want 404", respMissing.StatusCode)
	}

	// Extensionless client route returns 200 index.html.
	respSPA, err := appHTTP.Test(httptest.NewRequest(http.MethodGet, "/tasks/123", nil))
	if err != nil {
		t.Fatalf("GET /tasks/123: %v", err)
	}
	spaBody, _ := io.ReadAll(respSPA.Body)
	_ = respSPA.Body.Close()
	if respSPA.StatusCode != http.StatusOK {
		t.Fatalf("SPA route status = %d, want 200 (body=%q)", respSPA.StatusCode, string(spaBody))
	}

	// HTTPS BaseURL -> HSTS header present.
	httpsCfg := testAppConfig(t)
	httpsCfg.BaseURL = "https://opus.example.com"
	appHTTPS, err := GetFiber(httpsCfg, logger, db, bus)
	if err != nil {
		t.Fatalf("GetFiber HTTPS: %v", err)
	}
	respHTTPS, _ := appHTTPS.Test(httptest.NewRequest(http.MethodGet, "https://opus.example.com/api/health", nil))
	_ = respHTTPS.Body.Close()
	if got := respHTTPS.Header.Get("Strict-Transport-Security"); got == "" {
		t.Fatal("expected Strict-Transport-Security on https:// BaseURL")
	}
}
