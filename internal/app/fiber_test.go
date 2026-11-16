package app_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kilip/opus/internal/app"
	"github.com/kilip/opus/internal/platform/eventbus"
)

func parseResponseBody(t *testing.T, resp *http.Response) struct {
	Data  any    `json:"data"`
	Error string `json:"error"`
} {
	t.Helper()
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read body: %v", err)
	}
	var res struct {
		Data  any    `json:"data"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(body, &res); err != nil {
		t.Fatalf("failed to unmarshal body: %v (raw: %s)", err, string(body))
	}
	return res
}

func TestGetFiber_CSRFProtection(t *testing.T) {
	cfg := app.Config{
		AppName:        "test-opus",
		BaseURL:        "http://localhost:7880",
		TrustedOrigins: []string{"http://localhost:5173"},
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	fiberApp := app.GetFiber(cfg, logger, nil, eventbus.New())

	// Safe request (GET) with cross-origin Origin must pass through
	t.Run("GET safe method allowed even with cross-origin", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/health", nil)
		req.Header.Set("Origin", "https://evil.com")
		resp, err := fiberApp.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected status 200, got %d", resp.StatusCode)
		}
	})

	// Mutating request (POST) with cross-origin Origin must be rejected with 403 CSRF_FORBIDDEN
	t.Run("POST cross-origin rejected with 403 CSRF_FORBIDDEN", func(t *testing.T) {
		payload := bytes.NewReader([]byte(`{"title":"csrf task"}`))
		req := httptest.NewRequest("POST", "/api/tasks", payload)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", "https://evil.com")
		resp, err := fiberApp.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("expected status 403, got %d", resp.StatusCode)
		}
		body := parseResponseBody(t, resp)
		if body.Error != "CSRF_FORBIDDEN" {
			t.Errorf("expected error %q, got %q", "CSRF_FORBIDDEN", body.Error)
		}
	})

	// Mutating request (POST) with same-origin Origin must pass
	t.Run("POST same-origin allowed", func(t *testing.T) {
		payload := bytes.NewReader([]byte(`{"title":"same origin task"}`))
		req := httptest.NewRequest("POST", "/api/tasks", payload)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", "http://localhost:7880")
		resp, err := fiberApp.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("expected status 201, got %d", resp.StatusCode)
		}
	})

	// Mutating request (POST) with trusted origin must pass
	t.Run("POST trusted origin allowed", func(t *testing.T) {
		payload := bytes.NewReader([]byte(`{"title":"trusted origin task"}`))
		req := httptest.NewRequest("POST", "/api/tasks", payload)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", "http://localhost:5173")
		resp, err := fiberApp.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("expected status 201, got %d", resp.StatusCode)
		}
	})

	// Mutating request (POST) with Sec-Fetch-Site: cross-site rejected
	t.Run("POST Sec-Fetch-Site cross-site rejected", func(t *testing.T) {
		payload := bytes.NewReader([]byte(`{"title":"sec fetch site task"}`))
		req := httptest.NewRequest("POST", "/api/tasks", payload)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Sec-Fetch-Site", "cross-site")
		resp, err := fiberApp.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("expected status 403, got %d", resp.StatusCode)
		}
		body := parseResponseBody(t, resp)
		if body.Error != "CSRF_FORBIDDEN" {
			t.Errorf("expected error %q, got %q", "CSRF_FORBIDDEN", body.Error)
		}
	})

	// Mutating request (POST) with Sec-Fetch-Site: same-origin allowed
	t.Run("POST Sec-Fetch-Site same-origin allowed", func(t *testing.T) {
		payload := bytes.NewReader([]byte(`{"title":"sec fetch site same-origin task"}`))
		req := httptest.NewRequest("POST", "/api/tasks", payload)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Sec-Fetch-Site", "same-origin")
		resp, err := fiberApp.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("expected status 201, got %d", resp.StatusCode)
		}
	})
}
