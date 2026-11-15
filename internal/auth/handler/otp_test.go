package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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
	"github.com/kilip/opus/internal/auth/service"
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
	svc := service.New(sessRepo, otpRepo, m, service.Config{
		Secret:     testSecret,
		SessionTTL: 24 * time.Hour,
		Now:        now,
	})
	h := handler.New(svc, cc)
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
	var env httpx.Response[domain.UserResponse]
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		t.Fatal(err)
	}
	if env.Data.Email != email {
		t.Fatalf("want email %q, got %q", email, env.Data.Email)
	}
	if len(sessRepo.auditLogs) != 1 {
		t.Fatalf("expected 1 audit log entry, got %d", len(sessRepo.auditLogs))
	}
	if sessRepo.auditLogs[0].event != domain.AuditEventLoginOTP {
		t.Errorf("expected audit event %q, got %q", domain.AuditEventLoginOTP, sessRepo.auditLogs[0].event)
	}
	if sessRepo.auditLogs[0].userID == nil || *sessRepo.auditLogs[0].userID != userID {
		t.Errorf("expected audit log userID %v, got %v", userID, sessRepo.auditLogs[0].userID)
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

	sessRepo := newFakeRepo()
	app := setupOTPApp(t, sessRepo, otpRepo, &fakeMailer{}, func() time.Time { return fixedNow })
	resp := postJSON(t, app, "/auth/otp/verify", map[string]string{"email": email, "code": "000000"})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", resp.StatusCode)
	}
	if otpRepo.otps[email].Attempts != 1 {
		t.Fatalf("want 1 attempt, got %d", otpRepo.otps[email].Attempts)
	}
	if len(sessRepo.auditLogs) != 1 {
		t.Fatalf("expected 1 audit log entry, got %d", len(sessRepo.auditLogs))
	}
	if sessRepo.auditLogs[0].event != domain.AuditEventOTPFailed {
		t.Errorf("expected audit event %q, got %q", domain.AuditEventOTPFailed, sessRepo.auditLogs[0].event)
	}
}

func TestVerifyOTP_AuditFailureDoesNotFailRequest(t *testing.T) {
	email := "owner@example.com"
	userID := uuid.New()
	otpRepo := newFakeOTPRepo()
	otpRepo.users[email] = &domain.User{ID: userID, Email: email}
	sessRepo := newFakeRepo()
	sessRepo.auditErr = errors.New("audit log write error")

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
		t.Fatalf("want 200 despite audit failure, got %d", resp.StatusCode)
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

// --- Rate limiting tests ---

func postJSONFromIP(t *testing.T, app *fiber.App, path string, body any, ip string) *http.Response {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, jsonBody(body))
	req.Header.Set("Content-Type", "application/json")
	if ip != "" {
		req.Header.Set("X-Forwarded-For", ip)
		req.RemoteAddr = ip + ":12345"
	}
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("POST %s: %v", path, err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	return resp
}

func checkRateLimitedResponse(t *testing.T, resp *http.Response) {
	t.Helper()
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("want status 429, got %d", resp.StatusCode)
	}
	retryAfter := resp.Header.Get("Retry-After")
	if retryAfter == "" {
		t.Fatal("expected Retry-After header on 429 response")
	}
	var errResp httpx.ErrorResponse
	if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}
	if errResp.Error != "RATE_LIMITED" {
		t.Fatalf("want error code RATE_LIMITED, got %q", errResp.Error)
	}
}

func setupAppWithRateLimit(t *testing.T, rl handler.RateLimitConfig) *fiber.App {
	t.Helper()
	sessRepo := newFakeRepo()
	otpRepo := newFakeOTPRepo()
	svc := service.New(sessRepo, otpRepo, &fakeMailer{}, service.Config{
		Secret:     testSecret,
		SessionTTL: 24 * time.Hour,
	})
	cc := handler.CookieConfig{Secure: false, MaxAge: 86400}
	h := handler.New(svc, cc, rl)
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(newDiscardLogger())})
	h.RegisterPublic(app.Group("/auth"))
	h.RegisterSetup(app.Group("/setup"))
	return app
}

func TestOTP_PerEmailLimit_AuthAndSetup(t *testing.T) {
	app := setupAppWithRateLimit(t, handler.RateLimitConfig{})

	// 3 requests allowed with different IPs for the same email
	for i := 1; i <= 3; i++ {
		resp := postJSONFromIP(t, app, "/auth/otp/request", map[string]string{"email": "user@example.com"}, fmt.Sprintf("192.168.1.%d", i))
		if resp.StatusCode != http.StatusAccepted {
			t.Fatalf("request %d: want 202, got %d", i, resp.StatusCode)
		}
	}

	// 4th request exceeds per-email limit
	resp := postJSONFromIP(t, app, "/auth/otp/request", map[string]string{"email": "user@example.com"}, "192.168.1.99")
	checkRateLimitedResponse(t, resp)

	// Another email from the same blocked IP should succeed (IP limit is 10)
	respOther := postJSONFromIP(t, app, "/auth/otp/request", map[string]string{"email": "other@example.com"}, "192.168.1.99")
	if respOther.StatusCode != http.StatusAccepted {
		t.Fatalf("other email: want 202, got %d", respOther.StatusCode)
	}
}

