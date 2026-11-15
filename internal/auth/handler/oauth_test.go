package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/kilip/opus/internal/auth/domain"
	"github.com/kilip/opus/internal/auth/handler"
	"github.com/kilip/opus/internal/auth/provider"
	"github.com/kilip/opus/internal/auth/repo"
	"github.com/kilip/opus/internal/auth/service"
	"github.com/kilip/opus/internal/platform/httpx"
)

type fakeProvider struct {
	name        string
	label       string
	authCodeURL string
	profile     provider.Profile
	exchangeErr error
}

func (p *fakeProvider) Name() string {
	return p.name
}

func (p *fakeProvider) Label() string {
	if p.label != "" {
		return p.label
	}
	return p.name
}

func (p *fakeProvider) AuthCodeURL(state, verifier string) string {
	if p.authCodeURL != "" {
		return p.authCodeURL + "?state=" + state
	}
	return "https://example.com/oauth/authorize?state=" + state
}

func (p *fakeProvider) Exchange(_ context.Context, code, verifier string) (provider.Profile, error) {
	if p.exchangeErr != nil {
		return provider.Profile{}, p.exchangeErr
	}
	return p.profile, nil
}

func setupOAuthApp(r repo.Repo, reg *provider.Registry) (*fiber.App, *service.Auth) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(logger)})
	cc := handler.CookieConfig{Secure: false, MaxAge: 86400}
	svc := service.New(r, nil, nil, service.Config{
		Secret:   testSecret,
		Registry: reg,
	})
	h := handler.New(svc, cc)
	authGroup := app.Group("/api/auth")
	h.RegisterPublic(authGroup)
	h.Register(authGroup)
	return app, svc
}

func findCookie(resp *http.Response, name string) *http.Cookie {
	for _, c := range resp.Cookies() {
		if c.Name == name {
			return c
		}
	}
	return nil
}

// AC1: GET /api/auth/providers returns { data: [{ name, label }] } for enabled providers only.
func TestOAuth_ListProviders(t *testing.T) {
	r := newFakeRepo()
	reg := provider.NewRegistry(provider.Config{})
	reg.Register(&fakeProvider{name: "github", label: "GitHub"})
	reg.Register(&fakeProvider{name: "google", label: "Google"})

	app, _ := setupOAuthApp(r, reg)
	req := httptest.NewRequest(http.MethodGet, "/api/auth/providers", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("GET /api/auth/providers: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: want 200, got %d", resp.StatusCode)
	}

	var body struct {
		Data []domain.ProviderItem `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if len(body.Data) != 2 {
		t.Fatalf("want 2 providers, got %d", len(body.Data))
	}
	if body.Data[0].Name != "github" || body.Data[0].Label != "GitHub" {
		t.Errorf("provider 0 mismatch: %+v", body.Data[0])
	}
	if body.Data[1].Name != "google" || body.Data[1].Label != "Google" {
		t.Errorf("provider 1 mismatch: %+v", body.Data[1])
	}
}

// AC1 (empty): GET /api/auth/providers returns empty list when no providers enabled.
func TestOAuth_ListProviders_Empty(t *testing.T) {
	r := newFakeRepo()
	reg := provider.NewRegistry(provider.Config{})
	app, _ := setupOAuthApp(r, reg)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/providers", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("GET /api/auth/providers: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var body struct {
		Data []domain.ProviderItem `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Data) != 0 {
		t.Fatalf("want 0 providers, got %d", len(body.Data))
	}
}

