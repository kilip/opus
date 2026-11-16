package handler_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/gofiber/fiber/v3"
	"github.com/kilip/opus/ent"
	"github.com/kilip/opus/ent/enttest"
	entnode "github.com/kilip/opus/ent/node"
	"github.com/kilip/opus/internal/node/domain"
	"github.com/kilip/opus/internal/node/handler"
	"github.com/kilip/opus/internal/node/repo"
	"github.com/kilip/opus/internal/node/service"
	"github.com/kilip/opus/internal/platform/httpx"
	_ "modernc.org/sqlite"
)

func setupApp(t *testing.T) (*fiber.App, *ent.Client) {
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
	svc := service.New(r)
	h := handler.New(svc)
	h.Register(app.Group("/api"))
	return app, client
}

// AC4: GET /api/nodes returns { data: [...], meta: { total } } ordered by id;
// each item has id, name, role, status, lastSeenAt. credential_hash is never exposed.
func TestListNodes_ResponseShapeAndOrder(t *testing.T) {
	app, client := setupApp(t)
	ctx := context.Background()

	now := time.Now().Truncate(time.Second)
	credHash := "argon2id$hashedsecret"

	// Create nodes directly in DB with credential_hash set for worker
	n1, err := client.Node.Create().
		SetName("worker-b").
		SetRole(entnode.RoleWorker).
		SetStatus(entnode.StatusOffline).
		SetLastSeenAt(now).
		SetCredentialHash(credHash).
		Save(ctx)
	if err != nil {
		t.Fatalf("create node 1: %v", err)
	}

	n2, err := client.Node.Create().
		SetName("master-a").
		SetRole(entnode.RoleMaster).
		SetStatus(entnode.StatusOnline).
		Save(ctx)
	if err != nil {
		t.Fatalf("create node 2: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/nodes", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	rawJSON := string(bodyBytes)

	// Verify credential_hash is never exposed in response
	if strings.Contains(rawJSON, "credential_hash") || strings.Contains(rawJSON, "credentialHash") || strings.Contains(rawJSON, credHash) {
		t.Fatalf("credential_hash exposed in response: %s", rawJSON)
	}

	var envelope httpx.Response[[]domain.NodeResponse]
	if err := json.Unmarshal(bodyBytes, &envelope); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if envelope.Meta == nil || envelope.Meta.Total == nil {
		t.Fatalf("expected meta.total, got nil")
	}
	if *envelope.Meta.Total != 2 {
		t.Fatalf("expected meta.total=2, got %d", *envelope.Meta.Total)
	}
	if len(envelope.Data) != 2 {
		t.Fatalf("expected 2 items in data, got %d", len(envelope.Data))
	}

	// Verify ordered by id ascending
	if envelope.Data[0].ID > envelope.Data[1].ID {
		t.Errorf("expected items ordered by id ascending: %s > %s", envelope.Data[0].ID, envelope.Data[1].ID)
	}

	// Verify IDs match created nodes
	ids := map[string]bool{n1.ID.String(): true, n2.ID.String(): true}
	for _, item := range envelope.Data {
		if !ids[item.ID] {
			t.Errorf("unexpected node ID: %s", item.ID)
		}
		if item.Name == "" {
			t.Errorf("expected non-empty name")
		}
		if item.Role == "" {
			t.Errorf("expected non-empty role")
		}
		if item.Status == "" {
			t.Errorf("expected non-empty status")
		}
	}
}
