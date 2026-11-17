package handler_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log/slog"
	"maps"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/gofiber/fiber/v3"
	"github.com/kilip/opus/ent"
	"github.com/kilip/opus/ent/enttest"
	"github.com/kilip/opus/internal/platform/httpx"
	"github.com/kilip/opus/internal/tracker/domain"
	"github.com/kilip/opus/internal/tracker/handler"
	"github.com/kilip/opus/internal/tracker/repo"
	"github.com/kilip/opus/internal/tracker/service"
	_ "modernc.org/sqlite"
)

func setupApp(t *testing.T) (*fiber.App, *ent.Client, *repo.Ent) {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+t.Name()+"?mode=memory&cache=shared&_pragma=foreign_keys(ON)&_pragma=busy_timeout(10000)")
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	db.SetMaxOpenConns(1)
	drv := entsql.OpenDB(dialect.SQLite, db)
	client := enttest.NewClient(t, enttest.WithOptions(ent.Driver(drv)))
	t.Cleanup(func() {
		_ = client.Close()
		_ = db.Close()
	})

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	app := fiber.New(fiber.Config{
		ErrorHandler: httpx.ErrorHandler(logger),
	})

	r := repo.New(client)
	svc := service.New(r)
	handler.New(svc).Register(app.Group("/api"))
	return app, client, r
}

func doReq(t *testing.T, app *fiber.App, method, path string, body any) *http.Response {
	t.Helper()
	var bodyReader io.Reader
	if body != nil {
		switch b := body.(type) {
		case string:
			bodyReader = bytes.NewReader([]byte(b))
		default:
			buf, err := json.Marshal(body)
			if err != nil {
				t.Fatalf("marshal body: %v", err)
			}
			bodyReader = bytes.NewReader(buf)
		}
	}
	r := httptest.NewRequest(method, path, bodyReader)
	if body != nil {
		r.Header.Set("Content-Type", "application/json")
	}
	resp, err := app.Test(r)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	return resp
}

func decodeBody[T any](t *testing.T, resp *http.Response) T {
	t.Helper()
	var v T
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	return v
}

func TestHandler_GetProjectStatus(t *testing.T) {
	app, client, r := setupApp(t)
	ctx := context.Background()

	// 1. Invalid UUID -> 400
	respBad := doReq(t, app, http.MethodGet, "/api/projects/not-a-uuid/tracker", nil)
	if respBad.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", respBad.StatusCode)
	}
	errBad := decodeBody[httpx.ErrorResponse](t, respBad)
	if errBad.Error != "INVALID_PROJECT_ID" {
		t.Fatalf("expected INVALID_PROJECT_ID, got %q", errBad.Error)
	}

	// 2. Unknown UUID -> 404
	respMissing := doReq(t, app, http.MethodGet, "/api/projects/01933a3c-1b77-7000-8000-000000000000/tracker", nil)
	if respMissing.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", respMissing.StatusCode)
	}
	errMissing := decodeBody[httpx.ErrorResponse](t, respMissing)
	if errMissing.Error != "PROJECT_NOT_FOUND" {
		t.Fatalf("expected PROJECT_NOT_FOUND, got %q", errMissing.Error)
	}

	// 3. Project with no sync state yet -> 200 {lastSyncAt: null, lastError: ""}
	p, err := client.Project.Create().
		SetSlug("tracker-h-proj").
		SetName("Tracker Handler Proj").
		SetRepoProvider("github").
		SetRepoRef("org/proj").
		SetTrackerProvider("github").
		SetTrackerRef("org/proj").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	respEmpty := doReq(t, app, http.MethodGet, "/api/projects/"+p.ID.String()+"/tracker", nil)
	if respEmpty.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", respEmpty.StatusCode)
	}
	emptyData := decodeBody[httpx.Response[domain.TrackerStatusResponse]](t, respEmpty).Data
	if emptyData.LastSyncAt != nil || emptyData.LastError != "" {
		t.Fatalf("expected zero status, got %+v", emptyData)
	}

	// 4. Project with sync state -> 200 with timestamp and error
	ts := time.Date(2026, 11, 17, 1, 30, 0, 0, time.UTC)
	if _, err := r.SaveSyncSuccess(ctx, p.ID.String(), "2026-11-17T01:30:00Z", ts); err != nil {
		t.Fatalf("SaveSyncSuccess: %v", err)
	}
	if err := r.UpsertLastError(ctx, p.ID.String(), "gh failed"); err != nil {
		t.Fatalf("UpsertLastError: %v", err)
	}

	respPop := doReq(t, app, http.MethodGet, "/api/projects/"+p.ID.String()+"/tracker", nil)
	if respPop.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", respPop.StatusCode)
	}
	popData := decodeBody[httpx.Response[domain.TrackerStatusResponse]](t, respPop).Data
	if popData.LastSyncAt == nil || !popData.LastSyncAt.Equal(ts) {
		t.Fatalf("expected LastSyncAt %v, got %v", ts, popData.LastSyncAt)
	}
	if popData.LastError != "gh failed" {
		t.Fatalf("expected LastError %q, got %q", "gh failed", popData.LastError)
	}
}

