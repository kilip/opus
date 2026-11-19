package handler_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/gofiber/fiber/v3"
	"github.com/kilip/opus/ent"
	"github.com/kilip/opus/ent/enttest"
	"github.com/kilip/opus/internal/platform/httpx"
	"github.com/kilip/opus/internal/settings/domain"
	"github.com/kilip/opus/internal/settings/handler"
	"github.com/kilip/opus/internal/settings/repo"
	"github.com/kilip/opus/internal/settings/service"
	_ "modernc.org/sqlite"
)

const testAuthSecret = "0123456789abcdef0123456789abcdef"

func setupSettingsTest(t *testing.T, ghHandler http.HandlerFunc, getenv func(string) string) (*fiber.App, *ent.Client, *repo.Ent, *service.Settings) {
	t.Helper()

	db, err := sql.Open("sqlite", "file:"+t.Name()+"?mode=memory&cache=shared&_pragma=foreign_keys(ON)&_pragma=busy_timeout(10000)")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db.SetMaxOpenConns(1)
	drv := entsql.OpenDB(dialect.SQLite, db)
	client := enttest.NewClient(t, enttest.WithOptions(ent.Driver(drv)))
	t.Cleanup(func() {
		_ = client.Close()
		_ = db.Close()
	})

	ghSrv := httptest.NewServer(ghHandler)
	t.Cleanup(ghSrv.Close)

	r := repo.New(client)
	svc, err := service.New(r, service.Config{
		Secret:           testAuthSecret,
		GitHubAPIBaseURL: ghSrv.URL,
		HTTPClient:       ghSrv.Client(),
		Getenv:           getenv,
	})
	if err != nil {
		t.Fatalf("settings service: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	app := fiber.New(fiber.Config{
		ErrorHandler: httpx.ErrorHandler(logger),
	})
	handler.New(svc).Register(app.Group("/api"))
	return app, client, r, svc
}

func doSettingsReq(t *testing.T, app *fiber.App, method, path, body string) *http.Response {
	t.Helper()
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	return resp
}

