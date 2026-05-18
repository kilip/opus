package dash

import (
	"net/http/httptest"
	"testing"
)

func TestNewServer_SPAFallback(t *testing.T) {
	app := NewServer()

	// Test direct index.html
	req := httptest.NewRequest("GET", "/index.html", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Errorf("expected status 200 for index.html, got %d", resp.StatusCode)
	}

	// Test unknown route should return 200 (index.html fallback)
	req = httptest.NewRequest("GET", "/some-route", nil)
	resp, _ = app.Test(req)

	if resp.StatusCode != 200 {
		t.Errorf("expected status 200 for SPA fallback, got %d", resp.StatusCode)
	}
}
