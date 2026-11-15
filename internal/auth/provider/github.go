package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"golang.org/x/oauth2"
)

// GitHubOption configures a GitHubProvider.
type GitHubOption func(*GitHubProvider)

// WithAPIBaseURL overrides the GitHub API base URL.
// Defaults to "https://api.github.com".
func WithAPIBaseURL(apiBaseURL string) GitHubOption {
	return func(p *GitHubProvider) {
		p.apiBaseURL = strings.TrimRight(apiBaseURL, "/")
	}
}

// WithHTTPClient overrides the HTTP client used for OAuth token exchange and API requests.
func WithHTTPClient(client *http.Client) GitHubOption {
	return func(p *GitHubProvider) {
		p.httpClient = client
	}
}

// GitHubProvider implements the Provider interface for GitHub OAuth2 authentication.
type GitHubProvider struct {
	*BaseProvider
	apiBaseURL string
	httpClient *http.Client
}

var _ Provider = (*GitHubProvider)(nil)

// NewGitHubProvider creates a new GitHub OAuth2 provider.
func NewGitHubProvider(cfg oauth2.Config, opts ...GitHubOption) *GitHubProvider {
	p := &GitHubProvider{
		BaseProvider: NewBaseProvider("github", cfg, "GitHub"),
		apiBaseURL:   "https://api.github.com",
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

type githubUserResponse struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
}

type githubEmailResponse struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

// Exchange exchanges authorization code and PKCE verifier for a user profile via GitHub API.
func (p *GitHubProvider) Exchange(ctx context.Context, code, verifier string) (Profile, error) {
	if p.httpClient != nil && ctx.Value(oauth2.HTTPClient) == nil {
		ctx = context.WithValue(ctx, oauth2.HTTPClient, p.httpClient)
	}

	var opts []oauth2.AuthCodeOption
	if verifier != "" {
		opts = append(opts, oauth2.VerifierOption(verifier))
	}

	token, err := p.cfg.Exchange(ctx, code, opts...)
	if err != nil {
		return Profile{}, fmt.Errorf("github oauth exchange: %w", err)
	}

	client := p.cfg.Client(ctx, token)

	user, err := p.fetchUser(ctx, client)
	if err != nil {
		return Profile{}, err
	}

	emails, err := p.fetchEmails(ctx, client)
	if err != nil {
		return Profile{}, err
	}

	var (
		primaryEmail string
		isVerified   bool
	)
	for _, e := range emails {
		if e.Primary && e.Verified {
			primaryEmail = e.Email
			isVerified = true
			break
		}
	}

	name := user.Name
	if name == "" {
		name = user.Login
	}

	return Profile{
		ProviderUserID: strconv.FormatInt(user.ID, 10),
		Email:          primaryEmail,
		EmailVerified:  isVerified,
		Name:           name,
		AvatarURL:      user.AvatarURL,
	}, nil
}

func (p *GitHubProvider) fetchUser(ctx context.Context, client *http.Client) (githubUserResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.apiBaseURL+"/user", nil)
	if err != nil {
		return githubUserResponse{}, fmt.Errorf("github user request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := client.Do(req)
	if err != nil {
		return githubUserResponse{}, fmt.Errorf("fetch github user: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return githubUserResponse{}, fmt.Errorf("github api /user returned status %d", resp.StatusCode)
	}

	var user githubUserResponse
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return githubUserResponse{}, fmt.Errorf("decode github user response: %w", err)
	}
	if user.ID <= 0 {
		return githubUserResponse{}, errors.New("invalid or missing github user id")
	}
	return user, nil
}

func (p *GitHubProvider) fetchEmails(ctx context.Context, client *http.Client) ([]githubEmailResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.apiBaseURL+"/user/emails", nil)
	if err != nil {
		return nil, fmt.Errorf("github user emails request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch github user emails: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github api /user/emails returned status %d", resp.StatusCode)
	}

	var emails []githubEmailResponse
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return nil, fmt.Errorf("decode github user emails response: %w", err)
	}
	return emails, nil
}