// AC2: GET /api/auth/oauth/:provider redirects (302) to provider with state & PKCE,
// and sets a signed HttpOnly SameSite=Lax cookie with TTL 10 minutes.
func TestOAuth_Authorize(t *testing.T) {
	r := newFakeRepo()
	reg := provider.NewRegistry(provider.Config{})
	reg.Register(&fakeProvider{
		name:        "github",
		label:       "GitHub",
		authCodeURL: "https://github.com/login/oauth/authorize",
	})
	app, svc := setupOAuthApp(r, reg)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/oauth/github", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("GET /api/auth/oauth/github: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusFound {
		t.Fatalf("status: want 302, got %d", resp.StatusCode)
	}

	loc := resp.Header.Get("Location")
	if loc == "" || loc[:len("https://github.com/login/oauth/authorize")] != "https://github.com/login/oauth/authorize" {
		t.Fatalf("unexpected location header: %s", loc)
	}

	cookie := findCookie(resp, handler.OAuthStateCookie)
	if cookie == nil {
		t.Fatal("expected opus_oauth cookie to be set")
	}
	if !cookie.HttpOnly {
		t.Error("expected cookie to be HttpOnly")
	}
	if cookie.MaxAge != 600 {
		t.Errorf("cookie MaxAge: want 600, got %d", cookie.MaxAge)
	}
	if cookie.SameSite != http.SameSiteLaxMode {
		t.Errorf("cookie SameSite: want Lax, got %v", cookie.SameSite)
	}

	// Verify cookie signature
	state, verifier, ok := svc.VerifyOAuthCookie(cookie.Value)
	if !ok || state == "" || verifier == "" {
		t.Fatalf("failed to verify signed cookie: state=%q, verifier=%q, ok=%v", state, verifier, ok)
	}
}

// AC5: Unknown or disabled provider returns 404 PROVIDER_NOT_FOUND.
func TestOAuth_Authorize_UnknownProvider(t *testing.T) {
	r := newFakeRepo()
	reg := provider.NewRegistry(provider.Config{})
	app, _ := setupOAuthApp(r, reg)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/oauth/unknown", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("GET /api/auth/oauth/unknown: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: want 404, got %d", resp.StatusCode)
	}

	var body httpx.ErrorResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Error != "PROVIDER_NOT_FOUND" {
		t.Errorf("error: want PROVIDER_NOT_FOUND, got %s", body.Error)
	}
}

// AC5: Callback with unknown or disabled provider returns 404 PROVIDER_NOT_FOUND.
func TestOAuth_Callback_UnknownProvider(t *testing.T) {
	r := newFakeRepo()
	reg := provider.NewRegistry(provider.Config{})
	app, _ := setupOAuthApp(r, reg)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/oauth/unknown/callback?code=foo&state=bar", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("GET /api/auth/oauth/unknown/callback: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: want 404, got %d", resp.StatusCode)
	}

	var body httpx.ErrorResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Error != "PROVIDER_NOT_FOUND" {
		t.Errorf("error: want PROVIDER_NOT_FOUND, got %s", body.Error)
	}
}

// AC3 & AC7: State mismatch (missing cookie, mismatched query state, or tampered signature)
// rejects with redirect to /login?error=STATE_MISMATCH.
func TestOAuth_Callback_StateMismatch(t *testing.T) {
	r := newFakeRepo()
	reg := provider.NewRegistry(provider.Config{})
	reg.Register(&fakeProvider{name: "github", label: "GitHub"})
	app, svc := setupOAuthApp(r, reg)

	// 1. Missing cookie
	req1 := httptest.NewRequest(http.MethodGet, "/api/auth/oauth/github/callback?code=c1&state=s1", nil)
	resp1, err := app.Test(req1)
	if err != nil {
		t.Fatalf("callback: %v", err)
	}
	_ = resp1.Body.Close()
	if resp1.StatusCode != http.StatusFound || resp1.Header.Get("Location") != "/login?error=STATE_MISMATCH" {
		t.Errorf("missing cookie: got status %d, loc %q", resp1.StatusCode, resp1.Header.Get("Location"))
	}

	// 2. Query state != cookie state
	cookieVal := svc.SignOAuthCookie("state-in-cookie", "verifier-val")
	req2 := httptest.NewRequest(http.MethodGet, "/api/auth/oauth/github/callback?code=c1&state=state-in-query", nil)
	req2.Header.Set("Cookie", handler.OAuthStateCookie+"="+cookieVal)
	resp2, err := app.Test(req2)
	if err != nil {
		t.Fatalf("callback: %v", err)
	}
	_ = resp2.Body.Close()
	if resp2.StatusCode != http.StatusFound || resp2.Header.Get("Location") != "/login?error=STATE_MISMATCH" {
		t.Errorf("mismatched state: got status %d, loc %q", resp2.StatusCode, resp2.Header.Get("Location"))
	}

	// 3. Tampered cookie signature
	req3 := httptest.NewRequest(http.MethodGet, "/api/auth/oauth/github/callback?code=c1&state=state-in-cookie", nil)
	req3.Header.Set("Cookie", handler.OAuthStateCookie+"="+cookieVal+"tampered")
	resp3, err := app.Test(req3)
	if err != nil {
		t.Fatalf("callback: %v", err)
	}
	_ = resp3.Body.Close()
	if resp3.StatusCode != http.StatusFound || resp3.Header.Get("Location") != "/login?error=STATE_MISMATCH" {
		t.Errorf("tampered cookie: got status %d, loc %q", resp3.StatusCode, resp3.Header.Get("Location"))
	}
}

