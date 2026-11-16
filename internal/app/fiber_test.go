package app_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/gofiber/fiber/v3"
	"github.com/kilip/opus/ent"
	"github.com/kilip/opus/ent/enttest"
	"github.com/kilip/opus/internal/app"
	"github.com/kilip/opus/internal/platform/eventbus"
	_ "modernc.org/sqlite"
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
	fiberApp.Post("/api/test-csrf", func(c fiber.Ctx) error {
		return c.SendStatus(http.StatusCreated)
	})

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
		req := httptest.NewRequest("POST", "/api/test-csrf", payload)
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
		req := httptest.NewRequest("POST", "/api/test-csrf", payload)
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
		req := httptest.NewRequest("POST", "/api/test-csrf", payload)
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
		req := httptest.NewRequest("POST", "/api/test-csrf", payload)
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
		req := httptest.NewRequest("POST", "/api/test-csrf", payload)
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

// AC3: task.New(db, bus).Register(api) is mounted inside `if db != nil` behind `api.Use(authFeature.Middleware())`.
func TestGetFiber_TasksRequiresAuth(t *testing.T) {
	db, err := sql.Open("sqlite", "file:"+t.Name()+"?mode=memory&cache=shared&_pragma=foreign_keys(ON)&_pragma=busy_timeout(10000)")
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	db.SetMaxOpenConns(1)
	drv := entsql.OpenDB(dialect.SQLite, db)
	client := enttest.NewClient(t, enttest.WithOptions(ent.Driver(drv)))
	t.Cleanup(func() {
		_ = client.Close()
	})

	cfg := app.Config{
		AppName:       "test-opus",
		BaseURL:       "http://localhost:7880",
		SessionSecret: "secret-needs-to-be-32-bytes-long-ok",
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	fiberApp := app.GetFiber(cfg, logger, client, eventbus.New())

	// 1. Health check is public (200)
	reqHealth := httptest.NewRequest("GET", "/api/health", nil)
	respHealth, err := fiberApp.Test(reqHealth)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if respHealth.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for /api/health, got %d", respHealth.StatusCode)
	}

	// 2. Unauthenticated GET /api/tasks returns 401
	reqTasks := httptest.NewRequest("GET", "/api/tasks", nil)
	respTasks, err := fiberApp.Test(reqTasks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if respTasks.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 for /api/tasks, got %d", respTasks.StatusCode)
	}
	body := parseResponseBody(t, respTasks)
	if body.Error != "UNAUTHORIZED" {
		t.Errorf("expected error %q, got %q", "UNAUTHORIZED", body.Error)
	}
}