func TestHandler_SettingsGetAndPut(t *testing.T) {
	app, client, _ := setupApp(t)
	ctx := context.Background()

	// 1. GET /api/tracker/settings returns defaults
	respGet := doReq(t, app, http.MethodGet, "/api/tracker/settings", nil)
	if respGet.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", respGet.StatusCode)
	}
	getData := decodeBody[httpx.Response[domain.TrackerSettingsResponse]](t, respGet).Data
	if !maps.Equal(getData.StatusLabels, domain.DefaultStatusLabels()) {
		t.Fatalf("expected defaults %v, got %v", domain.DefaultStatusLabels(), getData.StatusLabels)
	}

	// 2. PUT /api/tracker/settings with valid overrides -> 200 with defaults filled
	respPut := doReq(t, app, http.MethodPut, "/api/tracker/settings", `{"statusLabels":{"todo":"queue","blocked":"waiting"}}`)
	if respPut.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", respPut.StatusCode)
	}
	putData := decodeBody[httpx.Response[domain.TrackerSettingsResponse]](t, respPut).Data
	if putData.StatusLabels["todo"] != "queue" || putData.StatusLabels["review"] != "status:review" {
		t.Fatalf("unexpected settings response: %v", putData.StatusLabels)
	}

	// 3. Invalid inputs -> 400 TRACKER_INVALID_LABEL_CONFIG
	invalidPayloads := []string{
		`{}`,
		`{"statusLabels":{"done":"completed"}}`,
		`{"statusLabels":{"todo":" leading"}}`,
		`{"statusLabels":{"todo":"status:backlog"}}`,
	}
	for _, body := range invalidPayloads {
		resp := doReq(t, app, http.MethodPut, "/api/tracker/settings", body)
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400 for %s, got %d", body, resp.StatusCode)
		}
		errResp := decodeBody[httpx.ErrorResponse](t, resp)
		if errResp.Error != "TRACKER_INVALID_LABEL_CONFIG" {
			t.Fatalf("expected TRACKER_INVALID_LABEL_CONFIG for %s, got %q", body, errResp.Error)
		}
	}

	// 4. Collision with existing project override -> 400
	if _, err := client.Project.Create().
		SetSlug("colliding-proj").
		SetName("Colliding Proj").
		SetRepoProvider("github").
		SetRepoRef("org/collide").
		SetTrackerProvider("github").
		SetTrackerRef("org/collide").
		SetTrackerLabels(map[string]string{"review": "qa-queue"}).
		Save(ctx); err != nil {
		t.Fatalf("create project: %v", err)
	}
	respCollide := doReq(t, app, http.MethodPut, "/api/tracker/settings", `{"statusLabels":{"blocked":"qa-queue"}}`)
	if respCollide.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 on project collision, got %d", respCollide.StatusCode)
	}

	// 5. Empty map {} resets to defaults -> 200
	respReset := doReq(t, app, http.MethodPut, "/api/tracker/settings", `{"statusLabels":{}}`)
	if respReset.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on reset, got %d", respReset.StatusCode)
	}
	resetData := decodeBody[httpx.Response[domain.TrackerSettingsResponse]](t, respReset).Data
	if !maps.Equal(resetData.StatusLabels, domain.DefaultStatusLabels()) {
		t.Fatalf("expected defaults after reset, got %v", resetData.StatusLabels)
	}
}
