package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"golang.org/x/oauth2"
)

// DefaultFacebookGraphBaseURL is the default base URL for the Facebook Graph API.
const DefaultFacebookGraphBaseURL = "https://graph.facebook.com"

// FacebookOption configures a FacebookProvider.
type FacebookOption func(*FacebookProvider)

// WithGraphBaseURL overrides the Facebook Graph API base URL.
// Defaults to "https://graph.facebook.com".
func WithGraphBaseURL(rawURL string) FacebookOption {
	return func(p *FacebookProvider) {
		p.graphBaseURL = strings.TrimRight(rawURL, "/")
	}
}

// WithFacebookHTTPClient overrides the HTTP client used for OAuth token exchange and API requests.
func WithFacebookHTTPClient(client *http.Client) FacebookOption {
	return func(p *FacebookProvider) {
		p.httpClient = client
	}
}

// FacebookProvider implements the Provider interface for Facebook OAuth2 authentication.
type FacebookProvider struct {
	*BaseProvider
	graphBaseURL string
	httpClient   *http.Client
}

var _ Provider = (*FacebookProvider)(nil)

// NewFacebookProvider creates a new Facebook OAuth2 provider.
func NewFacebookProvider(cfg oauth2.Config, opts ...FacebookOption) *FacebookProvider {
	p := &FacebookProvider{
		BaseProvider: NewBaseProvider("facebook", cfg, "Facebook"),
		graphBaseURL: DefaultFacebookGraphBaseURL,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

type facebookUserResponse struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// Exchange exchanges authorization code and PKCE verifier for a user profile via Facebook Graph API.
func (p *FacebookProvider) Exchange(ctx context.Context, code, verifier string) (Profile, error) {
	if p.httpClient != nil && ctx.Value(oauth2.HTTPClient) == nil {
		ctx = context.WithValue(ctx, oauth2.HTTPClient, p.httpClient)
	}

	var opts []oauth2.AuthCodeOption
	if verifier != "" {
		opts = append(opts, oauth2.VerifierOption(verifier))
	}

	token, err := p.cfg.Exchange(ctx, code, opts...)
	if err != nil {
		return Profile{}, fmt.Errorf("facebook oauth exchange: %w", err)
	}

	client := p.cfg.Client(ctx, token)

	user, err := p.fetchUser(ctx, client)
	if err != nil {
		return Profile{}, err
	}

	// Facebook does not expose an email verification flag, so its email
	// must never be trusted for auto-linking (EmailVerified is always false).
	return Profile{
		ProviderUserID: user.ID,
		Email:          user.Email,
		EmailVerified:  false,
		Name:           user.Name,
	}, nil
}

func (p *FacebookProvider) fetchUser(ctx context.Context, client *http.Client) (facebookUserResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.graphBaseURL+"/me?fields=id,name,email", nil)
	if err != nil {
		return facebookUserResponse{}, fmt.Errorf("facebook user request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return facebookUserResponse{}, fmt.Errorf("fetch facebook user: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return facebookUserResponse{}, fmt.Errorf("facebook graph api returned status %d", resp.StatusCode)
	}

	var user facebookUserResponse
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return facebookUserResponse{}, fmt.Errorf("decode facebook user response: %w", err)
	}
	if strings.TrimSpace(user.ID) == "" {
		return facebookUserResponse{}, errors.New("invalid or missing facebook user id")
	}
	return user, nil
}
