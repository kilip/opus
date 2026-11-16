package handler_test

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
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
	entnode "github.com/kilip/opus/ent/node"
	"github.com/kilip/opus/internal/node/handler"
	"github.com/kilip/opus/internal/node/repo"
	"github.com/kilip/opus/internal/node/service"
	"github.com/kilip/opus/internal/platform/httpx"
	_ "modernc.org/sqlite"
)

func setupConnectApp(t *testing.T) (*fiber.App, *ent.Client, *service.Hub) {
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
	})

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	app := fiber.New(fiber.Config{
		ErrorHandler: httpx.ErrorHandler(logger),
	})

	r := repo.New(client)
	svc := service.New(r, service.Config{
		BaseURL: "http://localhost:7880",
	})
	hub := service.NewHub()
	h := handler.New(svc, hub)
	h.RegisterPublic(app.Group("/api"))
	return app, client, hub
}

func parseErr(t *testing.T, resp *http.Response) string {
	t.Helper()
	var errResp struct {
		Error string `json:"error"`
	}
	b, _ := io.ReadAll(resp.Body)
	_ = json.Unmarshal(b, &errResp)
	return errResp.Error
}

func TestConnect_MissingAuth(t *testing.T) {
	app, _, _ := setupConnectApp(t)

	req := httptest.NewRequest("GET", "/api/nodes/connect", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
	if errCode := parseErr(t, resp); errCode != "UNAUTHORIZED" {
		t.Errorf("expected UNAUTHORIZED, got %q", errCode)
	}
}

func TestConnect_InvalidAuthScheme(t *testing.T) {
	app, _, _ := setupConnectApp(t)

	req := httptest.NewRequest("GET", "/api/nodes/connect", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
	if errCode := parseErr(t, resp); errCode != "UNAUTHORIZED" {
		t.Errorf("expected UNAUTHORIZED, got %q", errCode)
	}
}

func TestConnect_WrongCredential(t *testing.T) {
	app, _, _ := setupConnectApp(t)

	req := httptest.NewRequest("GET", "/api/nodes/connect", nil)
	req.Header.Set("Authorization", "Bearer invalid-worker-credential")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
	if errCode := parseErr(t, resp); errCode != "UNAUTHORIZED" {
		t.Errorf("expected UNAUTHORIZED, got %q", errCode)
	}
}

func TestConnect_MasterNodeForbidden(t *testing.T) {
	app, client, _ := setupConnectApp(t)
	ctx := context.Background()

	// Seed master node with credential
	masterCred := "master-secret-key"
	mh := sha256.Sum256([]byte(masterCred))
	_, err := client.Node.Create().
		SetName("master-connect-node").
		SetRole(entnode.RoleMaster).
		SetStatus(entnode.StatusOnline).
		SetCredentialHash(hex.EncodeToString(mh[:])).
		Save(ctx)
	if err != nil {
		t.Fatalf("seed master: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/nodes/connect", nil)
	req.Header.Set("Authorization", "Bearer "+masterCred)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", resp.StatusCode)
	}
	if errCode := parseErr(t, resp); errCode != "MASTER_CONNECT_FORBIDDEN" {
		t.Errorf("expected MASTER_CONNECT_FORBIDDEN, got %q", errCode)
	}
}

func TestConnect_ValidWorkerAuth_UpgradeRequired(t *testing.T) {
	app, client, _ := setupConnectApp(t)
	ctx := context.Background()

	workerCred := "valid-secret-worker-credential"
	wh := sha256.Sum256([]byte(workerCred))
	_, err := client.Node.Create().
		SetName("worker-valid-connect").
		SetRole(entnode.RoleWorker).
		SetStatus(entnode.StatusOffline).
		SetCredentialHash(hex.EncodeToString(wh[:])).
		Save(ctx)
	if err != nil {
		t.Fatalf("seed worker: %v", err)
	}

	// Plain HTTP request without upgrade headers: auth passes, returns 426 Upgrade Required
	req := httptest.NewRequest("GET", "/api/nodes/connect", nil)
	req.Header.Set("Authorization", "Bearer "+workerCred)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.StatusCode != http.StatusUpgradeRequired {
		t.Fatalf("expected 426 Upgrade Required, got %d", resp.StatusCode)
	}
}
