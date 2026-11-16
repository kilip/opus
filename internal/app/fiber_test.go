package app_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/fasthttp/websocket"
	"github.com/gofiber/fiber/v3"
	"github.com/kilip/opus/ent"
	"github.com/kilip/opus/ent/enttest"
	entnode "github.com/kilip/opus/ent/node"
	"github.com/kilip/opus/internal/app"
	"github.com/kilip/opus/internal/node"
	"github.com/kilip/opus/internal/node/domain"
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

// AC2, AC4: Node registry startup master persistence across restarts and auth check (401 without session).
func TestGetFiber_NodesRequiresAuthAndPersistsMaster(t *testing.T) {
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

	// 1. First app startup
	fiberApp1 := app.GetFiber(cfg, logger, client, eventbus.New())

	// Unauthenticated GET /api/nodes returns 401
	reqNodes := httptest.NewRequest("GET", "/api/nodes", nil)
	respNodes, err := fiberApp1.Test(reqNodes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if respNodes.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 for /api/nodes, got %d", respNodes.StatusCode)
	}
	body := parseResponseBody(t, respNodes)
	if body.Error != "UNAUTHORIZED" {
		t.Errorf("expected error %q, got %q", "UNAUTHORIZED", body.Error)
	}

	// Unauthenticated POST /api/nodes/join-tokens returns 401
	reqJoinTokens := httptest.NewRequest("POST", "/api/nodes/join-tokens", nil)
	respJoinTokens, err := fiberApp1.Test(reqJoinTokens)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if respJoinTokens.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 for /api/nodes/join-tokens, got %d", respJoinTokens.StatusCode)
	}
	bodyJoinTokens := parseResponseBody(t, respJoinTokens)
	if bodyJoinTokens.Error != "UNAUTHORIZED" {
		t.Errorf("expected error %q, got %q", "UNAUTHORIZED", bodyJoinTokens.Error)
	}

	// Verify master node created in DB
	ctx := context.Background()
	masters1, err := client.Node.Query().Where(entnode.RoleEQ(entnode.RoleMaster)).All(ctx)
	if err != nil {
		t.Fatalf("query masters: %v", err)
	}
	if len(masters1) != 1 {
		t.Fatalf("expected 1 master node, got %d", len(masters1))
	}
	m1 := masters1[0]

	// 2. Second app startup (simulating restart)
	_ = app.GetFiber(cfg, logger, client, eventbus.New())

	masters2, err := client.Node.Query().Where(entnode.RoleEQ(entnode.RoleMaster)).All(ctx)
	if err != nil {
		t.Fatalf("query masters after restart: %v", err)
	}
	if len(masters2) != 1 {
		t.Fatalf("expected 1 master node after restart, got %d", len(masters2))
	}
	m2 := masters2[0]
	if m2.ID != m1.ID {
		t.Errorf("expected master ID to persist across restarts: got %v, want %v", m2.ID, m1.ID)
	}
}

// POST /api/nodes/join succeeds without session cookie.
func TestGetFiber_NodeJoinWithoutAuth(t *testing.T) {
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

	// Create a join token directly in DB
	ctx := context.Background()
	rawToken := "valid-join-token-for-fiber-test"
	h := sha256.Sum256([]byte(rawToken))
	_, err = client.JoinToken.Create().
		SetTokenHash(hex.EncodeToString(h[:])).
		SetExpiresAt(time.Now().Add(15 * time.Minute)).
		Save(ctx)
	if err != nil {
		t.Fatalf("create join token: %v", err)
	}

	// POST /api/nodes/join without session cookie must succeed (public endpoint)
	reqBody := `{"token":"valid-join-token-for-fiber-test","name":"worker-public"}`
	req := httptest.NewRequest("POST", "/api/nodes/join", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err := fiberApp.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 for /api/nodes/join without session, got %d", resp.StatusCode)
	}

	var res struct {
		Data struct {
			NodeID     string `json:"nodeId"`
			Credential string `json:"credential"`
		} `json:"data"`
	}
	bodyBytes, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(bodyBytes, &res); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if res.Data.NodeID == "" || res.Data.Credential == "" {
		t.Errorf("expected non-empty NodeID and Credential, got %+v", res.Data)
	}

	// Verify worker node created in DB
	worker, err := client.Node.Query().Where(entnode.NameEQ("worker-public")).Only(ctx)
	if err != nil {
		t.Fatalf("query worker: %v", err)
	}
	if worker.Role != entnode.RoleWorker {
		t.Errorf("expected worker role, got %s", worker.Role)
	}
}

