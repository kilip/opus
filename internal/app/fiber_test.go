package app_test

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
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
	"github.com/kilip/opus/internal/project"
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

	// 3. Unauthenticated GET /api/projects returns 401
	reqProjects := httptest.NewRequest("GET", "/api/projects", nil)
	respProjects, err := fiberApp.Test(reqProjects)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if respProjects.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 for /api/projects, got %d", respProjects.StatusCode)
	}
	bodyProjects := parseResponseBody(t, respProjects)
	if bodyProjects.Error != "UNAUTHORIZED" {
		t.Errorf("expected error %q, got %q", "UNAUTHORIZED", bodyProjects.Error)
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

	// Unauthenticated DELETE /api/nodes/:id returns 401
	reqDeleteNode := httptest.NewRequest("DELETE", "/api/nodes/01933a3c-1b77-7000-8000-000000000000", nil)
	respDeleteNode, err := fiberApp1.Test(reqDeleteNode)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if respDeleteNode.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 for /api/nodes/:id, got %d", respDeleteNode.StatusCode)
	}
	bodyDeleteNode := parseResponseBody(t, respDeleteNode)
	if bodyDeleteNode.Error != "UNAUTHORIZED" {
		t.Errorf("expected error %q, got %q", "UNAUTHORIZED", bodyDeleteNode.Error)
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

	fiberApp := app.GetFiber(cfg, logger, client, eventbus.New(), app.WithNode(nodeFeature))

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

func TestGetFiber_ProjectDeleteGuardWithTasks(t *testing.T) {
	ctx := context.Background()
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

	// Create user & session
	u, err := client.User.Create().
		SetEmail("admin@example.com").
		Save(ctx)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	token := "valid-session-token-12345678901234567890"
	mac := hmac.New(sha256.New, []byte(cfg.SessionSecret))
	mac.Write([]byte(token))
	tokenHash := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	_, err = client.AuthSession.Create().
		SetUserID(u.ID).
		SetTokenHash(tokenHash).
		SetExpiresAt(time.Now().Add(time.Hour)).
		Save(ctx)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	authReq := func(method, path string, body any) *http.Request {
		var r io.Reader
		if body != nil {
			b, _ := json.Marshal(body)
			r = bytes.NewReader(b)
		}
		req := httptest.NewRequest(method, path, r)
		req.Header.Set("Cookie", "opus_session="+token)
		req.Header.Set("Origin", "http://localhost:7880")
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		return req
	}

	// 1. Create project 1
	p1Body := project.CreateProjectRequest{
		Slug:            "p1",
		Name:            "Project 1",
		RepoProvider:    "github",
		RepoRef:         "org/p1",
		TrackerProvider: "github",
		TrackerRef:      "org/p1",
	}
	respP1, err := fiberApp.Test(authReq("POST", "/api/projects", p1Body))
	if err != nil || respP1.StatusCode != http.StatusCreated {
		t.Fatalf("create project 1 failed: %v (code %d)", err, respP1.StatusCode)
	}
	var p1Created struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	_ = json.NewDecoder(respP1.Body).Decode(&p1Created)
	p1ID := p1Created.Data.ID

	// 2. Create project 2 (no tasks)
	p2Body := project.CreateProjectRequest{
		Slug:            "p2",
		Name:            "Project 2",
		RepoProvider:    "github",
		RepoRef:         "org/p2",
		TrackerProvider: "github",
		TrackerRef:      "org/p2",
	}
	respP2, err := fiberApp.Test(authReq("POST", "/api/projects", p2Body))
	if err != nil || respP2.StatusCode != http.StatusCreated {
		t.Fatalf("create project 2 failed: %v (code %d)", err, respP2.StatusCode)
	}
	var p2Created struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	_ = json.NewDecoder(respP2.Body).Decode(&p2Created)
	p2ID := p2Created.Data.ID

	// 3. Create a task for project 1
	taskBody := map[string]string{
		"projectId": p1ID,
		"title":     "Task for P1",
	}
	respTask, err := fiberApp.Test(authReq("POST", "/api/tasks", taskBody))
	if err != nil || respTask.StatusCode != http.StatusCreated {
		t.Fatalf("create task failed: %v (code %d)", err, respTask.StatusCode)
	}

	// 4. Attempt DELETE /api/projects/:p1ID -> 409 Conflict (AC5)
	respDelP1, err := fiberApp.Test(authReq("DELETE", "/api/projects/"+p1ID, nil))
	if err != nil {
		t.Fatalf("delete p1 failed: %v", err)
	}
	if respDelP1.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 Conflict for project with tasks, got %d", respDelP1.StatusCode)
	}
	delP1Body := parseResponseBody(t, respDelP1)
	if delP1Body.Error != "PROJECT_HAS_TASKS" {
		t.Fatalf("expected PROJECT_HAS_TASKS error, got %s", delP1Body.Error)
	}

	// 5. DELETE /api/projects/:p2ID (has no tasks) -> 204 No Content (AC5)
	respDelP2, err := fiberApp.Test(authReq("DELETE", "/api/projects/"+p2ID, nil))
	if err != nil {
		t.Fatalf("delete p2 failed: %v", err)
	}
	if respDelP2.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 No Content for project without tasks, got %d", respDelP2.StatusCode)
	}
}

