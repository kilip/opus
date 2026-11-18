package handler_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
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

type mockConn struct {
	mu     sync.Mutex
	closed bool
}

func (m *mockConn) WriteJSON(v any) error {
	return nil
}

func (m *mockConn) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

func (m *mockConn) isClosed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.closed
}

func setupAppWithHub(t *testing.T) (*fiber.App, *ent.Client, *service.Hub) {
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
	hub := service.NewHub()
	svc := service.New(r, service.Config{
		BaseURL: "http://localhost:7880",
		Hub:     hub,
	})
	h := handler.New(svc, hub)
	h.Register(app.Group("/api"))
	h.RegisterPublic(app.Group("/api"))
	return app, client, hub
}

func setupApp(t *testing.T) (*fiber.App, *ent.Client) {
	t.Helper()
	app, client, _ := setupAppWithHub(t)
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

// AC2: POST /api/nodes/join-tokens returns 201 with { data: { url, expiresAt } }
// where url is {APP_BASE_URL}/join/<token> and expiresAt is 15 minutes ahead.
func TestCreateJoinToken_Success(t *testing.T) {
	app, _ := setupApp(t)

	req := httptest.NewRequest("POST", "/api/nodes/join-tokens", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	var envelope struct {
		Data struct {
			URL       string    `json:"url"`
			ExpiresAt time.Time `json:"expiresAt"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("unmarshal body: %v (raw: %s)", err, string(body))
	}

	const expectedPrefix = "http://localhost:7880/join/"
	if !strings.HasPrefix(envelope.Data.URL, expectedPrefix) {
		t.Errorf("expected URL to start with %s, got %s", expectedPrefix, envelope.Data.URL)
	}
	token := strings.TrimPrefix(envelope.Data.URL, expectedPrefix)
	if len(token) == 0 {
		t.Errorf("expected non-empty token in URL")
	}

	// Verify expiration is ~15m in the future (within 10s tolerance)
	diff := time.Until(envelope.Data.ExpiresAt)
	if diff < 14*time.Minute || diff > 16*time.Minute {
		t.Errorf("expected expiration ~15m ahead, got diff %v", diff)
	}
}

func postJSON(t *testing.T, app *fiber.App, path string, body any, ip ...string) *http.Response {
	t.Helper()
	var b io.Reader
	if body != nil {
		switch v := body.(type) {
		case string:
			b = strings.NewReader(v)
		default:
			data, err := json.Marshal(v)
			if err != nil {
				t.Fatalf("marshal body: %v", err)
			}
			b = bytes.NewReader(data)
		}
	}
	req := httptest.NewRequest("POST", path, b)
	req.Header.Set("Content-Type", "application/json")
	if len(ip) > 0 && ip[0] != "" {
		req.Header.Set("X-Forwarded-For", ip[0])
	}
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	return resp
}

// AC1: POST /api/nodes/join with body { token, name } returns 201 with { data: { nodeId, credential } }
// for a valid token; a node with role worker, status offline and credential_hash = sha256(credential) is created.
func TestJoin_Success(t *testing.T) {
	app, client := setupApp(t)
	ctx := context.Background()

	// 1. Create a join token
	reqTok := httptest.NewRequest("POST", "/api/nodes/join-tokens", nil)
	respTok, err := app.Test(reqTok)
	if err != nil {
		t.Fatalf("request join-tokens failed: %v", err)
	}
	bodyTokBytes, _ := io.ReadAll(respTok.Body)
	var tokEnv struct {
		Data struct {
			URL string `json:"url"`
		} `json:"data"`
	}
	_ = json.Unmarshal(bodyTokBytes, &tokEnv)
	rawToken := strings.TrimPrefix(tokEnv.Data.URL, "http://localhost:7880/join/")

	// 2. Join node
	joinBody := domain.JoinNodeRequest{
		Token: rawToken,
		Name:  "worker-prod-1",
	}
	resp := postJSON(t, app, "/api/nodes/join", joinBody)
	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 201, got %d (body: %s)", resp.StatusCode, string(b))
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	var envelope httpx.Response[domain.JoinNodeResponse]
	if err := json.Unmarshal(bodyBytes, &envelope); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if envelope.Data.NodeID == "" {
		t.Errorf("expected non-empty NodeID")
	}
	if envelope.Data.Credential == "" {
		t.Errorf("expected non-empty Credential")
	}

	// 3. Verify node in DB
	n, err := client.Node.Query().Where(entnode.NameEQ("worker-prod-1")).Only(ctx)
	if err != nil {
		t.Fatalf("query node: %v", err)
	}
	if n.Role != entnode.RoleWorker {
		t.Errorf("expected role worker, got %s", n.Role)
	}
	if n.Status != entnode.StatusOffline {
		t.Errorf("expected status offline, got %s", n.Status)
	}
	if n.CredentialHash == nil {
		t.Fatalf("expected credential hash in DB")
	}
	expectedHash := sha256.Sum256([]byte(envelope.Data.Credential))
	if *n.CredentialHash != hex.EncodeToString(expectedHash[:]) {
		t.Errorf("credential hash mismatch: got %s, want %s", *n.CredentialHash, hex.EncodeToString(expectedHash[:]))
	}
}

// AC3: An expired, used or unknown token returns 401 or 403 with an httpx error envelope;
// a name already in use returns 409; an empty or invalid name returns 400.
func TestJoin_Errors(t *testing.T) {
	app, client := setupApp(t)
	ctx := context.Background()

	genToken := func() string {
		req := httptest.NewRequest("POST", "/api/nodes/join-tokens", nil)
		resp, _ := app.Test(req)
		b, _ := io.ReadAll(resp.Body)
		var env struct {
			Data struct {
				URL string `json:"url"`
			} `json:"data"`
		}
		_ = json.Unmarshal(b, &env)
		return strings.TrimPrefix(env.Data.URL, "http://localhost:7880/join/")
	}

	validToken := genToken()

	t.Run("400 on empty name", func(t *testing.T) {
		resp := postJSON(t, app, "/api/nodes/join", map[string]string{
			"token": validToken,
			"name":  "",
		})
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", resp.StatusCode)
		}
		var errResp httpx.ErrorResponse
		_ = json.NewDecoder(resp.Body).Decode(&errResp)
		if errResp.Error != "INVALID_NODE_NAME" {
			t.Errorf("expected INVALID_NODE_NAME, got %s", errResp.Error)
		}
	})

	t.Run("400 on invalid name characters", func(t *testing.T) {
		resp := postJSON(t, app, "/api/nodes/join", map[string]string{
			"token": validToken,
			"name":  "invalid name with spaces",
		})
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", resp.StatusCode)
		}
		var errResp httpx.ErrorResponse
		_ = json.NewDecoder(resp.Body).Decode(&errResp)
		if errResp.Error != "INVALID_NODE_NAME" {
			t.Errorf("expected INVALID_NODE_NAME, got %s", errResp.Error)
		}
	})

	t.Run("400 on malformed json", func(t *testing.T) {
		resp := postJSON(t, app, "/api/nodes/join", "{invalid json")
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", resp.StatusCode)
		}
	})

	t.Run("401 on unknown token", func(t *testing.T) {
		resp := postJSON(t, app, "/api/nodes/join", map[string]string{
			"token": "unknown-nonexistent-token",
			"name":  "worker-unknown",
		})
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", resp.StatusCode)
		}
		var errResp httpx.ErrorResponse
		_ = json.NewDecoder(resp.Body).Decode(&errResp)
		if errResp.Error != "JOIN_TOKEN_NOT_FOUND" {
			t.Errorf("expected JOIN_TOKEN_NOT_FOUND, got %s", errResp.Error)
		}
	})

	t.Run("401 on expired token", func(t *testing.T) {
		rawExpired := "expired-token-val"
		h := sha256.Sum256([]byte(rawExpired))
		_ = client.JoinToken.Create().
			SetTokenHash(hex.EncodeToString(h[:])).
			SetExpiresAt(time.Now().Add(-1 * time.Hour)).
			Exec(ctx)

		resp := postJSON(t, app, "/api/nodes/join", map[string]string{
			"token": rawExpired,
			"name":  "worker-expired",
		})
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", resp.StatusCode)
		}
		var errResp httpx.ErrorResponse
		_ = json.NewDecoder(resp.Body).Decode(&errResp)
		if errResp.Error != "JOIN_TOKEN_EXPIRED" {
			t.Errorf("expected JOIN_TOKEN_EXPIRED, got %s", errResp.Error)
		}
	})

	t.Run("401 on already used token", func(t *testing.T) {
		tok := genToken()
		resp1 := postJSON(t, app, "/api/nodes/join", map[string]string{
			"token": tok,
			"name":  "worker-first-use",
		})
		if resp1.StatusCode != http.StatusCreated {
			t.Fatalf("first join failed: %d", resp1.StatusCode)
		}

		resp2 := postJSON(t, app, "/api/nodes/join", map[string]string{
			"token": tok,
			"name":  "worker-second-use",
		})
		if resp2.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", resp2.StatusCode)
		}
		var errResp httpx.ErrorResponse
		_ = json.NewDecoder(resp2.Body).Decode(&errResp)
		if errResp.Error != "JOIN_TOKEN_ALREADY_USED" {
			t.Errorf("expected JOIN_TOKEN_ALREADY_USED, got %s", errResp.Error)
		}
	})

	t.Run("409 on name already in use", func(t *testing.T) {
		_, _ = client.Node.Create().
			SetName("duplicate-name").
			SetRole(entnode.RoleWorker).
			SetStatus(entnode.StatusOffline).
			Save(ctx)

		tok := genToken()
		resp := postJSON(t, app, "/api/nodes/join", map[string]string{
			"token": tok,
			"name":  "duplicate-name",
		})
		if resp.StatusCode != http.StatusConflict {
			t.Errorf("expected 409, got %d", resp.StatusCode)
		}
		var errResp httpx.ErrorResponse
		_ = json.NewDecoder(resp.Body).Decode(&errResp)
		if errResp.Error != "NODE_NAME_CONFLICT" {
			t.Errorf("expected NODE_NAME_CONFLICT, got %s", errResp.Error)
		}
	})
}

// AC4: The 11th request within a minute from one IP returns 429; covered by a test.
func TestJoin_RateLimit(t *testing.T) {
	_, client, hub := setupAppWithHub(t)
	app := fiber.New(fiber.Config{
		TrustProxy: true,
		TrustProxyConfig: fiber.TrustProxyConfig{
			Proxies: []string{"0.0.0.0/0"},
		},
		ProxyHeader:  fiber.HeaderXForwardedFor,
		ErrorHandler: httpx.ErrorHandler(slog.New(slog.NewTextHandler(io.Discard, nil))),
	})
	svc := service.New(repo.New(client), service.Config{BaseURL: "http://localhost:7880", Hub: hub})
	handler.New(svc, hub).RegisterPublic(app.Group("/api"))
	ip := "192.168.1.100"

	// 10 requests should not be rate limited
	for i := 1; i <= 10; i++ {
		resp := postJSON(t, app, "/api/nodes/join", map[string]string{
			"token": "tok",
			"name":  "w",
		}, ip)
		if resp.StatusCode == http.StatusTooManyRequests {
			t.Fatalf("request %d was unexpectedly rate limited (429)", i)
		}
	}

	// 11th request from same IP must return 429
	resp11 := postJSON(t, app, "/api/nodes/join", map[string]string{
		"token": "tok",
		"name":  "w",
	}, ip)
	if resp11.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("expected 11th request to return 429, got %d", resp11.StatusCode)
	}

	var errResp httpx.ErrorResponse
	if err := json.NewDecoder(resp11.Body).Decode(&errResp); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if errResp.Error != "RATE_LIMITED" {
		t.Errorf("expected error RATE_LIMITED, got %s", errResp.Error)
	}

	// Different IP should not be rate limited
	otherIP := "192.168.1.101"
	respOther := postJSON(t, app, "/api/nodes/join", map[string]string{
		"token": "tok",
		"name":  "w",
	}, otherIP)
	if respOther.StatusCode == http.StatusTooManyRequests {
		t.Fatalf("request from different IP was unexpectedly rate limited")
	}
}

// AC1: DELETE /api/nodes/{id} returns 204 for existing worker; row is gone afterwards.
func TestDeleteNode_AC1_ExistingWorker(t *testing.T) {
	app, client := setupApp(t)
	ctx := context.Background()

	worker, err := client.Node.Create().
		SetName("worker-to-delete").
		SetRole(entnode.RoleWorker).
		SetStatus(entnode.StatusOffline).
		Save(ctx)
	if err != nil {
		t.Fatalf("create worker: %v", err)
	}

	req := httptest.NewRequest("DELETE", "/api/nodes/"+worker.ID.String(), nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}

	// Verify row is gone from DB
	_, err = client.Node.Get(ctx, worker.ID)
	if !ent.IsNotFound(err) {
		t.Fatalf("expected node to be deleted from DB, got err: %v", err)
	}

	// Second delete on deleted node returns 404
	req2 := httptest.NewRequest("DELETE", "/api/nodes/"+worker.ID.String(), nil)
	resp2, err := app.Test(req2)
	if err != nil {
		t.Fatalf("request 2 failed: %v", err)
	}
	if resp2.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp2.StatusCode)
	}
}

// AC2: Deleting master returns 409 with httpx error; unknown or malformed ID returns 404.
func TestDeleteNode_AC2_MasterForbiddenAndNotFound(t *testing.T) {
	app, client := setupApp(t)
	ctx := context.Background()

	master, err := client.Node.Create().
		SetName("master-primary").
		SetRole(entnode.RoleMaster).
		SetStatus(entnode.StatusOnline).
		Save(ctx)
	if err != nil {
		t.Fatalf("create master: %v", err)
	}

	// Deleting master returns 409 Conflict
	reqMaster := httptest.NewRequest("DELETE", "/api/nodes/"+master.ID.String(), nil)
	respMaster, err := app.Test(reqMaster)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if respMaster.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 Conflict, got %d", respMaster.StatusCode)
	}
	var errResp httpx.ErrorResponse
	if err := json.NewDecoder(respMaster.Body).Decode(&errResp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if errResp.Error != "MASTER_DELETE_FORBIDDEN" {
		t.Errorf("expected error MASTER_DELETE_FORBIDDEN, got %s", errResp.Error)
	}

	// Verify master node still exists in DB
	masterAfter, err := client.Node.Get(ctx, master.ID)
	if err != nil || masterAfter == nil {
		t.Fatalf("expected master to remain in DB, got err: %v", err)
	}

	// Unknown valid UUID returns 404
	reqUnknown := httptest.NewRequest("DELETE", "/api/nodes/01933a3c-1b77-7000-8000-000000000000", nil)
	respUnknown, err := app.Test(reqUnknown)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if respUnknown.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", respUnknown.StatusCode)
	}
	var errRespUnknown httpx.ErrorResponse
	if err := json.NewDecoder(respUnknown.Body).Decode(&errRespUnknown); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if errRespUnknown.Error != "NODE_NOT_FOUND" {
		t.Errorf("expected error NODE_NOT_FOUND, got %s", errRespUnknown.Error)
	}

	// Malformed ID returns 404
	reqMalformed := httptest.NewRequest("DELETE", "/api/nodes/not-a-valid-uuid", nil)
	respMalformed, err := app.Test(reqMalformed)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if respMalformed.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", respMalformed.StatusCode)
	}
	var errRespMalformed httpx.ErrorResponse
	if err := json.NewDecoder(respMalformed.Body).Decode(&errRespMalformed); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if errRespMalformed.Error != "NODE_NOT_FOUND" {
		t.Errorf("expected error NODE_NOT_FOUND, got %s", errRespMalformed.Error)
	}
}

// AC3: Live control channel connection is closed before call returns,
// and the same credential can no longer connect (401).
func TestDeleteNode_AC3_LiveControlChannelClosedAndCredentialRevoked(t *testing.T) {
	app, client, hub := setupAppWithHub(t)
	ctx := context.Background()

	rawSecret := "secret-node-credential-value"
	h := sha256.Sum256([]byte(rawSecret))
	credHash := hex.EncodeToString(h[:])

	worker, err := client.Node.Create().
		SetName("worker-live-chan").
		SetRole(entnode.RoleWorker).
		SetStatus(entnode.StatusOnline).
		SetCredentialHash(credHash).
		Save(ctx)
	if err != nil {
		t.Fatalf("create worker: %v", err)
	}

	conn := &mockConn{}
	_, err = hub.Register(worker.ID.String(), conn)
	if err != nil {
		t.Fatalf("register conn in hub: %v", err)
	}

	if !hub.Connected(worker.ID.String()) {
		t.Fatal("expected hub.Connected to be true")
	}
	if conn.isClosed() {
		t.Fatal("expected conn to not be closed yet")
	}

	// DELETE node
	req := httptest.NewRequest("DELETE", "/api/nodes/"+worker.ID.String(), nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("delete request failed: %v", err)
	}
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}

	// Verify connection was closed before the call returned
	if !conn.isClosed() {
		t.Error("expected live connection to be closed by delete")
	}

	// Verify credential can no longer connect (401 Unauthorized)
	reqConnect := httptest.NewRequest("GET", "/api/nodes/connect", nil)
	reqConnect.Header.Set("Authorization", "Bearer "+rawSecret)
	respConnect, err := app.Test(reqConnect)
	if err != nil {
		t.Fatalf("connect request failed: %v", err)
	}
	if respConnect.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", respConnect.StatusCode)
	}
	var errResp httpx.ErrorResponse
	if err := json.NewDecoder(respConnect.Body).Decode(&errResp); err != nil {
		t.Fatalf("decode connect error: %v", err)
	}
	if errResp.Error != "UNAUTHORIZED" {
		t.Errorf("expected error UNAUTHORIZED, got %s", errResp.Error)
	}
}

func TestPatchNode(t *testing.T) {
	app, client := setupApp(t)
	ctx := context.Background()

	master, err := client.Node.Create().
		SetName("master-host").
		SetHostname("master-host").
		SetRole(entnode.RoleMaster).
		SetStatus(entnode.StatusOnline).
		Save(ctx)
	if err != nil {
		t.Fatalf("create master: %v", err)
	}
	worker, err := client.Node.Create().
		SetName("worker-1").
		SetRole(entnode.RoleWorker).
		SetStatus(entnode.StatusOffline).
		Save(ctx)
	if err != nil {
		t.Fatalf("create worker: %v", err)
	}

	patch := func(id, body string) *http.Response {
		t.Helper()
		req := httptest.NewRequest(http.MethodPatch, "/api/nodes/"+id, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("PATCH %s failed: %v", id, err)
		}
		return resp
	}

	// 1. 200 OK updating master name, IP, MAC (normalized)
	resp := patch(master.ID.String(), `{"name":"control-plane","ipAddress":"192.168.1.10","macAddress":"AA-BB-CC-DD-EE-01"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on master patch, got %d", resp.StatusCode)
	}
	var okResp httpx.Response[domain.NodeResponse]
	if err := json.NewDecoder(resp.Body).Decode(&okResp); err != nil {
		t.Fatalf("decode 200 response: %v", err)
	}
	_ = resp.Body.Close()
	if okResp.Data.Name != "control-plane" {
		t.Errorf("expected name control-plane, got %q", okResp.Data.Name)
	}
	if okResp.Data.Hostname == nil || *okResp.Data.Hostname != "master-host" {
		t.Errorf("expected hostname master-host, got %v", okResp.Data.Hostname)
	}
	if okResp.Data.IPAddress == nil || *okResp.Data.IPAddress != "192.168.1.10" {
		t.Errorf("expected ipAddress 192.168.1.10, got %v", okResp.Data.IPAddress)
	}
	if okResp.Data.MACAddress == nil || *okResp.Data.MACAddress != "aa:bb:cc:dd:ee:01" {
		t.Errorf("expected normalized macAddress aa:bb:cc:dd:ee:01, got %v", okResp.Data.MACAddress)
	}

	// 2. 200 OK clearing IP and MAC with ""
	respClear := patch(master.ID.String(), `{"ipAddress":"","macAddress":""}`)
	if respClear.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on clear, got %d", respClear.StatusCode)
	}
	var clearResp httpx.Response[domain.NodeResponse]
	if err := json.NewDecoder(respClear.Body).Decode(&clearResp); err != nil {
		t.Fatalf("decode clear response: %v", err)
	}
	_ = respClear.Body.Close()
	if clearResp.Data.IPAddress != nil || clearResp.Data.MACAddress != nil {
		t.Errorf("expected nil ipAddress and macAddress after clear, got ip=%v mac=%v", clearResp.Data.IPAddress, clearResp.Data.MACAddress)
	}

	// 3. 400 validation errors
	for _, tc := range []struct {
		body string
		want string
	}{
		{body: `{"name":"invalid name"}`, want: "INVALID_NODE_NAME"},
		{body: `{"ipAddress":"not-an-ip"}`, want: "INVALID_IP_ADDRESS"},
		{body: `{"macAddress":"not-a-mac"}`, want: "INVALID_MAC_ADDRESS"},
	} {
		r := patch(worker.ID.String(), tc.body)
		if r.StatusCode != http.StatusBadRequest {
			t.Fatalf("body %s: expected 400, got %d", tc.body, r.StatusCode)
		}
		var er httpx.ErrorResponse
		if err := json.NewDecoder(r.Body).Decode(&er); err != nil {
			t.Fatalf("decode 400: %v", err)
		}
		_ = r.Body.Close()
		if er.Error != tc.want {
			t.Errorf("body %s: expected error %s, got %s", tc.body, tc.want, er.Error)
		}
	}

	// 4. 404 Not Found (unknown UUID & malformed UUID)
	for _, badID := range []string{"00000000-0000-0000-0000-000000000000", "bad-uuid"} {
		r := patch(badID, `{"name":"worker-x"}`)
		if r.StatusCode != http.StatusNotFound {
			t.Fatalf("id %s: expected 404, got %d", badID, r.StatusCode)
		}
		var er httpx.ErrorResponse
		if err := json.NewDecoder(r.Body).Decode(&er); err != nil {
			t.Fatalf("decode 404: %v", err)
		}
		_ = r.Body.Close()
		if er.Error != "NODE_NOT_FOUND" {
			t.Errorf("id %s: expected NODE_NOT_FOUND, got %s", badID, er.Error)
		}
	}

	// 5. 409 Conflict on duplicate name
	rConflict := patch(worker.ID.String(), `{"name":"control-plane"}`)
	if rConflict.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 on duplicate name, got %d", rConflict.StatusCode)
	}
	var conflictErr httpx.ErrorResponse
	if err := json.NewDecoder(rConflict.Body).Decode(&conflictErr); err != nil {
		t.Fatalf("decode 409: %v", err)
	}
	_ = rConflict.Body.Close()
	if conflictErr.Error != "NODE_NAME_CONFLICT" {
		t.Errorf("expected NODE_NAME_CONFLICT, got %s", conflictErr.Error)
	}
}

func TestWakeNode(t *testing.T) {
	db, err := sql.Open("sqlite", "file:"+t.Name()+"?mode=memory&cache=shared&_pragma=foreign_keys(ON)&_pragma=busy_timeout(10000)")
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	db.SetMaxOpenConns(1)
	drv := entsql.OpenDB(dialect.SQLite, db)
	client := enttest.NewClient(t, enttest.WithOptions(ent.Driver(drv)))
	t.Cleanup(func() { _ = client.Close() })

	var sentAddr string
	var sentPkt []byte
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(logger)})
	r := repo.New(client)
	hub := service.NewHub()
	svc := service.New(r, service.Config{
		Hub: hub,
		WakeSender: func(_ context.Context, addr string, pkt []byte) error {
			sentAddr = addr
			sentPkt = append([]byte(nil), pkt...)
			return nil
		},
	})
	handler.New(svc, hub).Register(app.Group("/api"))

	ctx := context.Background()
	withMAC, err := client.Node.Create().
		SetName("worker-wol").
		SetRole(entnode.RoleWorker).
		SetStatus(entnode.StatusOffline).
		SetMACAddress("aa:bb:cc:dd:ee:ff").
		Save(ctx)
	if err != nil {
		t.Fatalf("create worker with mac: %v", err)
	}
	noMAC, err := client.Node.Create().
		SetName("worker-nomac").
		SetRole(entnode.RoleWorker).
		SetStatus(entnode.StatusOffline).
		Save(ctx)
	if err != nil {
		t.Fatalf("create worker without mac: %v", err)
	}

	// 1. 204 No Content and packet dispatched
	req := httptest.NewRequest(http.MethodPost, "/api/nodes/"+withMAC.ID.String()+"/wake", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("POST wake failed: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}
	if sentAddr != "255.255.255.255:9" || len(sentPkt) != 102 {
		t.Errorf("unexpected packet dispatch: addr=%q len=%d", sentAddr, len(sentPkt))
	}

	// 2. 400 NODE_MAC_NOT_CONFIGURED
	reqNoMAC := httptest.NewRequest(http.MethodPost, "/api/nodes/"+noMAC.ID.String()+"/wake", nil)
	respNoMAC, err := app.Test(reqNoMAC)
	if err != nil {
		t.Fatalf("POST wake no mac failed: %v", err)
	}
	if respNoMAC.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", respNoMAC.StatusCode)
	}
	var noMACErr httpx.ErrorResponse
	if err := json.NewDecoder(respNoMAC.Body).Decode(&noMACErr); err != nil {
		t.Fatalf("decode 400: %v", err)
	}
	_ = respNoMAC.Body.Close()
	if noMACErr.Error != "NODE_MAC_NOT_CONFIGURED" {
		t.Errorf("expected NODE_MAC_NOT_CONFIGURED, got %s", noMACErr.Error)
	}

	// 3. 404 NODE_NOT_FOUND
	reqMissing := httptest.NewRequest(http.MethodPost, "/api/nodes/00000000-0000-0000-0000-000000000000/wake", nil)
	respMissing, err := app.Test(reqMissing)
	if err != nil {
		t.Fatalf("POST wake missing failed: %v", err)
	}
	if respMissing.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", respMissing.StatusCode)
	}
	var missingErr httpx.ErrorResponse
	if err := json.NewDecoder(respMissing.Body).Decode(&missingErr); err != nil {
		t.Fatalf("decode 404: %v", err)
	}
	_ = respMissing.Body.Close()
	if missingErr.Error != "NODE_NOT_FOUND" {
		t.Errorf("expected NODE_NOT_FOUND, got %s", missingErr.Error)
	}
}
