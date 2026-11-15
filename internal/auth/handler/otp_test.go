package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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
	"github.com/kilip/opus/internal/platform/httpx"
	"github.com/kilip/opus/internal/platform/mailer"
)

func newDiscardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// --- fakeOTPRepo ---

type fakeOTPRepo struct {
	otps  map[string]*domain.OTP // keyed by email
	users map[string]*domain.User
}

func newFakeOTPRepo() *fakeOTPRepo {
	return &fakeOTPRepo{
		otps:  map[string]*domain.OTP{},
		users: map[string]*domain.User{},
	}
}

func (f *fakeOTPRepo) CreateOTP(_ context.Context, email, codeHash string, expiresAt time.Time) (*domain.OTP, error) {
	delete(f.otps, email)
	o := &domain.OTP{
		ID:        uuid.New(),
		Email:     email,
		CodeHash:  codeHash,
		ExpiresAt: expiresAt,
		Attempts:  0,
		CreatedAt: time.Now(),
	}
	f.otps[email] = o
	return o, nil
}

func (f *fakeOTPRepo) FindByEmail(_ context.Context, email string) (*domain.OTP, error) {
	o, ok := f.otps[email]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return o, nil
}

func (f *fakeOTPRepo) DeleteByEmail(_ context.Context, email string) error {
	delete(f.otps, email)
	return nil
}

func (f *fakeOTPRepo) IncrementAttempts(_ context.Context, id uuid.UUID) (int, error) {
	for _, o := range f.otps {
		if o.ID == id {
			o.Attempts++
			return o.Attempts, nil
		}
	}
	return 0, fmt.Errorf("not found")
}

func (f *fakeOTPRepo) FindUserByEmail(_ context.Context, email string) (*domain.User, error) {
	u, ok := f.users[email]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return u, nil
}

var _ repo.OTPRepo = (*fakeOTPRepo)(nil)

// --- fakeMailer ---

type fakeMailer struct {
	sent []string
}

func (m *fakeMailer) Send(_ context.Context, to, _, _ string) error {
	m.sent = append(m.sent, to)
	return nil
}

var _ mailer.Mailer = (*fakeMailer)(nil)

// --- helpers ---

func setupOTPApp(t *testing.T, sessRepo repo.Repo, otpRepo repo.OTPRepo, m mailer.Mailer, now func() time.Time) *fiber.App {
	t.Helper()
	cc := handler.CookieConfig{Secure: false, MaxAge: 86400}
	h := handler.NewWithOTP(sessRepo, otpRepo, m, testSecret, cc)
	if now != nil {
		h.SetNow(now)
	}
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(newDiscardLogger())})
	g := app.Group("/auth")
	h.RegisterPublic(g)
	h.Register(g)
	return app
}

func jsonBody(v any) *bytes.Buffer {
	b, _ := json.Marshal(v)
	return bytes.NewBuffer(b)
}

func postJSON(t *testing.T, app *fiber.App, path string, body any) *http.Response {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, jsonBody(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("POST %s: %v", path, err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	return resp
}

// codeHashFor returns the HMAC hash of a code the same way the handler does.
func codeHashFor(code string) string {
	return computeHash(code, testSecret)
}

// --- OTP request tests ---

func TestRequestOTP_UnknownEmail_Returns202(t *testing.T) {
	otp := newFakeOTPRepo() // no users registered
	m := &fakeMailer{}
	app := setupOTPApp(t, newFakeRepo(), otp, m, nil)

	resp := postJSON(t, app, "/auth/otp/request", map[string]string{"email": "ghost@example.com"})
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("want 202, got %d", resp.StatusCode)
	}
	if len(m.sent) != 0 {
		t.Fatal("should not send mail for unknown email")
	}
}

func TestRequestOTP_KnownEmail_Returns202AndSendsMail(t *testing.T) {
	otpRepo := newFakeOTPRepo()
	otpRepo.users["owner@example.com"] = &domain.User{ID: uuid.New(), Email: "owner@example.com"}
	m := &fakeMailer{}
	app := setupOTPApp(t, newFakeRepo(), otpRepo, m, nil)

	resp := postJSON(t, app, "/auth/otp/request", map[string]string{"email": "owner@example.com"})
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("want 202, got %d", resp.StatusCode)
	}
	if len(m.sent) != 1 || m.sent[0] != "owner@example.com" {
		t.Fatalf("want mail sent to owner@example.com, got %v", m.sent)
	}
	if _, ok := otpRepo.otps["owner@example.com"]; !ok {
		t.Fatal("OTP should be stored")
	}
}

