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
	svc := service.New(r, service.Config{
		BaseURL: "http://localhost:7880",
	})
	h := handler.New(svc)
	h.Register(app.Group("/api"))
	h.RegisterPublic(app.Group("/api"))
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
	app, _ := setupApp(t)
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
