// Package service holds auth business logic: OTP issue/verify, session
// lifecycle and owner setup. It depends on domain, repo and platform/mailer
// only; HTTP concerns (cookies, status codes) stay in handler.
package service

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kilip/opus/internal/auth/domain"
	"github.com/kilip/opus/internal/auth/provider"
	"github.com/kilip/opus/internal/auth/repo"
	"github.com/kilip/opus/internal/platform/mailer"
)

const (
	otpTTL         = 10 * time.Minute
	defaultSession = 24 * time.Hour
)

// Config holds service parameters.
type Config struct {
	// Secret keys the HMAC used to hash OTP codes and session tokens.
	Secret string
	// SessionTTL is session lifetime; defaults to 24h when <= 0.
	SessionTTL time.Duration
	// Now is the clock; defaults to time.Now. Injectable for tests.
	Now func() time.Time
	// Registry holds configured OAuth2 identity providers.
	Registry *provider.Registry
}

// Auth owns OTP, session and setup business rules.
type Auth struct {
	sessions repo.Repo
	otps     repo.OTPRepo
	mailer   mailer.Mailer
	registry *provider.Registry
	secret   string
	ttl      time.Duration
	now      func() time.Time
}

// New returns an Auth service.
func New(sessions repo.Repo, otps repo.OTPRepo, m mailer.Mailer, cfg Config) *Auth {
	ttl := cfg.SessionTTL
	if ttl <= 0 {
		ttl = defaultSession
	}
	now := cfg.Now
	if now == nil {
		now = time.Now
	}
	return &Auth{
		sessions: sessions,
		otps:     otps,
		mailer:   m,
		registry: cfg.Registry,
		secret:   cfg.Secret,
		ttl:      ttl,
		now:      now,
	}
}

// Session resolves a raw session token to a live session.
// Returns domain.ErrUnauthorized on any failure.
func (s *Auth) Session(ctx context.Context, raw string) (*domain.Session, error) {
	if raw == "" {
		return nil, domain.ErrUnauthorized
	}
	sess, err := s.sessions.FindByTokenHash(ctx, s.hash(raw))
	if err != nil || sess == nil || s.now().After(sess.ExpiresAt) {
		return nil, domain.ErrUnauthorized
	}
	return sess, nil
}

// Logout deletes the session for raw token; noop when raw is empty or unknown.
func (s *Auth) Logout(ctx context.Context, raw string, meta domain.AuditMeta) {
	if raw == "" {
		s.recordAudit(ctx, nil, domain.AuditEventLogout, meta)
		return
	}
	var uid *uuid.UUID
	sess, err := s.sessions.FindByTokenHash(ctx, s.hash(raw))
	if err == nil && sess != nil {
		uid = &sess.UserID
	}
	_ = s.sessions.DeleteByTokenHash(ctx, s.hash(raw))
	s.recordAudit(ctx, uid, domain.AuditEventLogout, meta)
}

// NeedsSetup reports whether no owner exists yet.
func (s *Auth) NeedsSetup(ctx context.Context) (bool, error) {
	n, err := s.sessions.CountUsers(ctx)
	if err != nil {
		return false, err
	}
	return n == 0, nil
}

// RequestOTP mails a login code when email belongs to an existing user.
// Unknown emails are silently ignored so callers cannot enumerate users.
func (s *Auth) RequestOTP(ctx context.Context, email string) error {
	user, err := s.otps.FindUserByEmail(ctx, email)
	if err != nil || user == nil {
		return nil
	}
	return s.issueOTP(ctx, email, "Your OTP code")
}

