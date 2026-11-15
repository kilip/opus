package csrf_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/opus/internal/platform/csrf"
	"github.com/kilip/opus/internal/platform/httpx"
)

func setupTestApp(cfg csrf.Config) *fiber.App {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	app := fiber.New(fiber.Config{
		ErrorHandler: httpx.ErrorHandler(logger),
	})
	app.Use(csrf.New(cfg))

	handler := func(c fiber.Ctx) error {
		return httpx.OK(c, fiber.Map{"status": "success"})
	}

	app.Get("/test", handler)
	app.Head("/test", handler)
	app.Post("/test", handler)
	app.Put("/test", handler)
	app.Patch("/test", handler)
	app.Delete("/test", handler)

	return app
}

func parseError(t *testing.T, resp *http.Response) string {
	t.Helper()
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	var res struct {
		Data  any    `json:"data"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(body, &res); err != nil {
		t.Fatalf("failed to parse json response: %v (body: %s)", err, string(body))
	}
	return res.Error
}

func TestSafeMethods_Unaffected(t *testing.T) {
	app := setupTestApp(csrf.Config{
		BaseURL: "http://localhost:7880",
	})

	for _, method := range []string{"GET", "HEAD"} {
		req := httptest.NewRequest(method, "/test", nil)
		req.Header.Set("Origin", "https://evil.com")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("%s /test failed: %v", method, err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("%s /test got status %d, want %d", method, resp.StatusCode, http.StatusOK)
		}
	}
}

func TestMutating_SameOrigin_Allowed(t *testing.T) {
	app := setupTestApp(csrf.Config{
		BaseURL: "http://localhost:7880",
	})

	mutatingMethods := []string{"POST", "PUT", "PATCH", "DELETE"}
	for _, method := range mutatingMethods {
		req := httptest.NewRequest(method, "/test", nil)
		req.Header.Set("Origin", "http://localhost:7880")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("%s /test failed: %v", method, err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("%s /test same-origin got status %d, want %d", method, resp.StatusCode, http.StatusOK)
		}
	}
}

func TestMutating_CrossOrigin_Rejected(t *testing.T) {
	app := setupTestApp(csrf.Config{
		BaseURL: "http://localhost:7880",
	})

	mutatingMethods := []string{"POST", "PUT", "PATCH", "DELETE"}
	for _, method := range mutatingMethods {
		req := httptest.NewRequest(method, "/test", nil)
		req.Header.Set("Origin", "https://evil.com")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("%s /test failed: %v", method, err)
		}
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("%s /test cross-origin got status %d, want %d", method, resp.StatusCode, http.StatusForbidden)
		}
		errCode := parseError(t, resp)
		if errCode != "CSRF_FORBIDDEN" {
			t.Errorf("%s /test got error code %q, want %q", method, errCode, "CSRF_FORBIDDEN")
		}
	}
}

func TestMutating_TrustedOrigin_Allowed(t *testing.T) {
	app := setupTestApp(csrf.Config{
		BaseURL:        "http://localhost:7880",
		TrustedOrigins: []string{"http://localhost:5173", "http://localhost:3000"},
	})

	for _, origin := range []string{"http://localhost:5173", "http://localhost:3000"} {
		req := httptest.NewRequest("POST", "/test", nil)
		req.Header.Set("Origin", origin)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("POST /test with trusted origin %s failed: %v", origin, err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("trusted origin %s got status %d, want %d", origin, resp.StatusCode, http.StatusOK)
		}
	}
}

func TestMutating_SecFetchSite(t *testing.T) {
	app := setupTestApp(csrf.Config{
		BaseURL: "http://localhost:7880",
	})

	tests := []struct {
		name       string
		headerVal  string
		wantStatus int
	}{
		{
			name:       "cross-site rejected",
			headerVal:  "cross-site",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "same-origin allowed",
			headerVal:  "same-origin",
			wantStatus: http.StatusOK,
		},
		{
			name:       "none allowed",
			headerVal:  "none",
			wantStatus: http.StatusOK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/test", nil)
			req.Header.Set("Sec-Fetch-Site", tc.headerVal)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			if resp.StatusCode != tc.wantStatus {
				t.Errorf("got status %d, want %d", resp.StatusCode, tc.wantStatus)
			}
			if tc.wantStatus == http.StatusForbidden {
				errCode := parseError(t, resp)
				if errCode != "CSRF_FORBIDDEN" {
					t.Errorf("got error code %q, want %q", errCode, "CSRF_FORBIDDEN")
				}
			}
		})
	}
}

func TestMutating_NoOriginHeaders_Allowed(t *testing.T) {
	app := setupTestApp(csrf.Config{
		BaseURL: "http://localhost:7880",
	})

	req := httptest.NewRequest("POST", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("got status %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestOriginNormalization(t *testing.T) {
	app := setupTestApp(csrf.Config{
		BaseURL:        "http://localhost:7880/",
		TrustedOrigins: []string{"https://example.com:443/", "http://test.local:80"},
	})

	tests := []struct {
		name       string
		origin     string
		wantStatus int
	}{
		{
			name:       "base URL without trailing slash",
			origin:     "http://localhost:7880",
			wantStatus: http.StatusOK,
		},
		{
			name:       "base URL with uppercase",
			origin:     "HTTP://LOCALHOST:7880",
			wantStatus: http.StatusOK,
		},
		{
			name:       "trusted origin with default https port stripped",
			origin:     "https://example.com",
			wantStatus: http.StatusOK,
		},
		{
			name:       "trusted origin with default http port stripped",
			origin:     "http://test.local",
			wantStatus: http.StatusOK,
		},
		{
			name:       "untrusted different port",
			origin:     "http://localhost:7881",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "untrusted null origin",
			origin:     "null",
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/test", nil)
			req.Header.Set("Origin", tc.origin)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			if resp.StatusCode != tc.wantStatus {
				t.Errorf("origin %q got status %d, want %d", tc.origin, resp.StatusCode, tc.wantStatus)
			}
		})
	}
}
