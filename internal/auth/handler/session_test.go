package handler_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/kilip/opus/internal/auth/domain"
	"github.com/kilip/opus/internal/auth/handler"
	"github.com/kilip/opus/internal/auth/repo"
	"github.com/kilip/opus/internal/auth/service"
	"github.com/kilip/opus/internal/platform/httpx"
)

// --- fake repo ---

type fakeRepo struct {
	sessions map[string]*domain.Session
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{sessions: map[string]*domain.Session{}}
}

func (f *fakeRepo) CreateSession(_ context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) (*domain.Session, error) {
	s := &domain.Session{
		ID:        uuid.New(),
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
		User:      &domain.User{ID: userID, Email: "test@example.com"},
	}
	f.sessions[tokenHash] = s
	return s, nil
}

func (f *fakeRepo) FindByTokenHash(_ context.Context, hash string) (*domain.Session, error) {
	s, ok := f.sessions[hash]
	if !ok {
		return nil, domain.ErrUnauthorized
	}
	return s, nil
}

func (f *fakeRepo) DeleteByTokenHash(_ context.Context, hash string) error {
	delete(f.sessions, hash)
	return nil
}

func (f *fakeRepo) CountUsers(_ context.Context) (int, error) {
	return 0, nil // fakeRepo used in session tests; no users needed
}

func (f *fakeRepo) CreateUserAndSession(_ context.Context, _ string, _ string, _ time.Time) (*domain.Session, error) {
	return nil, nil // unused in session tests
}

var _ repo.Repo = (*fakeRepo)(nil)

// --- helpers ---

const testSecret = "test-secret"

// computeHash replicates the HMAC logic from service/auth.go.
func computeHash(raw, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(raw))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func setupApp(r repo.Repo) *fiber.App {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(logger)})
	cc := handler.CookieConfig{Secure: false, MaxAge: 86400}
	h := handler.New(service.New(r, nil, nil, service.Config{Secret: testSecret}), cc)
	h.Register(app.Group("/auth"))
	return app
}

func doReq(t *testing.T, app *fiber.App, method, path, sessionCookie string) *http.Response {
	t.Helper()
	r := httptest.NewRequest(method, path, nil)
	if sessionCookie != "" {
		r.Header.Set("Cookie", "opus_session="+sessionCookie)
	}
	resp, err := app.Test(r)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	return resp
}

// --- me tests ---

func TestMe_ValidSession(t *testing.T) {
	r := newFakeRepo()
	userID := uuid.New()
	hash := computeHash("valid-token", testSecret)
	_, _ = r.CreateSession(context.Background(), userID, hash, time.Now().Add(time.Hour))

	resp := doReq(t, setupApp(r), http.MethodGet, "/auth/me", "valid-token")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var env httpx.Response[domain.UserResponse]
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		t.Fatal(err)
	}
	if env.Data.ID != userID {
		t.Fatalf("expected user %s, got %s", userID, env.Data.ID)
	}
}

func TestMe_MissingCookie(t *testing.T) {
	resp := doReq(t, setupApp(newFakeRepo()), http.MethodGet, "/auth/me", "")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestMe_InvalidCookie(t *testing.T) {
	resp := doReq(t, setupApp(newFakeRepo()), http.MethodGet, "/auth/me", "bad-token")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestMe_ExpiredSession(t *testing.T) {
	r := newFakeRepo()
	hash := computeHash("expired-token", testSecret)
	_, _ = r.CreateSession(context.Background(), uuid.New(), hash, time.Now().Add(-time.Hour))

	resp := doReq(t, setupApp(r), http.MethodGet, "/auth/me", "expired-token")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

// --- logout tests ---

func TestLogout_WithSession(t *testing.T) {
	r := newFakeRepo()
	hash := computeHash("my-token", testSecret)
	_, _ = r.CreateSession(context.Background(), uuid.New(), hash, time.Now().Add(time.Hour))

	resp := doReq(t, setupApp(r), http.MethodPost, "/auth/logout", "my-token")
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}
	if _, ok := r.sessions[hash]; ok {
		t.Fatal("session should have been deleted")
	}
	cleared := false
	for _, c := range resp.Cookies() {
		if c.Name == "opus_session" && c.MaxAge < 0 {
			cleared = true
		}
	}
	if !cleared {
		t.Fatal("expected session cookie to be cleared (MaxAge < 0)")
	}
}

func TestLogout_NoSession(t *testing.T) {
	resp := doReq(t, setupApp(newFakeRepo()), http.MethodPost, "/auth/logout", "")
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}
}

// --- middleware tests (via fiber wiring) ---

func makeProtectedApp(r repo.Repo) *fiber.App {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(logger)})
	api := app.Group("/api")

	// public
	api.Get("/health", func(c fiber.Ctx) error { return httpx.OK(c, "ok") })

	// middleware
	api.Use(func(c fiber.Ctx) error {
		raw := c.Cookies("opus_session")
		if raw == "" {
			return httpx.Unauthorized()
		}
		hash := computeHash(raw, testSecret)
		sess, err := r.FindByTokenHash(c.Context(), hash)
		if err != nil || time.Now().After(sess.ExpiresAt) {
			return httpx.Unauthorized()
		}
		return c.Next()
	})

	// protected
	api.Get("/tasks", func(c fiber.Ctx) error { return httpx.OK(c, "tasks") })
	return app
}

func TestMiddleware_PublicHealthNoSession(t *testing.T) {
	app := makeProtectedApp(newFakeRepo())
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("health: expected 200, got %d", resp.StatusCode)
	}
}

func TestMiddleware_Protected_NoSession(t *testing.T) {
	app := makeProtectedApp(newFakeRepo())
	req := httptest.NewRequest(http.MethodGet, "/api/tasks", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestMiddleware_Protected_ValidSession(t *testing.T) {
	r := newFakeRepo()
	hash := computeHash("good-token", testSecret)
	_, _ = r.CreateSession(context.Background(), uuid.New(), hash, time.Now().Add(time.Hour))

	app := makeProtectedApp(r)
	req := httptest.NewRequest(http.MethodGet, "/api/tasks", nil)
	req.Header.Set("Cookie", "opus_session=good-token")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}