// VerifyOTP checks code for email, consumes the OTP and opens a session.
// Returns the user and the raw session token for the caller to deliver.
func (s *Auth) VerifyOTP(ctx context.Context, email, code string, meta domain.AuditMeta) (*domain.User, string, error) {
	otp, err := s.otps.FindByEmail(ctx, email)
	if err != nil || otp == nil {
		s.recordAudit(ctx, nil, domain.AuditEventOTPFailed, meta)
		return nil, "", domain.ErrInvalidOTP
	}

	// lockout before checking code
	if otp.Attempts >= domain.MaxOTPAttempts {
		_ = s.otps.DeleteByEmail(ctx, email)
		s.recordAudit(ctx, nil, domain.AuditEventOTPFailed, meta)
		return nil, "", domain.ErrTooManyAttempts
	}
	if s.now().After(otp.ExpiresAt) {
		s.recordAudit(ctx, nil, domain.AuditEventOTPFailed, meta)
		return nil, "", domain.ErrOTPExpired
	}
	if !s.codeMatches(otp.CodeHash, code) {
		attempts, _ := s.otps.IncrementAttempts(ctx, otp.ID)
		if attempts >= domain.MaxOTPAttempts {
			_ = s.otps.DeleteByEmail(ctx, email)
		}
		s.recordAudit(ctx, nil, domain.AuditEventOTPFailed, meta)
		if attempts >= domain.MaxOTPAttempts {
			return nil, "", domain.ErrTooManyAttempts
		}
		return nil, "", domain.ErrInvalidOTP
	}

	// single-use: delete OTP before creating session
	_ = s.otps.DeleteByEmail(ctx, email)

	user, err := s.otps.FindUserByEmail(ctx, email)
	if err != nil || user == nil {
		s.recordAudit(ctx, nil, domain.AuditEventOTPFailed, meta)
		return nil, "", domain.ErrInvalidOTP
	}

	raw, hash, err := s.newToken()
	if err != nil {
		return nil, "", err
	}
	if _, err = s.sessions.CreateSession(ctx, user.ID, hash, s.now().Add(s.ttl)); err != nil {
		return nil, "", err
	}
	s.recordAudit(ctx, &user.ID, domain.AuditEventLoginOTP, meta)
	return user, raw, nil
}

// RequestSetupOTP mails a setup code while no owner exists.
func (s *Auth) RequestSetupOTP(ctx context.Context, email string) error {
	if err := s.requireOpenSetup(ctx); err != nil {
		return err
	}
	return s.issueOTP(ctx, email, "Your setup OTP code")
}

// CompleteSetup verifies the setup code, creates the owner and opens a session.
// Returns the owner and the raw session token for the caller to deliver.
//
// ponytail: unlike VerifyOTP there is no attempt counter here; add lockout
// before exposing setup to the internet.
func (s *Auth) CompleteSetup(ctx context.Context, email, code string, meta domain.AuditMeta) (*domain.User, string, error) {
	if err := s.requireOpenSetup(ctx); err != nil {
		return nil, "", err
	}

	otp, err := s.otps.FindByEmail(ctx, email)
	if err != nil || otp == nil {
		s.recordAudit(ctx, nil, domain.AuditEventOTPFailed, meta)
		return nil, "", domain.ErrInvalidOTP
	}
	if s.now().After(otp.ExpiresAt) {
		s.recordAudit(ctx, nil, domain.AuditEventOTPFailed, meta)
		return nil, "", domain.ErrOTPExpired
	}
	if !s.codeMatches(otp.CodeHash, code) {
		s.recordAudit(ctx, nil, domain.AuditEventOTPFailed, meta)
		return nil, "", domain.ErrInvalidOTP
	}
	_ = s.otps.DeleteByEmail(ctx, email)

	raw, hash, err := s.newToken()
	if err != nil {
		return nil, "", err
	}
	sess, err := s.sessions.CreateUserAndSession(ctx, email, hash, s.now().Add(s.ttl))
	if err != nil {
		// concurrent call hit the unique constraint: treat as setup closed
		if n, _ := s.sessions.CountUsers(ctx); n > 0 {
			return nil, "", domain.ErrSetupClosed
		}
		return nil, "", err
	}
	s.recordAudit(ctx, &sess.User.ID, domain.AuditEventSetupCompleted, meta)
	return sess.User, raw, nil
}

func (s *Auth) requireOpenSetup(ctx context.Context) error {
	needs, err := s.NeedsSetup(ctx)
	if err != nil {
		return err
	}
	if !needs {
		return domain.ErrSetupClosed
	}
	return nil
}

// issueOTP stores a fresh code for email and mails it. Mail failure is
// ignored on purpose: the response must not reveal delivery state.
func (s *Auth) issueOTP(ctx context.Context, email, subject string) error {
	code, err := generateOTPCode()
	if err != nil {
		return err
	}
	if _, err := s.otps.CreateOTP(ctx, email, s.hash(code), s.now().Add(otpTTL)); err != nil {
		return err
	}
	_ = s.mailer.Send(ctx, email, subject,
		fmt.Sprintf("Your one-time code is: %s (valid for %d minutes)", code, int(otpTTL.Minutes())))
	return nil
}

// newToken returns a random session token and its stored hash.
func (s *Auth) newToken() (raw, hash string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", err
	}
	raw = base64.RawURLEncoding.EncodeToString(b)
	return raw, s.hash(raw), nil
}

