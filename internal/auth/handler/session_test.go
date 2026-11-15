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
	"sort"
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
	accounts map[string]*domain.Account
	owner    *domain.User
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		sessions: map[string]*domain.Session{},
		accounts: map[string]*domain.Account{},
	}
}

func (f *fakeRepo) CreateSession(_ context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) (*domain.Session, error) {
	u := f.owner
	if u == nil || u.ID != userID {
		u = &domain.User{ID: userID, Email: "test@example.com"}
	}
	s := &domain.Session{
		ID:        uuid.New(),
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
		User:      u,
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
	if f.owner != nil {
		return 1, nil
	}
	return 0, nil
}

func (f *fakeRepo) CreateUserAndSession(_ context.Context, email, tokenHash string, expiresAt time.Time) (*domain.Session, error) {
	u := &domain.User{ID: uuid.New(), Email: email}
	f.owner = u
	s := &domain.Session{
		ID:        uuid.New(),
		UserID:    u.ID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
		User:      u,
	}
	f.sessions[tokenHash] = s
	return s, nil
}

func (f *fakeRepo) FindAccount(_ context.Context, provider, providerUserID string) (*domain.Account, error) {
	acc, ok := f.accounts[provider+":"+providerUserID]
	if !ok {
		return nil, nil
	}
	return acc, nil
}

func (f *fakeRepo) CreateAccount(_ context.Context, userID uuid.UUID, provider, providerUserID, email string) (*domain.Account, error) {
	acc := &domain.Account{
		ID:             uuid.New(),
		UserID:         userID,
		Provider:       provider,
		ProviderUserID: providerUserID,
		Email:          email,
		CreatedAt:      time.Now(),
	}
	f.accounts[provider+":"+providerUserID] = acc
	return acc, nil
}

func (f *fakeRepo) ListAccountsByUserID(_ context.Context, userID uuid.UUID) ([]*domain.Account, error) {
	var res []*domain.Account
	for _, acc := range f.accounts {
		if acc.UserID == userID {
			res = append(res, acc)
		}
	}
	sort.Slice(res, func(i, j int) bool {
		return res[i].CreatedAt.Before(res[j].CreatedAt)
	})
	return res, nil
}

func (f *fakeRepo) DeleteAccount(_ context.Context, userID, accountID uuid.UUID) error {
	for k, acc := range f.accounts {
		if acc.ID == accountID {
			if acc.UserID != userID {
				return domain.ErrAccountNotFound
			}
			delete(f.accounts, k)
			return nil
		}
	}
	return domain.ErrAccountNotFound
}

func (f *fakeRepo) GetOwner(_ context.Context) (*domain.User, error) {
	return f.owner, nil
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

// --- accounts tests ---

func TestAccounts_List_Unauthenticated(t *testing.T) {
	resp := doReq(t, setupApp(newFakeRepo()), http.MethodGet, "/auth/accounts", "")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestAccounts_List_Empty(t *testing.T) {
	r := newFakeRepo()
	userID := uuid.New()
	hash := computeHash("tok-1", testSecret)
	_, _ = r.CreateSession(context.Background(), userID, hash, time.Now().Add(time.Hour))

	resp := doReq(t, setupApp(r), http.MethodGet, "/auth/accounts", "tok-1")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var env httpx.Response[[]domain.AccountResponse]
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		t.Fatal(err)
	}
	if len(env.Data) != 0 {
		t.Fatalf("expected 0 accounts, got %d", len(env.Data))
	}
}

func TestAccounts_List_Success(t *testing.T) {
	r := newFakeRepo()
	user1 := uuid.New()
	user2 := uuid.New()
	hash1 := computeHash("tok-user1", testSecret)
	_, _ = r.CreateSession(context.Background(), user1, hash1, time.Now().Add(time.Hour))

	acc1, err := r.CreateAccount(context.Background(), user1, "github", "gh-1", "user1@github.com")
	if err != nil {
		t.Fatal(err)
	}
	acc2, err := r.CreateAccount(context.Background(), user1, "google", "goog-1", "user1@gmail.com")
	if err != nil {
		t.Fatal(err)
	}
	// user2 account should NOT appear in user1's list
	_, err = r.CreateAccount(context.Background(), user2, "github", "gh-2", "user2@github.com")
	if err != nil {
		t.Fatal(err)
	}

	resp := doReq(t, setupApp(r), http.MethodGet, "/auth/accounts", "tok-user1")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var env httpx.Response[[]domain.AccountResponse]
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		t.Fatal(err)
	}
	if len(env.Data) != 2 {
		t.Fatalf("expected 2 accounts, got %d", len(env.Data))
	}

	if env.Data[0].ID != acc1.ID || env.Data[0].Provider != "github" || env.Data[0].Email != "user1@github.com" || env.Data[0].CreatedAt.IsZero() {
		t.Fatalf("unexpected account 0: %+v", env.Data[0])
	}
	if env.Data[1].ID != acc2.ID || env.Data[1].Provider != "google" || env.Data[1].Email != "user1@gmail.com" || env.Data[1].CreatedAt.IsZero() {
		t.Fatalf("unexpected account 1: %+v", env.Data[1])
	}
}

func TestAccounts_Unlink_Unauthenticated(t *testing.T) {
	resp := doReq(t, setupApp(newFakeRepo()), http.MethodDelete, "/auth/accounts/"+uuid.NewString(), "")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestAccounts_Unlink_Success(t *testing.T) {
	r := newFakeRepo()
	userID := uuid.New()
	hash := computeHash("tok-unlink", testSecret)
	_, _ = r.CreateSession(context.Background(), userID, hash, time.Now().Add(time.Hour))

	acc, _ := r.CreateAccount(context.Background(), userID, "github", "gh-unlink", "unlink@example.com")

	resp := doReq(t, setupApp(r), http.MethodDelete, "/auth/accounts/"+acc.ID.String(), "tok-unlink")
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}

	// Verify account is removed
	accounts, _ := r.ListAccountsByUserID(context.Background(), userID)
	if len(accounts) != 0 {
		t.Fatalf("expected 0 accounts, got %d", len(accounts))
	}
}

func TestAccounts_Unlink_NotFound(t *testing.T) {
	r := newFakeRepo()
	userID := uuid.New()
	hash := computeHash("tok-404", testSecret)
	_, _ = r.CreateSession(context.Background(), userID, hash, time.Now().Add(time.Hour))

	// Non-existent UUID
	resp := doReq(t, setupApp(r), http.MethodDelete, "/auth/accounts/"+uuid.NewString(), "tok-404")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}

	// Malformed UUID string
	resp2 := doReq(t, setupApp(r), http.MethodDelete, "/auth/accounts/not-a-uuid", "tok-404")
	if resp2.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp2.StatusCode)
	}
}

func TestAccounts_Unlink_CrossUser(t *testing.T) {
	r := newFakeRepo()
	user1 := uuid.New()
	user2 := uuid.New()
	hash1 := computeHash("tok-u1", testSecret)
	_, _ = r.CreateSession(context.Background(), user1, hash1, time.Now().Add(time.Hour))

	// Account belongs to user2
	acc2, _ := r.CreateAccount(context.Background(), user2, "github", "gh-u2", "u2@example.com")

	// User1 tries to unlink user2's account -> 404
	resp := doReq(t, setupApp(r), http.MethodDelete, "/auth/accounts/"+acc2.ID.String(), "tok-u1")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}

	// Verify user2's account was NOT deleted
	accounts, _ := r.ListAccountsByUserID(context.Background(), user2)
	if len(accounts) != 1 {
		t.Fatalf("expected user2 to still have 1 account, got %d", len(accounts))
	}
}
