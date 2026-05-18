package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/kilip/opus/server/internal/auth"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

// GitHubProvider handles GitHub federated callbacks.
type GitHubProvider struct {
	config *oauth2.Config
}

// NewGitHubProvider initializes GitHub scopes.
func NewGitHubProvider(creds auth.ProviderCredentials) *GitHubProvider {
	return &GitHubProvider{
		config: &oauth2.Config{
			ClientID:     creds.ClientID,
			ClientSecret: creds.ClientSecret,
			RedirectURL:  creds.RedirectURL,
			Endpoint:     github.Endpoint,
			Scopes:       []string{"user:email", "read:user"},
		},
	}
}

// Name returns the identifier string.
func (p *GitHubProvider) Name() string { return "github" }

// AuthURL generates secure redirect paths.
func (p *GitHubProvider) AuthURL(state string) string { return p.config.AuthCodeURL(state) }

// Exchange maps user profile hashes.
func (p *GitHubProvider) Exchange(ctx context.Context, code string) (*auth.OAuthProfile, error) {
	token, err := p.config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("github.Exchange: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user", nil)
	if err != nil {
		return nil, fmt.Errorf("github.Exchange: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("github.Exchange: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github.Exchange: non-200 status %d", resp.StatusCode)
	}

	var raw struct {
		ID        int    `json:"id"`
		Login     string `json:"login"`
		Name      string `json:"name"`
		AvatarURL string `json:"avatar_url"`
		Email     string `json:"email"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("github.Exchange: %w", err)
	}

	if raw.Email == "" {
		emailReq, _ := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user/emails", nil)
		emailReq.Header.Set("Authorization", "Bearer "+token.AccessToken)
		emailResp, err := http.DefaultClient.Do(emailReq)
		if err == nil {
			defer func() { _ = emailResp.Body.Close() }()
			var emails []struct {
				Email    string `json:"email"`
				Primary  bool   `json:"primary"`
				Verified bool   `json:"verified"`
			}
			if err := json.NewDecoder(emailResp.Body).Decode(&emails); err == nil {
				for _, e := range emails {
					if e.Primary {
						raw.Email = e.Email
						break
					}
				}
			}
		}
	}

	return &auth.OAuthProfile{
		ProviderID: strconv.Itoa(raw.ID),
		Provider:   "github",
		Email:      raw.Email,
		Name:       raw.Name,
		AvatarURL:  raw.AvatarURL,
	}, nil
}
