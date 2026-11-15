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

// DefaultGoogleUserInfoURL is the default Google OpenID Connect UserInfo endpoint.
const DefaultGoogleUserInfoURL = "https://openidconnect.googleapis.com/v1/userinfo"

// GoogleOption configures a GoogleProvider.
type GoogleOption func(*GoogleProvider)

// WithUserInfoURL overrides the Google OIDC userinfo endpoint URL.
// Defaults to "https://openidconnect.googleapis.com/v1/userinfo".
func WithUserInfoURL(url string) GoogleOption {
	return func(p *GoogleProvider) {
		p.userInfoURL = url
	}
}

// WithGoogleHTTPClient overrides the HTTP client used for OAuth token exchange and API requests.
func WithGoogleHTTPClient(client *http.Client) GoogleOption {
	return func(p *GoogleProvider) {
		p.httpClient = client
	}
}

// GoogleProvider implements the Provider interface for Google OAuth2 authentication.
type GoogleProvider struct {
	*BaseProvider
	userInfoURL string
	httpClient  *http.Client
}

var _ Provider = (*GoogleProvider)(nil)

// NewGoogleProvider creates a new Google OAuth2 provider.
func NewGoogleProvider(cfg oauth2.Config, opts ...GoogleOption) *GoogleProvider {
	p := &GoogleProvider{
		BaseProvider: NewBaseProvider("google", cfg, "Google"),
		userInfoURL:  DefaultGoogleUserInfoURL,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

type googleUserInfoResponse struct {
	Sub           string `json:"sub"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
}

// Exchange exchanges authorization code and PKCE verifier for a user profile via Google OIDC UserInfo.
func (p *GoogleProvider) Exchange(ctx context.Context, code, verifier string) (Profile, error) {
	if p.httpClient != nil && ctx.Value(oauth2.HTTPClient) == nil {
		ctx = context.WithValue(ctx, oauth2.HTTPClient, p.httpClient)
	}

	var opts []oauth2.AuthCodeOption
	if verifier != "" {
		opts = append(opts, oauth2.VerifierOption(verifier))
	}

	token, err := p.cfg.Exchange(ctx, code, opts...)
	if err != nil {
		return Profile{}, fmt.Errorf("google oauth exchange: %w", err)
	}

	client := p.cfg.Client(ctx, token)

	user, err := p.fetchUserInfo(ctx, client)
	if err != nil {
		return Profile{}, err
	}

	name := user.Name
	if name == "" {
		name = strings.TrimSpace(user.GivenName + " " + user.FamilyName)
	}

	return Profile{
		ProviderUserID: user.Sub,
		Email:          user.Email,
		EmailVerified:  user.EmailVerified,
		Name:           name,
		AvatarURL:      user.Picture,
	}, nil
}

func (p *GoogleProvider) fetchUserInfo(ctx context.Context, client *http.Client) (googleUserInfoResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.userInfoURL, nil)
	if err != nil {
		return googleUserInfoResponse{}, fmt.Errorf("google userinfo request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return googleUserInfoResponse{}, fmt.Errorf("fetch google userinfo: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return googleUserInfoResponse{}, fmt.Errorf("google api userinfo returned status %d", resp.StatusCode)
	}

	var user googleUserInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return googleUserInfoResponse{}, fmt.Errorf("decode google userinfo response: %w", err)
	}
	if strings.TrimSpace(user.Sub) == "" {
		return googleUserInfoResponse{}, errors.New("invalid or missing google user sub")
	}
	return user, nil
}