// AC3 & AC7: Replayed state is rejected because state cookie is cleared after use.
func TestOAuth_Callback_ReplayedState(t *testing.T) {
	r := newFakeRepo()
	owner := &domain.User{ID: uuid.New(), Email: "owner@example.com"}
	r.owner = owner
	_, _ = r.CreateAccount(context.Background(), owner.ID, "github", "gh-123", "owner@example.com")

	reg := provider.NewRegistry(provider.Config{})
	reg.Register(&fakeProvider{
		name: "github",
		profile: provider.Profile{
			ProviderUserID: "gh-123",
			Email:          "owner@example.com",
			EmailVerified:  true,
		},
	})
	app, svc := setupOAuthApp(r, reg)

	const state = "unique-state-123"
	cookieVal := svc.SignOAuthCookie(state, "verifier-val")

	// First request succeeds
	req1 := httptest.NewRequest(http.MethodGet, "/api/auth/oauth/github/callback?code=c1&state="+state, nil)
	req1.Header.Set("Cookie", handler.OAuthStateCookie+"="+cookieVal)
	resp1, err := app.Test(req1)
	if err != nil {
		t.Fatalf("first callback: %v", err)
	}
	defer func() { _ = resp1.Body.Close() }()

	if resp1.StatusCode != http.StatusFound || resp1.Header.Get("Location") != "/" {
		t.Fatalf("first callback: want 302 to /, got %d to %s", resp1.StatusCode, resp1.Header.Get("Location"))
	}

	// Verify the state cookie is cleared in response
	clearedCookie := findCookie(resp1, handler.OAuthStateCookie)
	if clearedCookie == nil || clearedCookie.MaxAge > 0 {
		t.Fatal("expected opus_oauth cookie to be cleared (MaxAge <= 0)")
	}

	// Second request replay (reusing the same cookie value) fails because the browser cleared it.
	// Even if a client illegally resends the cleared cookie:
	// If no cookie is sent (normal browser after clear):
	req2 := httptest.NewRequest(http.MethodGet, "/api/auth/oauth/github/callback?code=c1&state="+state, nil)
	resp2, err := app.Test(req2)
	if err != nil {
		t.Fatalf("second callback: %v", err)
	}
	defer func() { _ = resp2.Body.Close() }()

	if resp2.StatusCode != http.StatusFound || resp2.Header.Get("Location") != "/login?error=STATE_MISMATCH" {
		t.Errorf("replay without cookie: want redirect to /login?error=STATE_MISMATCH, got %s", resp2.Header.Get("Location"))
	}
}

// AC4a, AC6 & AC7: Existing account -> log in as its user, set session cookie, redirect /.
func TestOAuth_Callback_ValidLoginExistingAccount(t *testing.T) {
	r := newFakeRepo()
	owner := &domain.User{ID: uuid.New(), Email: "owner@example.com"}
	r.owner = owner
	_, _ = r.CreateAccount(context.Background(), owner.ID, "github", "gh-user-999", "owner@example.com")

	reg := provider.NewRegistry(provider.Config{})
	reg.Register(&fakeProvider{
		name: "github",
		profile: provider.Profile{
			ProviderUserID: "gh-user-999",
			Email:          "owner@example.com",
			EmailVerified:  true,
		},
	})
	app, svc := setupOAuthApp(r, reg)

	const state = "valid-state-abc"
	cookieVal := svc.SignOAuthCookie(state, "verifier-val")

	req := httptest.NewRequest(http.MethodGet, "/api/auth/oauth/github/callback?code=c1&state="+state, nil)
	req.Header.Set("Cookie", handler.OAuthStateCookie+"="+cookieVal)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("callback: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusFound {
		t.Fatalf("status: want 302, got %d", resp.StatusCode)
	}
	if loc := resp.Header.Get("Location"); loc != "/" {
		t.Errorf("location: want /, got %s", loc)
	}

	sessCookie := findCookie(resp, handler.SessionCookie)
	if sessCookie == nil || sessCookie.Value == "" {
		t.Fatal("expected opus_session cookie to be set")
	}

	// Verify session was created in repo
	hash := computeHash(sessCookie.Value, testSecret)
	sess, err := r.FindByTokenHash(context.Background(), hash)
	if err != nil || sess == nil {
		t.Fatalf("session not found in repo: %v", err)
	}
	if sess.UserID != owner.ID {
		t.Errorf("session UserID: want %s, got %s", owner.ID, sess.UserID)
	}
}