func TestRequestOTP_NewRequestInvalidatesOld(t *testing.T) {
	otpRepo := newFakeOTPRepo()
	email := "owner@example.com"
	otpRepo.users[email] = &domain.User{ID: uuid.New(), Email: email}
	m := &fakeMailer{}
	app := setupOTPApp(t, newFakeRepo(), otpRepo, m, nil)

	postJSON(t, app, "/auth/otp/request", map[string]string{"email": email})
	firstHash := otpRepo.otps[email].CodeHash

	postJSON(t, app, "/auth/otp/request", map[string]string{"email": email})
	secondHash := otpRepo.otps[email].CodeHash

	if firstHash == secondHash {
		t.Fatal("second request should produce a different code hash")
	}
}

// --- OTP verify tests ---

func TestVerifyOTP_Success(t *testing.T) {
	email := "owner@example.com"
	userID := uuid.New()
	otpRepo := newFakeOTPRepo()
	otpRepo.users[email] = &domain.User{ID: userID, Email: email}
	sessRepo := newFakeRepo()

	fixedNow := time.Now()
	otpRepo.otps[email] = &domain.OTP{
		ID:        uuid.New(),
		Email:     email,
		CodeHash:  codeHashFor("123456"),
		ExpiresAt: fixedNow.Add(10 * time.Minute),
	}

	app := setupOTPApp(t, sessRepo, otpRepo, &fakeMailer{}, func() time.Time { return fixedNow })
	resp := postJSON(t, app, "/auth/otp/verify", map[string]string{"email": email, "code": "123456"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}
	// OTP should be deleted (single-use).
	if _, ok := otpRepo.otps[email]; ok {
		t.Fatal("OTP should be deleted after successful verify")
	}
	// Session cookie should be set.
	hasCookie := false
	for _, c := range resp.Cookies() {
		if c.Name == "opus_session" && c.Value != "" {
			hasCookie = true
		}
	}
	if !hasCookie {
		t.Fatal("session cookie not set")
	}
	// Response body should include user.
	var env httpx.Response[handler.OTPVerifyResponse]
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		t.Fatal(err)
	}
	if env.Data.Email != email {
		t.Fatalf("want email %q, got %q", email, env.Data.Email)
	}
}

func TestVerifyOTP_WrongCode_Returns401AndIncrementsAttempts(t *testing.T) {
	email := "owner@example.com"
	otpRepo := newFakeOTPRepo()
	otpRepo.users[email] = &domain.User{ID: uuid.New(), Email: email}
	fixedNow := time.Now()
	otpRepo.otps[email] = &domain.OTP{
		ID:        uuid.New(),
		Email:     email,
		CodeHash:  codeHashFor("999999"),
		ExpiresAt: fixedNow.Add(10 * time.Minute),
	}

	app := setupOTPApp(t, newFakeRepo(), otpRepo, &fakeMailer{}, func() time.Time { return fixedNow })
	resp := postJSON(t, app, "/auth/otp/verify", map[string]string{"email": email, "code": "000000"})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", resp.StatusCode)
	}
	if otpRepo.otps[email].Attempts != 1 {
		t.Fatalf("want 1 attempt, got %d", otpRepo.otps[email].Attempts)
	}
}