// codeMatches compares the stored code hash with the hash of the given code
// in constant time.
func (s *Auth) codeMatches(storedHash, code string) bool {
	return subtle.ConstantTimeCompare([]byte(storedHash), []byte(s.hash(code))) == 1
}

// hash returns the HMAC-SHA256 base64url hash of raw keyed with the secret.
func (s *Auth) hash(raw string) string {
	mac := hmac.New(sha256.New, []byte(s.secret))
	mac.Write([]byte(raw))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// generateOTPCode returns a random 6-digit string from crypto/rand.
func generateOTPCode() (string, error) {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", binary.BigEndian.Uint32(b)%1_000_000), nil
}

// EnabledProviders returns the list of configured and enabled OAuth identity providers.
func (s *Auth) EnabledProviders() []domain.ProviderItem {
	if s.registry == nil {
		return []domain.ProviderItem{}
	}
	names := s.registry.Enabled()
	items := make([]domain.ProviderItem, 0, len(names))
	for _, name := range names {
		p, ok := s.registry.Get(name)
		if !ok {
			continue
		}
		items = append(items, domain.ProviderItem{
			Name:  p.Name(),
			Label: p.Label(),
		})
	}
	return items
}

// HasProvider reports whether providerName is registered and enabled.
func (s *Auth) HasProvider(providerName string) bool {
	if s.registry == nil {
		return false
	}
	_, ok := s.registry.Get(strings.ToLower(providerName))
	return ok
}

// OAuthAuthorize generates an authorization URL, state, and PKCE verifier for providerName.
func (s *Auth) OAuthAuthorize(providerName string) (authURL, state, verifier string, err error) {
	if s.registry == nil {
		return "", "", "", domain.ErrProviderNotFound
	}
	p, ok := s.registry.Get(strings.ToLower(providerName))
	if !ok {
		return "", "", "", domain.ErrProviderNotFound
	}

	state, err = randomString(32)
	if err != nil {
		return "", "", "", err
	}
	verifier, err = randomString(32)
	if err != nil {
		return "", "", "", err
	}

	authURL = p.AuthCodeURL(state, verifier)
	return authURL, state, verifier, nil
}

const oauthCookieSeparator = "."

// SignOAuthCookie signs the state, verifier, and optional linkUserID using HMAC-SHA256 keyed with s.secret.
func (s *Auth) SignOAuthCookie(state, verifier string, linkUserID ...string) string {
	raw := state + ":" + verifier
	if len(linkUserID) > 0 && linkUserID[0] != "" {
		raw += ":" + linkUserID[0]
	}
	payload := base64.RawURLEncoding.EncodeToString([]byte(raw))
	mac := hmac.New(sha256.New, []byte(s.secret))
	mac.Write([]byte(payload))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return payload + oauthCookieSeparator + sig
}

// VerifyOAuthCookie checks the HMAC signature of val and extracts state, verifier, and optional linkUserID.
func (s *Auth) VerifyOAuthCookie(val string) (state, verifier, linkUserID string, ok bool) {
	parts := strings.Split(val, oauthCookieSeparator)
	if len(parts) != 2 {
		return "", "", "", false
	}
	payload, sig := parts[0], parts[1]
	mac := hmac.New(sha256.New, []byte(s.secret))
	mac.Write([]byte(payload))
	expectedSig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if subtle.ConstantTimeCompare([]byte(sig), []byte(expectedSig)) != 1 {
		return "", "", "", false
	}
	data, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return "", "", "", false
	}
	pieces := strings.SplitN(string(data), ":", 3)
	if len(pieces) < 2 {
		return "", "", "", false
	}
	state = pieces[0]
	verifier = pieces[1]
	if len(pieces) == 3 {
		linkUserID = pieces[2]
	}
	return state, verifier, linkUserID, true
}

