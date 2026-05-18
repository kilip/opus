package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/kilip/opus/server/internal/auth"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// GoogleProvider handles Google OpenID profile mapping.
type GoogleProvider struct {
	config *oauth2.Config
}

// NewGoogleProvider registers authentication properties.
func NewGoogleProvider(creds auth.ProviderCredentials) *GoogleProvider {
	return &GoogleProvider{
		config: &oauth2.Config{
			ClientID:     creds.ClientID,
			ClientSecret: creds.ClientSecret,
			RedirectURL:  creds.RedirectURL,
			Endpoint:     google.Endpoint,
			Scopes: []string{
				"https://www.googleapis.com/auth/userinfo.email",
				"https://www.googleapis.com/auth/userinfo.profile",
			},
		},
	}
}

// Name returns the identifier string.
func (p *GoogleProvider) Name() string { return "google" }

// AuthURL generates secure redirect paths.
func (p *GoogleProvider) AuthURL(state string) string { return p.config.AuthCodeURL(state) }

// Exchange contacts Google API to resolve tokens and parse profiles.
func (p *GoogleProvider) Exchange(ctx context.Context, code string) (*auth.OAuthProfile, error) {
	token, err := p.config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("google.Exchange: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return nil, fmt.Errorf("google.Exchange: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("google.Exchange: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google.Exchange: non-200 status %d", resp.StatusCode)
	}

	var raw struct {
		ID       string `json:"id"`
		Email    string `json:"email"`
		Name     string `json:"name"`
		Picture  string `json:"picture"`
		Verified bool   `json:"verified_email"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("google.Exchange: %w", err)
	}

	return &auth.OAuthProfile{
		ProviderID: raw.ID,
		Provider:   "google",
		Email:      raw.Email,
		Name:       raw.Name,
		AvatarURL:  raw.Picture,
	}, nil
}
