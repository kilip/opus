package handler_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
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

// --- fakeSetupRepo ---

type fakeSetupRepo struct {
	mu       sync.Mutex
	users    map[string]*domain.User // keyed by email
	sessions map[string]*domain.Session
	otpStore map[string]*domain.OTP
}

func newFakeSetupRepo() *fakeSetupRepo {
	return &fakeSetupRepo{
		users:    map[string]*domain.User{},
		sessions: map[string]*domain.Session{},
		otpStore: map[string]*domain.OTP{},
	}
}

func (f *fakeSetupRepo) CreateSession(_ context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) (*domain.Session, error) {
	s := &domain.Session{
		ID:        uuid.New(),
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}
	f.mu.Lock()
	f.sessions[tokenHash] = s
	f.mu.Unlock()
	return s, nil
}

func (f *fakeSetupRepo) FindByTokenHash(_ context.Context, hash string) (*domain.Session, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s, ok := f.sessions[hash]
	if !ok {
		return nil, domain.ErrUnauthorized
	}
	return s, nil
}

func (f *fakeSetupRepo) DeleteByTokenHash(_ context.Context, hash string) error {
	f.mu.Lock()
	delete(f.sessions, hash)
	f.mu.Unlock()
	return nil
}

func (f *fakeSetupRepo) CountUsers(_ context.Context) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.users), nil
}

func (f *fakeSetupRepo) CreateUserAndSession(_ context.Context, email, tokenHash string, expiresAt time.Time) (*domain.Session, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.users) > 0 {
		return nil, fmt.Errorf("unique constraint: user already exists")
	}
	u := &domain.User{ID: uuid.New(), Email: email}
	f.users[email] = u
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

func (f *fakeSetupRepo) FindAccount(_ context.Context, _, _ string) (*domain.Account, error) {
	return nil, nil
}

func (f *fakeSetupRepo) CreateAccount(_ context.Context, _ uuid.UUID, _, _, _ string) (*domain.Account, error) {
	return nil, nil
}

func (f *fakeSetupRepo) GetOwner(_ context.Context) (*domain.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, u := range f.users {
		return u, nil
	}
	return nil, nil
}

var _ repo.Repo = (*fakeSetupRepo)(nil)

// --- fakeSetupOTPRepo ---

type fakeSetupOTPRepo struct {
	mu   sync.Mutex
	otps map[string]*domain.OTP
}

func newFakeSetupOTPRepo() *fakeSetupOTPRepo {
	return &fakeSetupOTPRepo{otps: map[string]*domain.OTP{}}
}

func (f *fakeSetupOTPRepo) CreateOTP(_ context.Context, email, codeHash string, expiresAt time.Time) (*domain.OTP, error) {
	otp := &domain.OTP{
		ID:        uuid.New(),
		Email:     email,
		CodeHash:  codeHash,
		ExpiresAt: expiresAt,
	}
	f.mu.Lock()
	f.otps[email] = otp
	f.mu.Unlock()
	return otp, nil
}

func (f *fakeSetupOTPRepo) FindByEmail(_ context.Context, email string) (*domain.OTP, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	otp, ok := f.otps[email]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return otp, nil
}

func (f *fakeSetupOTPRepo) DeleteByEmail(_ context.Context, email string) error {
	f.mu.Lock()
	delete(f.otps, email)
	f.mu.Unlock()
	return nil
}

func (f *fakeSetupOTPRepo) FindUserByEmail(_ context.Context, _ string) (*domain.User, error) {
	return nil, nil
}

func (f *fakeSetupOTPRepo) IncrementAttempts(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}

var _ repo.OTPRepo = (*fakeSetupOTPRepo)(nil)

// --- helpers ---

func newSetupApp(r *fakeSetupRepo, otpR *fakeSetupOTPRepo) *fiber.App {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(logger)})
	cc := handler.CookieConfig{Secure: false, MaxAge: 86400}
	h := handler.New(service.New(r, otpR, &fakeMailer{}, service.Config{Secret: testSecret, SessionTTL: 24 * time.Hour}), cc)
	h.RegisterSetup(app.Group("/setup"))
	return app
}

func doJSON(t *testing.T, app *fiber.App, method, path string, body io.Reader) *http.Response {
	t.Helper()
	req := httptest.NewRequest(method, path, body)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	return resp
}

// --- tests ---

// AC1: status before setup → needs_setup: true
func TestSetupStatus_NeedsSetup(t *testing.T) {
	r := newFakeSetupRepo()
	app := newSetupApp(r, newFakeSetupOTPRepo())
	req := httptest.NewRequest(http.MethodGet, "/setup/status", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var env httpx.Response[domain.SetupStatusResponse]
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		t.Fatal(err)
	}
	if !env.Data.NeedsSetup {
		t.Fatal("expected needs_setup=true before owner exists")
	}
}