func TestGetFiber_TrackerAuthAndProjectLabelValidation(t *testing.T) {
	ctx := context.Background()
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

	// 1. Tracker endpoints reject unauthenticated requests with 401
	for _, tc := range []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/tracker/settings"},
		{http.MethodPut, "/api/tracker/settings"},
		{http.MethodGet, "/api/projects/01933a3c-1b77-7000-8000-000000000000/tracker"},
	} {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		req.Header.Set("Origin", "http://localhost:7880")
		resp, err := fiberApp.Test(req)
		if err != nil {
			t.Fatalf("%s %s: %v", tc.method, tc.path, err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected 401 for unauthenticated %s %s, got %d", tc.method, tc.path, resp.StatusCode)
		}
	}

	// Create authenticated session
	u, err := client.User.Create().SetEmail("tracker@example.com").Save(ctx)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	token := "valid-session-token-tracker-1234567890"
	mac := hmac.New(sha256.New, []byte(cfg.SessionSecret))
	mac.Write([]byte(token))
	tokenHash := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if _, err := client.AuthSession.Create().
		SetUserID(u.ID).
		SetTokenHash(tokenHash).
		SetExpiresAt(time.Now().Add(time.Hour)).
		Save(ctx); err != nil {
		t.Fatalf("create session: %v", err)
	}

	authReq := func(method, path string, body any) *http.Request {
		var r io.Reader
		if body != nil {
			b, _ := json.Marshal(body)
			r = bytes.NewReader(b)
		}
		req := httptest.NewRequest(method, path, r)
		req.Header.Set("Cookie", "opus_session="+token)
		req.Header.Set("Origin", "http://localhost:7880")
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		return req
	}

	// 2. Create a project and validate PATCH trackerLabels through wired tracker validator
	respCreate, err := fiberApp.Test(authReq(http.MethodPost, "/api/projects", project.CreateProjectRequest{
		Slug:         "wired-tracker",
		Name:         "Wired Tracker",
		RepoProvider: "github",
		RepoRef:      "org/wired-tracker",
	}))
	if err != nil || respCreate.StatusCode != http.StatusCreated {
		t.Fatalf("create project failed: %v (code %d)", err, respCreate.StatusCode)
	}
	var created struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	_ = json.NewDecoder(respCreate.Body).Decode(&created)
	_ = respCreate.Body.Close()

	// Invalid trackerLabels ("done") returns 400 PROJECT_INVALID_TRACKER_LABELS
	respBadPatch, err := fiberApp.Test(authReq(http.MethodPatch, "/api/projects/"+created.Data.ID, map[string]any{
		"trackerLabels": map[string]string{"done": "finished"},
	}))
	if err != nil {
		t.Fatalf("patch project failed: %v", err)
	}
	if respBadPatch.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 on invalid trackerLabels, got %d", respBadPatch.StatusCode)
	}
	badBody := parseResponseBody(t, respBadPatch)
	if badBody.Error != "PROJECT_INVALID_TRACKER_LABELS" {
		t.Fatalf("expected PROJECT_INVALID_TRACKER_LABELS, got %q", badBody.Error)
	}

	// Valid trackerLabels succeeds -> 200
	respGoodPatch, err := fiberApp.Test(authReq(http.MethodPatch, "/api/projects/"+created.Data.ID, map[string]any{
		"trackerLabels": map[string]string{"todo": "queue"},
	}))
	if err != nil || respGoodPatch.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on valid trackerLabels, got %d (err: %v)", respGoodPatch.StatusCode, err)
	}
	_ = respGoodPatch.Body.Close()

	// GET /api/projects/:id/tracker -> 200
	respStatus, err := fiberApp.Test(authReq(http.MethodGet, "/api/projects/"+created.Data.ID+"/tracker", nil))
	if err != nil || respStatus.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on GET /tracker, got %d (err: %v)", respStatus.StatusCode, err)
	}
	_ = respStatus.Body.Close()

	// GET /api/tracker/settings -> 200
	respSettings, err := fiberApp.Test(authReq(http.MethodGet, "/api/tracker/settings", nil))
	if err != nil || respSettings.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on GET /tracker/settings, got %d (err: %v)", respSettings.StatusCode, err)
	}
	_ = respSettings.Body.Close()
}
