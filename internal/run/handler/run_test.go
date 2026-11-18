package handler_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/kilip/opus/ent"
	"github.com/kilip/opus/ent/enttest"
	"github.com/kilip/opus/internal/platform/eventbus"
	"github.com/kilip/opus/internal/platform/httpx"
	"github.com/kilip/opus/internal/run/domain"
	"github.com/kilip/opus/internal/run/handler"
	"github.com/kilip/opus/internal/run/repo"
	"github.com/kilip/opus/internal/run/service"
	_ "modernc.org/sqlite"
)

func setupHandlerApp(t *testing.T) (*fiber.App, *service.Service) {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+t.Name()+"?mode=memory&cache=shared&_pragma=foreign_keys(ON)&_pragma=busy_timeout(10000)")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db.SetMaxOpenConns(1)
	drv := entsql.OpenDB(dialect.SQLite, db)
	client := enttest.NewClient(t, enttest.WithOptions(ent.Driver(drv)))
	t.Cleanup(func() { _ = client.Close() })

	bus := eventbus.New()
	t.Cleanup(func() { _ = bus.Close(context.Background()) })

	svc := service.New(repo.New(client), service.Config{Bus: bus})
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(slog.Default())})
	api := app.Group("/api")
	handler.New(svc, service.StageConfig{}).Register(api)
	return app, svc
}

func TestRunHandler_ListGetAndCancel(t *testing.T) {
	ctx := context.Background()
	app, svc := setupHandlerApp(t)

	r1, err := svc.Create(ctx, domain.CreateInput{
		TaskID:    uuid.New(),
		ProjectID: uuid.New(),
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// GET /api/runs
	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/runs", nil))
	if err != nil {
		t.Fatalf("GET /api/runs: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/runs status = %d, want 200", resp.StatusCode)
	}
	var listBody httpx.Response[[]domain.RunResponse]
	if err := json.NewDecoder(resp.Body).Decode(&listBody); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(listBody.Data) != 1 || listBody.Data[0].ID != r1.ID.String() {
		t.Fatalf("unexpected list response: %+v", listBody)
	}

	// GET /api/runs/:id
	resp, err = app.Test(httptest.NewRequest(http.MethodGet, "/api/runs/"+r1.ID.String(), nil))
	if err != nil {
		t.Fatalf("GET /api/runs/:id: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/runs/:id status = %d, want 200", resp.StatusCode)
	}

	// POST /api/runs/:id/cancel
	req := httptest.NewRequest(http.MethodPost, "/api/runs/"+r1.ID.String()+"/cancel", bytes.NewBufferString(`{"reason":"abort"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("POST /api/runs/:id/cancel: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST cancel status = %d, want 200", resp.StatusCode)
	}
	var cancelBody httpx.Response[domain.RunResponse]
	if err := json.NewDecoder(resp.Body).Decode(&cancelBody); err != nil {
		t.Fatalf("decode cancel: %v", err)
	}
	if cancelBody.Data.State != domain.StateCancelled {
		t.Fatalf("cancelled state = %s, want cancelled", cancelBody.Data.State)
	}

	// GET /api/runs/bad-id -> 400
	resp, err = app.Test(httptest.NewRequest(http.MethodGet, "/api/runs/not-a-uuid", nil))
	if err != nil {
		t.Fatalf("GET bad id: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("GET bad id status = %d, want 400", resp.StatusCode)
	}
}