// AC5: A test using an in-process Fiber app and a WebSocket client sends hello,
// receives a command sent through Send, and sends an event that reaches a registered handler callback.
func TestGetFiber_NodeControlChannel(t *testing.T) {
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
	nodeFeature := node.New(client, node.Config{BaseURL: cfg.BaseURL})

	// Register event callback on hub
	eventCh := make(chan domain.Envelope, 1)
	nodeFeature.Hub().OnEvent(func(nodeID string, env domain.Envelope) {
		eventCh <- env
	})

	fiberApp := app.GetFiber(cfg, logger, client, eventbus.New(), nodeFeature)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}
	defer func() { _ = ln.Close() }()

	go func() {
		_ = fiberApp.Listener(ln)
	}()

	// Create a worker node in DB
	ctx := context.Background()
	workerCred := "worker-secret-control-token"
	h := sha256.Sum256([]byte(workerCred))
	workerNode, err := client.Node.Create().
		SetName("worker-ws-test").
		SetRole(entnode.RoleWorker).
		SetStatus(entnode.StatusOffline).
		SetCredentialHash(hex.EncodeToString(h[:])).
		Save(ctx)
	if err != nil {
		t.Fatalf("create worker: %v", err)
	}

	wsURL := fmt.Sprintf("ws://%s/api/nodes/connect", ln.Addr().String())
	header := http.Header{
		"Authorization": []string{"Bearer " + workerCred},
	}

	wsConn, resp, err := websocket.DefaultDialer.Dial(wsURL, header)
	if err != nil {
		t.Fatalf("websocket dial failed: %v", err)
	}
	defer func() { _ = wsConn.Close() }()

	if resp.StatusCode != http.StatusSwitchingProtocols {
		t.Fatalf("expected 101 Switching Protocols, got %d", resp.StatusCode)
	}

	// 1. Client sends hello
	helloEnv := domain.Envelope{
		Type: domain.EnvelopeTypeHello,
		ID:   "hello-1",
	}
	if err := wsConn.WriteJSON(helloEnv); err != nil {
		t.Fatalf("send hello failed: %v", err)
	}

	// Wait briefly for connection registration in Hub
	deadline := time.Now().Add(2 * time.Second)
	for !nodeFeature.Hub().Connected(workerNode.ID.String()) {
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for worker to be connected in hub")
		}
		time.Sleep(10 * time.Millisecond)
	}

	// 2. Server sends command through Send; client receives it
	cmdEnv := domain.Envelope{
		Type:    domain.EnvelopeTypeCommand,
		ID:      "cmd-42",
		Payload: []byte(`{"action":"sync"}`),
	}
	if err := nodeFeature.Hub().Send(workerNode.ID.String(), cmdEnv); err != nil {
		t.Fatalf("send command failed: %v", err)
	}

	var receivedCmd domain.Envelope
	if err := wsConn.ReadJSON(&receivedCmd); err != nil {
		t.Fatalf("client read command failed: %v", err)
	}
	if receivedCmd.Type != domain.EnvelopeTypeCommand {
		t.Errorf("expected command type, got %s", receivedCmd.Type)
	}
	if receivedCmd.ID != "cmd-42" {
		t.Errorf("expected cmd-42 id, got %s", receivedCmd.ID)
	}

	// 3. Client sends event that reaches the registered handler callback
	eventEnv := domain.Envelope{
		Type:    domain.EnvelopeTypeEvent,
		ID:      "evt-99",
		Payload: []byte(`{"status":"progress"}`),
	}
	if err := wsConn.WriteJSON(eventEnv); err != nil {
		t.Fatalf("client send event failed: %v", err)
	}

	select {
	case receivedEvent := <-eventCh:
		if receivedEvent.Type != domain.EnvelopeTypeEvent {
			t.Errorf("expected event type, got %s", receivedEvent.Type)
		}
		if receivedEvent.ID != "evt-99" {
			t.Errorf("expected evt-99 id, got %s", receivedEvent.ID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for event callback")
	}
}