// AC4b, AC6 & AC7: Auto-link by verified email when email equals owner email.
func TestOAuth_Callback_AutoLinkVerifiedEmail(t *testing.T) {
	r := newFakeRepo()
	owner := &domain.User{ID: uuid.New(), Email: "owner@example.com"}
	r.owner = owner

	reg := provider.NewRegistry(provider.Config{})
	reg.Register(&fakeProvider{
		name: "github",
		profile: provider.Profile{
			ProviderUserID: "gh-new-user-1",
			Email:          "OWNER@example.com", // case-insensitive check
			EmailVerified:  true,
		},
	})
	app, svc := setupOAuthApp(r, reg)

	const state = "link-state-xyz"
	cookieVal := svc.SignOAuthCookie(state, "verifier-val")

	req := httptest.NewRequest(http.MethodGet, "/api/auth/oauth/github/callback?code=c1&state="+state, nil)
	req.Header.Set("Cookie", handler.OAuthStateCookie+"="+cookieVal)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("callback: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusFound {
		t.Fatalf("status: want 302, got %d", resp.StatusCode)
	}
	if loc := resp.Header.Get("Location"); loc != "/" {
		t.Errorf("location: want /, got %s", loc)
	}

	sessCookie := findCookie(resp, handler.SessionCookie)
	if sessCookie == nil || sessCookie.Value == "" {
		t.Fatal("expected opus_session cookie to be set")
	}

	// Verify Account link was created in repo
	acc, err := r.FindAccount(context.Background(), "github", "gh-new-user-1")
	if err != nil || acc == nil {
		t.Fatalf("account link not created in repo: %v", err)
	}
	if acc.UserID != owner.ID {
		t.Errorf("account UserID: want %s, got %s", owner.ID, acc.UserID)
	}
}

// AC4c & AC7: Unverified email rejected -> ACCOUNT_NOT_ALLOWED.
func TestOAuth_Callback_UnverifiedEmailRejected(t *testing.T) {
	r := newFakeRepo()
	owner := &domain.User{ID: uuid.New(), Email: "owner@example.com"}
	r.owner = owner

	reg := provider.NewRegistry(provider.Config{})
	reg.Register(&fakeProvider{
		name: "github",
		profile: provider.Profile{
			ProviderUserID: "gh-unverified",
			Email:          "owner@example.com",
			EmailVerified:  false, // NOT verified
		},
	})
	app, svc := setupOAuthApp(r, reg)

	const state = "unverified-state"
	cookieVal := svc.SignOAuthCookie(state, "verifier-val")

	req := httptest.NewRequest(http.MethodGet, "/api/auth/oauth/github/callback?code=c1&state="+state, nil)
	req.Header.Set("Cookie", handler.OAuthStateCookie+"="+cookieVal)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("callback: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusFound {
		t.Fatalf("status: want 302, got %d", resp.StatusCode)
	}
	if loc := resp.Header.Get("Location"); loc != "/login?error=ACCOUNT_NOT_ALLOWED" {
		t.Errorf("location: want /login?error=ACCOUNT_NOT_ALLOWED, got %s", loc)
	}

	sessCookie := findCookie(resp, handler.SessionCookie)
	if sessCookie != nil && sessCookie.Value != "" {
		t.Error("expected no session cookie on rejection")
	}
}