// OAuthCallback validates the OAuth callback parameters, exchanges the code with the provider,
// and enforces linking policy. Returns a new raw session token on success.
func (s *Auth) OAuthCallback(ctx context.Context, providerName, code, cookieState, queryState, verifier string, meta domain.AuditMeta) (string, error) {
	if s.registry == nil {
		return "", domain.ErrProviderNotFound
	}
	p, ok := s.registry.Get(strings.ToLower(providerName))
	if !ok {
		return "", domain.ErrProviderNotFound
	}

	if cookieState == "" || queryState == "" || subtle.ConstantTimeCompare([]byte(cookieState), []byte(queryState)) != 1 {
		return "", domain.ErrStateMismatch
	}

	profile, err := p.Exchange(ctx, code, verifier)
	if err != nil {
		return "", err
	}

	if profile.ProviderUserID == "" {
		return "", domain.ErrAccountNotAllowed
	}

	// Policy (a): existing account link -> log in as its user
	acc, err := s.sessions.FindAccount(ctx, p.Name(), profile.ProviderUserID)
	if err != nil {
		return "", err
	}
	if acc != nil {
		raw, hash, err := s.newToken()
		if err != nil {
			return "", err
		}
		if _, err = s.sessions.CreateSession(ctx, acc.UserID, hash, s.now().Add(s.ttl)); err != nil {
			return "", err
		}
		s.recordAudit(ctx, &acc.UserID, domain.AuditEventLoginOAuth(p.Name()), meta)
		return raw, nil
	}

	// Policy (b) & (c): linking requires an existing owner
	owner, err := s.sessions.GetOwner(ctx)
	if err != nil {
		return "", err
	}
	if owner == nil {
		return "", domain.ErrSetupRequired
	}

	if profile.EmailVerified && profile.Email != "" && strings.EqualFold(profile.Email, owner.Email) {
		if _, err := s.sessions.CreateAccount(ctx, owner.ID, p.Name(), profile.ProviderUserID, profile.Email); err != nil {
			return "", err
		}
		s.recordAudit(ctx, &owner.ID, domain.AuditEventAccountLinked, meta)
		raw, hash, err := s.newToken()
		if err != nil {
			return "", err
		}
		if _, err = s.sessions.CreateSession(ctx, owner.ID, hash, s.now().Add(s.ttl)); err != nil {
			return "", err
		}
		s.recordAudit(ctx, &owner.ID, domain.AuditEventLoginOAuth(p.Name()), meta)
		return raw, nil
	}

	return "", domain.ErrAccountNotAllowed
}

// OAuthLinkCallback handles the OAuth provider callback specifically for linking an external account
// to an existing authenticated user.
func (s *Auth) OAuthLinkCallback(ctx context.Context, userID uuid.UUID, providerName, code, cookieState, queryState, verifier string, meta domain.AuditMeta) error {
	if s.registry == nil {
		return domain.ErrProviderNotFound
	}
	p, ok := s.registry.Get(strings.ToLower(providerName))
	if !ok {
		return domain.ErrProviderNotFound
	}

	if cookieState == "" || queryState == "" || subtle.ConstantTimeCompare([]byte(cookieState), []byte(queryState)) != 1 {
		return domain.ErrStateMismatch
	}

	profile, err := p.Exchange(ctx, code, verifier)
	if err != nil {
		return err
	}

	if profile.ProviderUserID == "" {
		return domain.ErrAccountNotAllowed
	}

	acc, err := s.sessions.FindAccount(ctx, p.Name(), profile.ProviderUserID)
	if err != nil {
		return err
	}
	if acc != nil {
		if acc.UserID != userID {
			return domain.ErrAccountAlreadyLinked
		}
		// Idempotent: already linked to the current user
		return nil
	}

	_, err = s.sessions.CreateAccount(ctx, userID, p.Name(), profile.ProviderUserID, profile.Email)
	if err != nil {
		return err
	}
	s.recordAudit(ctx, &userID, domain.AuditEventAccountLinked, meta)
	return nil
}

// ListAccounts returns all external OAuth accounts linked to userID.
func (s *Auth) ListAccounts(ctx context.Context, userID uuid.UUID) ([]*domain.Account, error) {
	return s.sessions.ListAccountsByUserID(ctx, userID)
}

// UnlinkAccount removes the linked account with accountID belonging to userID.
// Returns domain.ErrAccountNotFound if the account does not exist or belongs to another user.
func (s *Auth) UnlinkAccount(ctx context.Context, userID, accountID uuid.UUID, meta domain.AuditMeta) error {
	if err := s.sessions.DeleteAccount(ctx, userID, accountID); err != nil {
		return err
	}
	s.recordAudit(ctx, &userID, domain.AuditEventAccountUnlinked, meta)
	return nil
}

func (s *Auth) recordAudit(ctx context.Context, userID *uuid.UUID, event string, meta domain.AuditMeta) {
	if err := s.sessions.RecordAuditLog(ctx, userID, event, meta); err != nil {
		slog.Warn("failed to write audit log", "event", event, "error", err)
	}
}

func randomString(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