// AC5: status after setup → needs_setup: false
func TestSetupStatus_AfterSetup(t *testing.T) {
	r := newFakeSetupRepo()
	otpR := newFakeSetupOTPRepo()
	app := newSetupApp(r, otpR)

	// seed a user directly
	r.mu.Lock()
	r.users["owner@example.com"] = &domain.User{ID: uuid.New(), Email: "owner@example.com"}
	r.mu.Unlock()

	req := httptest.NewRequest(http.MethodGet, "/setup/status", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var env httpx.Response[domain.SetupStatusResponse]
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		t.Fatal(err)
	}
	if env.Data.NeedsSetup {
		t.Fatal("expected needs_setup=false after owner exists")
	}
}

// AC2: POST /setup/otp requests OTP successfully
func TestSetupRequestOTP_HappyPath(t *testing.T) {
	r := newFakeSetupRepo()
	app := newSetupApp(r, newFakeSetupOTPRepo())
	resp := doJSON(t, app, http.MethodPost, "/setup/otp", jsonBody(map[string]string{"email": "owner@example.com"}))
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", resp.StatusCode)
	}
}

// AC5: POST /setup/otp after owner exists → 409 SETUP_CLOSED
func TestSetupRequestOTP_AfterSetup(t *testing.T) {
	r := newFakeSetupRepo()
	r.mu.Lock()
	r.users["owner@example.com"] = &domain.User{ID: uuid.New(), Email: "owner@example.com"}
	r.mu.Unlock()

	app := newSetupApp(r, newFakeSetupOTPRepo())
	resp := doJSON(t, app, http.MethodPost, "/setup/otp", jsonBody(map[string]string{"email": "owner@example.com"}))
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d", resp.StatusCode)
	}
}

// AC3: POST /setup happy path
func TestSetupComplete_HappyPath(t *testing.T) {
	r := newFakeSetupRepo()
	otpR := newFakeSetupOTPRepo()
	app := newSetupApp(r, otpR)

	const email = "owner@example.com"
	const code = "123456"
	codeHash := computeHash(code, testSecret)
	_, _ = otpR.CreateOTP(context.Background(), email, codeHash, time.Now().Add(time.Hour))

	resp := doJSON(t, app, http.MethodPost, "/setup", jsonBody(map[string]string{"email": email, "code": code}))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var env httpx.Response[domain.UserResponse]
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		t.Fatal(err)
	}
	if env.Data.Email != email {
		t.Fatalf("expected email %q, got %q", email, env.Data.Email)
	}
	// cookie set
	found := false
	for _, c := range resp.Cookies() {
		if c.Name == "opus_session" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected opus_session cookie to be set")
	}
}

// wrong code → 401
func TestSetupComplete_WrongCode(t *testing.T) {
	r := newFakeSetupRepo()
	otpR := newFakeSetupOTPRepo()
	app := newSetupApp(r, otpR)

	const email = "owner@example.com"
	codeHash := computeHash("correct", testSecret)
	_, _ = otpR.CreateOTP(context.Background(), email, codeHash, time.Now().Add(time.Hour))

	resp := doJSON(t, app, http.MethodPost, "/setup", jsonBody(map[string]string{"email": email, "code": "wrong"}))
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

// AC4: concurrent POST /setup → exactly one succeeds, rest get 409
func TestSetupComplete_Concurrency(t *testing.T) {
	r := newFakeSetupRepo()
	otpR := newFakeSetupOTPRepo()
	app := newSetupApp(r, otpR)

	const email = "owner@example.com"
	const code = "654321"
	codeHash := computeHash(code, testSecret)
	_, _ = otpR.CreateOTP(context.Background(), email, codeHash, time.Now().Add(time.Hour))

	const n = 10
	results := make([]int, n)
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		i := i
		go func() {
			defer wg.Done()
			// re-seed OTP for each goroutine (single-use in real flow, but here we re-add to allow all to reach CreateUserAndSession)
			otpR.mu.Lock()
			otpR.otps[email] = &domain.OTP{
				ID:        uuid.New(),
				Email:     email,
				CodeHash:  codeHash,
				ExpiresAt: time.Now().Add(time.Hour),
			}
			otpR.mu.Unlock()
			resp := doJSON(t, app, http.MethodPost, "/setup", jsonBody(map[string]string{"email": email, "code": code}))
			results[i] = resp.StatusCode
		}()
	}
	wg.Wait()

	ok := 0
	for _, s := range results {
		if s == http.StatusOK {
			ok++
		}
	}
	// AC4: exactly one goroutine must succeed; at most one owner user created.
	if ok != 1 {
		t.Fatalf("expected exactly 1 success, got %d (results: %v)", ok, results)
	}
	r.mu.Lock()
	userCount := len(r.users)
	r.mu.Unlock()
	if userCount != 1 {
		t.Fatalf("expected exactly 1 user, got %d", userCount)
	}
}