// AC4c: Email mismatch rejected -> ACCOUNT_NOT_ALLOWED.
func TestOAuth_Callback_EmailMismatchRejected(t *testing.T) {
	r := newFakeRepo()
	owner := &domain.User{ID: uuid.New(), Email: "owner@example.com"}
	r.owner = owner

	reg := provider.NewRegistry(provider.Config{})
	reg.Register(&fakeProvider{
		name: "github",
		profile: provider.Profile{
			ProviderUserID: "gh-stranger",
			Email:          "stranger@example.com",
			EmailVerified:  true,
		},
	})
	app, svc := setupOAuthApp(r, reg)

	const state = "stranger-state"
	cookieVal := svc.SignOAuthCookie(state, "verifier-val")

	req := httptest.NewRequest(http.MethodGet, "/api/auth/oauth/github/callback?code=c1&state="+state, nil)
	req.Header.Set("Cookie", handler.OAuthStateCookie+"="+cookieVal)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("callback: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusFound {
		t.Fatalf("status: want 302, got %d", resp.StatusCode)
	}
	if loc := resp.Header.Get("Location"); loc != "/login?error=ACCOUNT_NOT_ALLOWED" {
		t.Errorf("location: want /login?error=ACCOUNT_NOT_ALLOWED, got %s", loc)
	}
}

// AC5: When no owner exists yet -> SETUP_REQUIRED.
func TestOAuth_Callback_SetupRequired(t *testing.T) {
	r := newFakeRepo()
	r.owner = nil // no owner exists

	reg := provider.NewRegistry(provider.Config{})
	reg.Register(&fakeProvider{
		name: "github",
		profile: provider.Profile{
			ProviderUserID: "gh-anyone",
			Email:          "anyone@example.com",
			EmailVerified:  true,
		},
	})
	app, svc := setupOAuthApp(r, reg)

	const state = "no-owner-state"
	cookieVal := svc.SignOAuthCookie(state, "verifier-val")

	req := httptest.NewRequest(http.MethodGet, "/api/auth/oauth/github/callback?code=c1&state="+state, nil)
	req.Header.Set("Cookie", handler.OAuthStateCookie+"="+cookieVal)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("callback: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusFound {
		t.Fatalf("status: want 302, got %d", resp.StatusCode)
	}
	if loc := resp.Header.Get("Location"); loc != "/login?error=SETUP_REQUIRED" {
		t.Errorf("location: want /login?error=SETUP_REQUIRED, got %s", loc)
	}
}

// Provider returned error in query param -> redirect /login?error=access_denied.
func TestOAuth_Callback_ProviderErrorQuery(t *testing.T) {
	r := newFakeRepo()
	reg := provider.NewRegistry(provider.Config{})
	reg.Register(&fakeProvider{name: "github", label: "GitHub"})
	app, _ := setupOAuthApp(r, reg)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/oauth/github/callback?error=access_denied", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("callback: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusFound {
		t.Fatalf("status: want 302, got %d", resp.StatusCode)
	}
	if loc := resp.Header.Get("Location"); loc != "/login?error=access_denied" {
		t.Errorf("location: want /login?error=access_denied, got %s", loc)
	}
}

// Exchange failure -> redirect /login?error=...
func TestOAuth_Callback_ExchangeError(t *testing.T) {
	r := newFakeRepo()
	r.owner = &domain.User{ID: uuid.New(), Email: "owner@example.com"}

	reg := provider.NewRegistry(provider.Config{})
	reg.Register(&fakeProvider{
		name:        "github",
		exchangeErr: errors.New("exchange_failed"),
	})
	app, svc := setupOAuthApp(r, reg)

	const state = "exchange-err-state"
	cookieVal := svc.SignOAuthCookie(state, "verifier-val")

	req := httptest.NewRequest(http.MethodGet, "/api/auth/oauth/github/callback?code=c1&state="+state, nil)
	req.Header.Set("Cookie", handler.OAuthStateCookie+"="+cookieVal)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("callback: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusFound {
		t.Fatalf("status: want 302, got %d", resp.StatusCode)
	}
	if loc := resp.Header.Get("Location"); loc != "/login?error=exchange_failed" {
		t.Errorf("location: want /login?error=exchange_failed, got %s", loc)
	}
}
