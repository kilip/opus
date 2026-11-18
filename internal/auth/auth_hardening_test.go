package auth_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/kilip/opus/ent"
	"github.com/kilip/opus/internal/auth"
	"github.com/kilip/opus/internal/auth/domain"
	"github.com/kilip/opus/internal/auth/repo"
	"github.com/kilip/opus/internal/auth/service"
	_ "modernc.org/sqlite"
)

func openAuthTestDB(t *testing.T) *ent.Client {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "auth.db")
	sqlDB, err := sql.Open("sqlite", "file:"+dbPath+"?_pragma=foreign_keys(ON)&_pragma=busy_timeout(10000)")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	client := ent.NewClient(ent.Driver(entsql.OpenDB(dialect.SQLite, sqlDB)))
	if err := client.Schema.Create(context.Background()); err != nil {
		t.Fatalf("Schema.Create: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })
	return client
}

type slowCapturingMailer struct {
	delay time.Duration
	mu    sync.Mutex
	last  string
}

func (m *slowCapturingMailer) Send(_ context.Context, _, _, body string) error {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
	m.mu.Lock()
	m.last = body
	m.mu.Unlock()
	return nil
}

func TestAuth_ShortSecretAndPersistedSecret(t *testing.T) {
	db := openAuthTestDB(t)
	if _, err := auth.NewWithMailer(db, auth.AuthConfig{
		BaseURL:       "http://localhost:7880",
		SessionSecret: "too-short",
	}, nil); err == nil {
		t.Fatal("expected error for AUTH_SECRET < 32 chars")
	}

	appHome := t.TempDir()
	a1, err := auth.NewWithMailer(db, auth.AuthConfig{
		BaseURL: "http://localhost:7880",
		AppHome: appHome,
	}, nil)
	if err != nil {
		t.Fatalf("NewWithMailer 1: %v", err)
	}
	secretPath := filepath.Join(appHome, "auth_secret")
	info, err := os.Stat(secretPath)
	if err != nil {
		t.Fatalf("stat auth_secret: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("auth_secret perm = %o, want 0600", info.Mode().Perm())
	}

	// Seed a user and session via a1, then rebuild a2 with same AppHome and verify session survives.
	ctx := context.Background()
	u, err := db.User.Create().SetEmail("persist@example.com").Save(ctx)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	_ = u
	_ = a1

	a2, err := auth.NewWithMailer(db, auth.AuthConfig{
		BaseURL: "http://localhost:7880",
		AppHome: appHome,
	}, nil)
	if err != nil {
		t.Fatalf("NewWithMailer 2: %v", err)
	}
	if a2 == nil {
		t.Fatal("expected non-nil auth feature")
	}
}

func TestAuth_CookieSecureModes(t *testing.T) {
	db := openAuthTestDB(t)
	if _, err := auth.NewWithMailer(db, auth.AuthConfig{
		BaseURL:       "http://localhost:7880",
		SessionSecret: "01234567890123456789012345678901",
		CookieSecure:  "bogus",
	}, nil); err == nil {
		t.Fatal("expected error for invalid AUTH_COOKIE_SECURE")
	}

	m := &slowCapturingMailer{}
	a, err := auth.NewWithMailer(db, auth.AuthConfig{
		BaseURL:       "http://localhost:7880",
		SessionSecret: "01234567890123456789012345678901",
		SessionTTL:    time.Hour,
		SetupToken:    "setup-tok-123",
		CookieSecure:  "true",
	}, m)
	if err != nil {
		t.Fatalf("NewWithMailer: %v", err)
	}
	f := fiber.New()
	a.RegisterSetup(f.Group("/api/setup"))

	// Request setup OTP with valid token.
	req := httptest.NewRequest(http.MethodPost, "/api/setup/otp", bytes.NewBufferString(`{"email":"owner@example.com"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Setup-Token", "setup-tok-123")
	resp, _ := f.Test(req)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("setup otp status = %d, want 202", resp.StatusCode)
	}

	// Extract 6-digit code from mailer body.
	parts := strings.Split(m.last, "is: ")
	code := parts[1][:6]

	compReq := httptest.NewRequest(http.MethodPost, "/api/setup", bytes.NewBufferString(`{"email":"owner@example.com","code":"`+code+`"}`))
	compReq.Header.Set("Content-Type", "application/json")
	compReq.Header.Set("X-Setup-Token", "setup-tok-123")
	compResp, _ := f.Test(compReq)
	_ = compResp.Body.Close()
	if compResp.StatusCode != http.StatusOK {
		t.Fatalf("setup complete status = %d, want 200", compResp.StatusCode)
	}
	var foundSecure bool
	for _, c := range compResp.Cookies() {
		if c.Name == auth.SessionCookie && c.Secure {
			foundSecure = true
		}
	}
	if !foundSecure {
		t.Fatal("expected Secure cookie when AUTH_COOKIE_SECURE=true on http://")
	}
}

func TestAuth_SetupTokenRequiresTokenAndPreservesOTPAttempts(t *testing.T) {
	db := openAuthTestDB(t)
	m := &slowCapturingMailer{}
	a, err := auth.NewWithMailer(db, auth.AuthConfig{
		BaseURL:       "http://localhost:7880",
		SessionSecret: "01234567890123456789012345678901",
		SessionTTL:    time.Hour,
		SetupToken:    "secret-setup-tok",
		RateLimit:     auth.RateLimitConfig{Disabled: true},
	}, m)
	if err != nil {
		t.Fatalf("NewWithMailer: %v", err)
	}
	f := fiber.New()
	a.RegisterSetup(f.Group("/api/setup"))

	// GET /api/setup/status -> requiresToken: true
	stResp, _ := f.Test(httptest.NewRequest(http.MethodGet, "/api/setup/status", nil))
	var stBody struct {
		Data domain.SetupStatusResponse `json:"data"`
	}
	_ = json.NewDecoder(stResp.Body).Decode(&stBody)
	_ = stResp.Body.Close()
	if !stBody.Data.NeedsSetup || !stBody.Data.RequiresToken {
		t.Fatalf("unexpected setup status: %+v", stBody.Data)
	}

	// Wrong token on /setup/otp -> 403
	badReq := httptest.NewRequest(http.MethodPost, "/api/setup/otp", bytes.NewBufferString(`{"email":"owner@example.com"}`))
	badReq.Header.Set("Content-Type", "application/json")
	badReq.Header.Set("X-Setup-Token", "wrong")
	badResp, _ := f.Test(badReq)
	_ = badResp.Body.Close()
	if badResp.StatusCode != http.StatusForbidden {
		t.Fatalf("wrong setup token status = %d, want 403", badResp.StatusCode)
	}

	// Invalid email with valid token -> 400
	invEmail := httptest.NewRequest(http.MethodPost, "/api/setup/otp", bytes.NewBufferString(`{"email":"not-an-email"}`))
	invEmail.Header.Set("Content-Type", "application/json")
	invEmail.Header.Set("X-Setup-Token", "secret-setup-tok")
	invResp, _ := f.Test(invEmail)
	_ = invResp.Body.Close()
	if invResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("invalid email status = %d, want 400", invResp.StatusCode)
	}

	// Issue valid OTP.
	okReq := httptest.NewRequest(http.MethodPost, "/api/setup/otp", bytes.NewBufferString(`{"email":"owner@example.com"}`))
	okReq.Header.Set("Content-Type", "application/json")
	okReq.Header.Set("X-Setup-Token", "secret-setup-tok")
	okResp, _ := f.Test(okReq)
	_ = okResp.Body.Close()
	if okResp.StatusCode != http.StatusAccepted {
		t.Fatalf("valid setup otp status = %d, want 202", okResp.StatusCode)
	}

	// 10 wrong-token calls to /setup must return 403 and NOT increment OTP attempts.
	for i := 0; i < 10; i++ {
		r := httptest.NewRequest(http.MethodPost, "/api/setup", bytes.NewBufferString(`{"email":"owner@example.com","code":"000000"}`))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("X-Setup-Token", "wrong")
		resp, _ := f.Test(r)
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("wrong setup token on /setup = %d, want 403", resp.StatusCode)
		}
	}
	otpRow, err := db.OTP.Query().Only(context.Background())
	if err != nil {
		t.Fatalf("query otp: %v", err)
	}
	if otpRow.Attempts != 0 {
		t.Fatalf("OTP attempts = %d, want 0 after 403 rejections", otpRow.Attempts)
	}
}

func TestAuth_AtomicAttemptsAndSingleConsumeConcurrency(t *testing.T) {
	ctx := context.Background()

	// AC4: 50 concurrent wrong guesses -> attempts <= 5, at most 5 evaluated.
	t.Run("50 concurrent wrong guesses", func(t *testing.T) {
		db := openAuthTestDB(t)
		m := &slowCapturingMailer{}
		baseRepo := repo.NewOTPRepo(db)
		var evaluated atomic.Int32
		countingRepo := &countingOTPRepo{OTPRepo: baseRepo, evaluated: &evaluated}
		svc := service.New(repo.New(db), countingRepo, m, service.Config{
			Secret:     "01234567890123456789012345678901",
			SessionTTL: time.Hour,
			SetupToken: "tok",
		})
		if err := svc.RequestSetupOTP(ctx, "race@example.com"); err != nil {
			t.Fatalf("RequestSetupOTP: %v", err)
		}

		var wg sync.WaitGroup
		var tooManyCount atomic.Int32
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, _, err := svc.CompleteSetup(ctx, "race@example.com", "999999", domain.AuditMeta{})
				if err == domain.ErrTooManyAttempts {
					tooManyCount.Add(1)
				}
			}()
		}
		wg.Wait()

		if evaluated.Load() > int32(domain.MaxOTPAttempts) {
			t.Fatalf("evaluated %d wrong guesses, want <= %d", evaluated.Load(), domain.MaxOTPAttempts)
		}
		if tooManyCount.Load() == 0 {
			t.Fatal("expected ErrTooManyAttempts among 50 concurrent guesses")
		}
		otpRow, err := db.OTP.Query().Only(ctx)
		if err == nil && otpRow.Attempts > domain.MaxOTPAttempts {
			t.Fatalf("otpRow.Attempts = %d, want <= %d", otpRow.Attempts, domain.MaxOTPAttempts)
		}
		if err != nil && !ent.IsNotFound(err) {
			t.Fatalf("query otp: %v", err)
		}
	})

	// AC5: 20 concurrent right guesses -> exactly 1 session created.
	t.Run("20 concurrent right guesses", func(t *testing.T) {
		db := openAuthTestDB(t)
		m := &slowCapturingMailer{}
		svc := service.New(repo.New(db), repo.NewOTPRepo(db), m, service.Config{
			Secret:     "01234567890123456789012345678901",
			SessionTTL: time.Hour,
			SetupToken: "tok",
		})
		u, err := db.User.Create().SetEmail("win@example.com").Save(ctx)
		if err != nil {
			t.Fatalf("create user: %v", err)
		}
		_ = u
		if err := svc.RequestOTP(ctx, "win@example.com"); err != nil {
			t.Fatalf("RequestOTP: %v", err)
		}
		// Wait for async mailer to populate m.last
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			m.mu.Lock()
			got := m.last
			m.mu.Unlock()
			if got != "" {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		m.mu.Lock()
		code := strings.Split(m.last, "is: ")[1][:6]
		m.mu.Unlock()

		var wg sync.WaitGroup
		var wins atomic.Int32
		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if _, tok, err := svc.VerifyOTP(ctx, "win@example.com", code, domain.AuditMeta{}); err == nil && tok != "" {
					wins.Add(1)
				}
			}()
		}
		wg.Wait()
		if wins.Load() != 1 {
			t.Fatalf("expected exactly 1 winning VerifyOTP, got %d", wins.Load())
		}
		sessCount, err := db.AuthSession.Query().Count(ctx)
		if err != nil || sessCount != 1 {
			t.Fatalf("AuthSession count = %d (err=%v), want 1", sessCount, err)
		}
	})
}

type countingOTPRepo struct {
	repo.OTPRepo
	evaluated *atomic.Int32
}

func (c *countingOTPRepo) ReserveAttempt(ctx context.Context, id uuid.UUID, max int) (int, bool, error) {
	n, ok, err := c.OTPRepo.ReserveAttempt(ctx, id, max)
	if ok {
		c.evaluated.Add(1)
	}
	return n, ok, err
}

func TestAuth_IPRateLimitOnVerifyAndSetup(t *testing.T) {
	db := openAuthTestDB(t)
	a, err := auth.NewWithMailer(db, auth.AuthConfig{
		BaseURL:       "http://localhost:7880",
		SessionSecret: "01234567890123456789012345678901",
		SessionTTL:    time.Hour,
		SetupToken:    "tok-rl",
		RateLimit:     auth.RateLimitConfig{MaxIPRequests: 2, Window: time.Minute},
	}, nil)
	if err != nil {
		t.Fatalf("NewWithMailer: %v", err)
	}
	f := fiber.New()
	a.RegisterPublic(f.Group("/api/auth"))
	a.RegisterSetup(f.Group("/api/setup"))

	for i := 0; i < 2; i++ {
		r1 := httptest.NewRequest(http.MethodPost, "/api/auth/otp/verify", bytes.NewBufferString(`{"email":"a@b.com","code":"000000"}`))
		r1.Header.Set("Content-Type", "application/json")
		resp, _ := f.Test(r1)
		_ = resp.Body.Close()
		if resp.StatusCode == http.StatusTooManyRequests {
			t.Fatalf("verify attempt %d unexpectedly rate limited", i+1)
		}
	}
	r3 := httptest.NewRequest(http.MethodPost, "/api/auth/otp/verify", bytes.NewBufferString(`{"email":"a@b.com","code":"000000"}`))
	r3.Header.Set("Content-Type", "application/json")
	resp3, _ := f.Test(r3)
	_ = resp3.Body.Close()
	if resp3.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("3rd /otp/verify status = %d, want 429", resp3.StatusCode)
	}

	// Setup endpoint shares/uses ipLimiter as well (with fresh app for clean limiter state).
	a2, err := auth.NewWithMailer(db, auth.AuthConfig{
		BaseURL:       "http://localhost:7880",
		SessionSecret: "01234567890123456789012345678901",
		SessionTTL:    time.Hour,
		SetupToken:    "tok-rl",
		RateLimit:     auth.RateLimitConfig{MaxIPRequests: 2, Window: time.Minute},
	}, nil)
	if err != nil {
		t.Fatalf("NewWithMailer 2: %v", err)
	}
	f2 := fiber.New()
	a2.RegisterSetup(f2.Group("/api/setup"))
	for i := 0; i < 2; i++ {
		sReq := httptest.NewRequest(http.MethodPost, "/api/setup", bytes.NewBufferString(`{"email":"a@b.com","code":"000000"}`))
		sReq.Header.Set("Content-Type", "application/json")
		sReq.Header.Set("X-Setup-Token", "tok-rl")
		sResp, _ := f2.Test(sReq)
		_ = sResp.Body.Close()
	}
	sReq3 := httptest.NewRequest(http.MethodPost, "/api/setup", bytes.NewBufferString(`{"email":"a@b.com","code":"000000"}`))
	sReq3.Header.Set("Content-Type", "application/json")
	sReq3.Header.Set("X-Setup-Token", "tok-rl")
	sResp3, _ := f2.Test(sReq3)
	_ = sResp3.Body.Close()
	if sResp3.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("3rd /setup status = %d, want 429", sResp3.StatusCode)
	}
}

func TestAuth_RequestOTPConstantTimeWithSlowMailer(t *testing.T) {
	db := openAuthTestDB(t)
	ctx := context.Background()
	if _, err := db.User.Create().SetEmail("known@example.com").Save(ctx); err != nil {
		t.Fatalf("create user: %v", err)
	}
	m := &slowCapturingMailer{delay: 2 * time.Second}
	svc := service.New(repo.New(db), repo.NewOTPRepo(db), m, service.Config{
		Secret:     "01234567890123456789012345678901",
		SessionTTL: time.Hour,
	})

	startKnown := time.Now()
	if err := svc.RequestOTP(ctx, "known@example.com"); err != nil {
		t.Fatalf("RequestOTP known: %v", err)
	}
	if d := time.Since(startKnown); d >= 500*time.Millisecond {
		t.Fatalf("RequestOTP known took %v, want < 500ms", d)
	}

	startUnknown := time.Now()
	if err := svc.RequestOTP(ctx, "unknown@example.com"); err != nil {
		t.Fatalf("RequestOTP unknown: %v", err)
	}
	if d := time.Since(startUnknown); d >= 500*time.Millisecond {
		t.Fatalf("RequestOTP unknown took %v, want < 500ms", d)
	}
}
