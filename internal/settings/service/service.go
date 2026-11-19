package service

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/kilip/opus/internal/settings/domain"
	"github.com/kilip/opus/internal/settings/repo"
)

const (
	gitHubTokenKey           = "github_token"
	defaultGitHubAPIBaseURL  = "https://api.github.com"
	defaultValidationTimeout = 10 * time.Second
)

// Config configures the Settings service.
type Config struct {
	Secret           string
	GitHubAPIBaseURL string
	HTTPClient       *http.Client
	Getenv           func(string) string
}

// Settings manages encrypted secret settings.
type Settings struct {
	repo          *repo.Ent
	key           []byte
	githubBaseURL string
	httpClient    *http.Client
	getenv        func(string) string
}

// New creates a Settings service using an HKDF key derived from cfg.Secret.
func New(r *repo.Ent, cfg Config) (*Settings, error) {
	key, err := DeriveKey(cfg.Secret)
	if err != nil {
		return nil, err
	}
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.GitHubAPIBaseURL), "/")
	if baseURL == "" {
		baseURL = defaultGitHubAPIBaseURL
	}
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultValidationTimeout}
	}
	getenv := cfg.Getenv
	if getenv == nil {
		getenv = os.Getenv
	}
	return &Settings{
		repo:          r,
		key:           key,
		githubBaseURL: baseURL,
		httpClient:    httpClient,
		getenv:        getenv,
	}, nil
}

// GetGitHubTokenStatus returns metadata about the stored GitHub token without exposing it.
func (s *Settings) GetGitHubTokenStatus(ctx context.Context) (domain.GitHubTokenStatusResponse, error) {
	envOverride := s.getenv("GH_TOKEN") != ""
	rec, err := s.repo.GetByKey(ctx, gitHubTokenKey)
	if err != nil {
		return domain.GitHubTokenStatusResponse{}, err
	}
	if rec == nil {
		return domain.GitHubTokenStatusResponse{
			Configured:  false,
			EnvOverride: envOverride,
		}, nil
	}

	if _, err := decryptSecret(s.key, rec.Ciphertext, rec.Nonce); err != nil {
		// AUTH_SECRET was rotated or ciphertext cannot be decrypted; surface needsReentry cleanly.
		return domain.GitHubTokenStatusResponse{
			Configured:   false,
			Last4:        rec.Last4,
			UpdatedAt:    &rec.UpdatedAt,
			EnvOverride:  envOverride,
			NeedsReentry: true,
		}, nil
	}

	return domain.GitHubTokenStatusResponse{
		Configured:   true,
		Last4:        rec.Last4,
		UpdatedAt:    &rec.UpdatedAt,
		EnvOverride:  envOverride,
		NeedsReentry: false,
	}, nil
}

// SetGitHubToken validates token against GitHub, encrypts it, and persists it.
func (s *Settings) SetGitHubToken(ctx context.Context, rawToken string) (domain.GitHubTokenStatusResponse, error) {
	token := strings.TrimSpace(rawToken)
	if token == "" {
		return domain.GitHubTokenStatusResponse{}, domain.ErrEmptyGitHubToken
	}

	if err := s.validateGitHubToken(ctx, token); err != nil {
		return domain.GitHubTokenStatusResponse{}, err
	}

	ciphertext, nonce, err := encryptSecret(s.key, token)
	if err != nil {
		return domain.GitHubTokenStatusResponse{}, err
	}

	last4 := token
	if len(token) > 4 {
		last4 = token[len(token)-4:]
	}

	if _, err := s.repo.Upsert(ctx, gitHubTokenKey, ciphertext, nonce, currentKeyVer, last4); err != nil {
		return domain.GitHubTokenStatusResponse{}, err
	}

	return s.GetGitHubTokenStatus(ctx)
}

// DeleteGitHubToken removes the stored GitHub token.
func (s *Settings) DeleteGitHubToken(ctx context.Context) error {
	return s.repo.DeleteByKey(ctx, gitHubTokenKey)
}

// EffectiveGitHubToken returns GH_TOKEN if set, otherwise the decrypted DB token,
// or "" if unconfigured or unreadable (e.g. after AUTH_SECRET rotation).
func (s *Settings) EffectiveGitHubToken(ctx context.Context) string {
	if tok := s.getenv("GH_TOKEN"); tok != "" {
		return tok
	}
	return s.StoredGitHubToken(ctx)
}

// StoredGitHubToken returns the decrypted DB token, or "" if unconfigured or unreadable.
func (s *Settings) StoredGitHubToken(ctx context.Context) string {
	rec, err := s.repo.GetByKey(ctx, gitHubTokenKey)
	if err != nil || rec == nil {
		return ""
	}
	tok, err := decryptSecret(s.key, rec.Ciphertext, rec.Nonce)
	if err != nil {
		return ""
	}
	return tok
}

func (s *Settings) validateGitHubToken(ctx context.Context, token string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.githubBaseURL+"/user", nil)
	if err != nil {
		return fmt.Errorf("build GitHub validation request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "opus")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("validate GitHub token: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: GitHub API returned status %d", domain.ErrInvalidGitHubToken, resp.StatusCode)
	}
	return nil
}