func TestVerifyOTP_ExpiredCode_Returns401(t *testing.T) {
	email := "owner@example.com"
	otpRepo := newFakeOTPRepo()
	otpRepo.users[email] = &domain.User{ID: uuid.New(), Email: email}
	past := time.Now().Add(-1 * time.Minute)
	otpRepo.otps[email] = &domain.OTP{
		ID:        uuid.New(),
		Email:     email,
		CodeHash:  codeHashFor("123456"),
		ExpiresAt: past,
	}

	fixedNow := time.Now()
	app := setupOTPApp(t, newFakeRepo(), otpRepo, &fakeMailer{}, func() time.Time { return fixedNow })
	resp := postJSON(t, app, "/auth/otp/verify", map[string]string{"email": email, "code": "123456"})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("want 401 (expired), got %d", resp.StatusCode)
	}
}

func TestVerifyOTP_SingleUse(t *testing.T) {
	email := "owner@example.com"
	userID := uuid.New()
	otpRepo := newFakeOTPRepo()
	otpRepo.users[email] = &domain.User{ID: userID, Email: email}
	fixedNow := time.Now()
	otpRepo.otps[email] = &domain.OTP{
		ID:        uuid.New(),
		Email:     email,
		CodeHash:  codeHashFor("123456"),
		ExpiresAt: fixedNow.Add(10 * time.Minute),
	}

	app := setupOTPApp(t, newFakeRepo(), otpRepo, &fakeMailer{}, func() time.Time { return fixedNow })

	// First verify: success
	resp1 := postJSON(t, app, "/auth/otp/verify", map[string]string{"email": email, "code": "123456"})
	if resp1.StatusCode != http.StatusOK {
		t.Fatalf("first verify: want 200, got %d", resp1.StatusCode)
	}
	// Second verify: OTP is gone → 401
	resp2 := postJSON(t, app, "/auth/otp/verify", map[string]string{"email": email, "code": "123456"})
	if resp2.StatusCode != http.StatusUnauthorized {
		t.Fatalf("second verify: want 401, got %d", resp2.StatusCode)
	}
}

func TestVerifyOTP_AttemptsLockout(t *testing.T) {
	email := "owner@example.com"
	otpRepo := newFakeOTPRepo()
	otpRepo.users[email] = &domain.User{ID: uuid.New(), Email: email}
	fixedNow := time.Now()
	otpID := uuid.New()
	otpRepo.otps[email] = &domain.OTP{
		ID:        otpID,
		Email:     email,
		CodeHash:  codeHashFor("999999"),
		ExpiresAt: fixedNow.Add(10 * time.Minute),
		Attempts:  domain.MaxOTPAttempts - 1, // one attempt before lockout
	}

	app := setupOTPApp(t, newFakeRepo(), otpRepo, &fakeMailer{}, func() time.Time { return fixedNow })
	resp := postJSON(t, app, "/auth/otp/verify", map[string]string{"email": email, "code": "000000"})
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("want 429, got %d", resp.StatusCode)
	}
	// OTP should be invalidated after lockout.
	if _, ok := otpRepo.otps[email]; ok {
		t.Fatal("OTP should be deleted after lockout")
	}
}

func TestVerifyOTP_AlreadyLockedOut(t *testing.T) {
	email := "owner@example.com"
	otpRepo := newFakeOTPRepo()
	otpRepo.users[email] = &domain.User{ID: uuid.New(), Email: email}
	fixedNow := time.Now()
	otpRepo.otps[email] = &domain.OTP{
		ID:        uuid.New(),
		Email:     email,
		CodeHash:  codeHashFor("999999"),
		ExpiresAt: fixedNow.Add(10 * time.Minute),
		Attempts:  domain.MaxOTPAttempts, // already at limit
	}

	app := setupOTPApp(t, newFakeRepo(), otpRepo, &fakeMailer{}, func() time.Time { return fixedNow })
	resp := postJSON(t, app, "/auth/otp/verify", map[string]string{"email": email, "code": "999999"})
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("want 429 (already locked), got %d", resp.StatusCode)
	}
}
