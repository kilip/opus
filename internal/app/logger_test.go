package app

import (
	"context"
	"io"
	"log/slog"
	"net/http/httptest"
	"testing"
)

func TestGetConfigAndLogger(t *testing.T) {
	t.Setenv("LOG_LEVEL", "debug")
	cfg, err := GetConfig(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.LogLevel != slog.LevelDebug {
		t.Fatalf("expected DEBUG level, got %v", cfg.LogLevel)
	}

	logger := GetLogger(cfg)
	if !logger.Enabled(context.Background(), slog.LevelDebug) {
		t.Fatal("expected debug level enabled")
	}
	if slog.Default() != logger {
		t.Fatal("expected default logger to be set")
	}
}

func TestGetFiber(t *testing.T) {
	cfg, err := GetConfig(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	f := GetFiber(cfg, GetLogger(cfg))

	req := httptest.NewRequest("GET", "/api/health", nil)
	resp, err := f.Test(req)
	if err != nil {
		t.Fatalf("unexpected test error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
	if resp.Header.Get("X-Request-ID") == "" {
		t.Fatal("expected X-Request-ID response header")
	}
	body, _ := io.ReadAll(resp.Body)
	expectedBody := `{"data":{"status":"ok"}}`
	if string(body) != expectedBody {
		t.Fatalf("expected body %s, got %s", expectedBody, string(body))
	}

	apiMissing := httptest.NewRequest("GET", "/api/unknown", nil)
	apiResp, err := f.Test(apiMissing)
	if err != nil {
		t.Fatalf("unexpected test error: %v", err)
	}
	defer func() { _ = apiResp.Body.Close() }()
	if apiResp.StatusCode != 404 {
		t.Fatalf("expected 404 for unknown /api route, got %d", apiResp.StatusCode)
	}
}