func TestOTP_EmailNormalization_CountsAgainstSameLimit(t *testing.T) {
	app := setupAppWithRateLimit(t, handler.RateLimitConfig{})

	variants := []string{
		"target@example.com",
		"Target@example.com",
		"  target@example.com  ",
	}

	for i, email := range variants {
		resp := postJSONFromIP(t, app, "/auth/otp/request", map[string]string{"email": email}, fmt.Sprintf("10.0.1.%d", i+1))
		if resp.StatusCode != http.StatusAccepted {
			t.Fatalf("variant %d (%q): want 202, got %d", i+1, email, resp.StatusCode)
		}
	}

	// 4th variant with uppercase & whitespace should be blocked
	resp := postJSONFromIP(t, app, "/auth/otp/request", map[string]string{"email": " TARGET@EXAMPLE.COM "}, "10.0.1.99")
	checkRateLimitedResponse(t, resp)
}

func TestOTP_PerIPLimit_AuthAndSetup(t *testing.T) {
	app := setupAppWithRateLimit(t, handler.RateLimitConfig{})
	ip := "10.20.30.40"

	// 10 requests allowed from same IP with different emails
	for i := 1; i <= 10; i++ {
		email := fmt.Sprintf("user%d@example.com", i)
		resp := postJSONFromIP(t, app, "/auth/otp/request", map[string]string{"email": email}, ip)
		if resp.StatusCode != http.StatusAccepted {
			t.Fatalf("request %d: want 202, got %d", i, resp.StatusCode)
		}
	}

	// 11th request from same IP should be blocked
	resp := postJSONFromIP(t, app, "/auth/otp/request", map[string]string{"email": "user11@example.com"}, ip)
	checkRateLimitedResponse(t, resp)

	// Same email from a different IP should succeed
	respDiffIP := postJSONFromIP(t, app, "/auth/otp/request", map[string]string{"email": "user11@example.com"}, "10.20.30.41")
	if respDiffIP.StatusCode != http.StatusAccepted {
		t.Fatalf("diff IP: want 202, got %d", respDiffIP.StatusCode)
	}
}

func TestOTP_SetupOTPRateLimit(t *testing.T) {
	app := setupAppWithRateLimit(t, handler.RateLimitConfig{})

	// 3 requests to /setup/otp for the same email
	for i := 1; i <= 3; i++ {
		resp := postJSONFromIP(t, app, "/setup/otp", map[string]string{"email": "setup@example.com"}, fmt.Sprintf("172.16.0.%d", i))
		if resp.StatusCode != http.StatusAccepted {
			t.Fatalf("setup request %d: want 202, got %d", i, resp.StatusCode)
		}
	}

	// 4th request exceeds limit
	resp := postJSONFromIP(t, app, "/setup/otp", map[string]string{"email": "setup@example.com"}, "172.16.0.99")
	checkRateLimitedResponse(t, resp)
}

func TestOTP_WindowRecovery(t *testing.T) {
	app := setupAppWithRateLimit(t, handler.RateLimitConfig{
		MaxEmailRequests: 2,
		Window:           1 * time.Second,
	})

	email := "recover@example.com"

	// 2 requests hit max
	for i := 1; i <= 2; i++ {
		resp := postJSONFromIP(t, app, "/auth/otp/request", map[string]string{"email": email}, fmt.Sprintf("10.0.2.%d", i))
		if resp.StatusCode != http.StatusAccepted {
			t.Fatalf("request %d: want 202, got %d", i, resp.StatusCode)
		}
	}

	// 3rd request blocked
	resp := postJSONFromIP(t, app, "/auth/otp/request", map[string]string{"email": email}, "10.0.2.99")
	checkRateLimitedResponse(t, resp)

	// Wait for window to expire (>1s)
	time.Sleep(1100 * time.Millisecond)

	// Request should now succeed
	respAfter := postJSONFromIP(t, app, "/auth/otp/request", map[string]string{"email": email}, "10.0.2.99")
	if respAfter.StatusCode != http.StatusAccepted {
		t.Fatalf("after window recovery: want 202, got %d", respAfter.StatusCode)
	}
}