func TestSettings_EncryptedAtRest_GetPutDelete(t *testing.T) {
	const rawToken = "ghp_super_secret_token_value_1234"
	app, _, r, svc := setupSettingsTest(t, func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/user" && req.Header.Get("Authorization") == "Bearer "+rawToken {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"login":"octocat"}`))
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
	}, func(string) string { return "" })

	// Initially unconfigured
	resp := doSettingsReq(t, app, http.MethodGet, "/api/settings/github-token", "")
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET initial status = %d, want 200", resp.StatusCode)
	}
	var initialEnv struct {
		Data domain.GitHubTokenStatusResponse `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&initialEnv); err != nil {
		t.Fatal(err)
	}
	if initialEnv.Data.Configured || initialEnv.Data.EnvOverride || initialEnv.Data.NeedsReentry {
		t.Fatalf("unexpected initial status: %+v", initialEnv.Data)
	}

	// AC2: PUT stores encrypted ciphertext; raw token never appears in DB or HTTP response.
	putResp := doSettingsReq(t, app, http.MethodPut, "/api/settings/github-token", `{"token":"`+rawToken+`"}`)
	defer func() { _ = putResp.Body.Close() }()
	if putResp.StatusCode != http.StatusOK {
		t.Fatalf("PUT status = %d, want 200", putResp.StatusCode)
	}
	putBody, err := io.ReadAll(putResp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(putBody), rawToken) {
		t.Fatalf("PUT response leaked raw token: %s", putBody)
	}
	var putEnv struct {
		Data domain.GitHubTokenStatusResponse `json:"data"`
	}
	if err := json.Unmarshal(putBody, &putEnv); err != nil {
		t.Fatal(err)
	}
	if !putEnv.Data.Configured || putEnv.Data.Last4 != "1234" || putEnv.Data.UpdatedAt == nil {
		t.Fatalf("unexpected PUT status: %+v", putEnv.Data)
	}

	// Verify DB ciphertext does not contain raw token.
	rec, err := r.GetByKey(context.Background(), "github_token")
	if err != nil || rec == nil {
		t.Fatalf("GetByKey failed: rec=%v err=%v", rec, err)
	}
	if bytes.Contains(rec.Ciphertext, []byte(rawToken)) {
		t.Fatal("raw token found in DB ciphertext column")
	}
	if rec.KeyVersion != 1 {
		t.Fatalf("expected key_version=1, got %d", rec.KeyVersion)
	}
	if got := svc.EffectiveGitHubToken(context.Background()); got != rawToken {
		t.Fatalf("EffectiveGitHubToken = %q, want %q", got, rawToken)
	}

	// AC3: DELETE removes stored token.
	delResp := doSettingsReq(t, app, http.MethodDelete, "/api/settings/github-token", "")
	defer func() { _ = delResp.Body.Close() }()
	if delResp.StatusCode != http.StatusNoContent {
		t.Fatalf("DELETE status = %d, want 204", delResp.StatusCode)
	}
	if got := svc.EffectiveGitHubToken(context.Background()); got != "" {
		t.Fatalf("expected empty token after DELETE, got %q", got)
	}
}

func TestSettings_InvalidTokenRejectedBeforeStore(t *testing.T) {
	app, _, r, _ := setupSettingsTest(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}, func(string) string { return "" })

	// AC4: Invalid token -> 400, nothing stored.
	resp := doSettingsReq(t, app, http.MethodPut, "/api/settings/github-token", `{"token":"ghp_invalid"}`)
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("PUT invalid token status = %d, want 400", resp.StatusCode)
	}
	rec, err := r.GetByKey(context.Background(), "github_token")
	if err != nil {
		t.Fatal(err)
	}
	if rec != nil {
		t.Fatalf("expected no record stored for invalid token, got %+v", rec)
	}

	// Empty token -> 400
	emptyResp := doSettingsReq(t, app, http.MethodPut, "/api/settings/github-token", `{"token":"   "}`)
	defer func() { _ = emptyResp.Body.Close() }()
	if emptyResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("PUT empty token status = %d, want 400", emptyResp.StatusCode)
	}
}

func TestSettings_EnvOverrideAndSecretRotation(t *testing.T) {
	const dbToken = "ghp_db_token_value_9999"
	const envToken = "ghp_env_override_value_8888"

	envMap := map[string]string{}
	app, _, r, svc := setupSettingsTest(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}, func(k string) string { return envMap[k] })

	putResp := doSettingsReq(t, app, http.MethodPut, "/api/settings/github-token", `{"token":"`+dbToken+`"}`)
	_ = putResp.Body.Close()
	if putResp.StatusCode != http.StatusOK {
		t.Fatalf("PUT status = %d, want 200", putResp.StatusCode)
	}

	// AC5: GH_TOKEN env var takes precedence and sets envOverride=true.
	envMap["GH_TOKEN"] = envToken
	if got := svc.EffectiveGitHubToken(context.Background()); got != envToken {
		t.Fatalf("EffectiveGitHubToken with GH_TOKEN = %q, want %q", got, envToken)
	}
	getResp := doSettingsReq(t, app, http.MethodGet, "/api/settings/github-token", "")
	defer func() { _ = getResp.Body.Close() }()
	var statusEnv struct {
		Data domain.GitHubTokenStatusResponse `json:"data"`
	}
	if err := json.NewDecoder(getResp.Body).Decode(&statusEnv); err != nil {
		t.Fatal(err)
	}
	if !statusEnv.Data.EnvOverride {
		t.Fatal("expected envOverride=true when GH_TOKEN is set")
	}

	// Clear env override
	delete(envMap, "GH_TOKEN")

	// AC6: Rotating AUTH_SECRET leaves ciphertext unreadable -> needsReentry=true, no crash.
	rotatedSvc, err := service.New(r, service.Config{
		Secret: "fedcba9876543210fedcba9876543210",
		Getenv: func(string) string { return "" },
	})
	if err != nil {
		t.Fatal(err)
	}
	status, err := rotatedSvc.GetGitHubTokenStatus(context.Background())
	if err != nil {
		t.Fatalf("GetGitHubTokenStatus after rotation failed: %v", err)
	}
	if status.Configured || !status.NeedsReentry || status.Last4 != "9999" {
		t.Fatalf("expected configured=false, needsReentry=true, last4=9999, got %+v", status)
	}
	if got := rotatedSvc.EffectiveGitHubToken(context.Background()); got != "" {
		t.Fatalf("expected empty EffectiveGitHubToken after rotation, got %q", got)
	}
}
