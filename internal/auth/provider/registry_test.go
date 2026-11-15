package provider_test

import (
	"context"
	"net/url"
	"reflect"
	"testing"

	"github.com/kilip/opus/internal/auth/provider"
)

func TestRegistry_EnabledProviders(t *testing.T) {
	tests := []struct {
		name     string
		cfg      provider.Config
		expected []string
	}{
		{
			name:     "empty config has no providers",
			cfg:      provider.Config{BaseURL: "http://localhost:7880"},
			expected: []string{},
		},
		{
			name: "only github configured",
			cfg: provider.Config{
				BaseURL:            "http://localhost:7880",
				GitHubClientID:     "gh-id",
				GitHubClientSecret: "gh-secret",
			},
			expected: []string{"github"},
		},
		{
			name: "github partial: id set but secret empty",
			cfg: provider.Config{
				BaseURL:        "http://localhost:7880",
				GitHubClientID: "gh-id",
			},
			expected: []string{},
		},
		{
			name: "github partial: secret set but id empty",
			cfg: provider.Config{
				BaseURL:            "http://localhost:7880",
				GitHubClientSecret: "gh-secret",
			},
			expected: []string{},
		},
		{
			name: "google partial: id set but secret empty",
			cfg: provider.Config{
				BaseURL:        "http://localhost:7880",
				GoogleClientID: "g-id",
			},
			expected: []string{},
		},
		{
			name: "facebook partial: secret set but id empty",
			cfg: provider.Config{
				BaseURL:              "http://localhost:7880",
				FacebookClientSecret: "fb-secret",
			},
			expected: []string{},
		},
		{
			name: "all providers configured",
			cfg: provider.Config{
				BaseURL:              "http://localhost:7880",
				GitHubClientID:       "gh-id",
				GitHubClientSecret:   "gh-secret",
				GoogleClientID:       "g-id",
				GoogleClientSecret:   "g-secret",
				FacebookClientID:     "fb-id",
				FacebookClientSecret: "fb-secret",
			},
			expected: []string{"facebook", "github", "google"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			reg := provider.NewRegistry(tc.cfg)
			got := reg.Enabled()
			if len(tc.expected) == 0 && len(got) == 0 {
				return
			}
			if !reflect.DeepEqual(got, tc.expected) {
				t.Errorf("got enabled providers %v, want %v", got, tc.expected)
			}
		})
	}
}

func TestRegistry_CallbackURL(t *testing.T) {
	tests := []struct {
		name             string
		baseURL          string
		expectedCallback string
	}{
		{
			name:             "without trailing slash",
			baseURL:          "https://example.com",
			expectedCallback: "https://example.com/api/auth/oauth/github/callback",
		},
		{
			name:             "with trailing slash",
			baseURL:          "https://example.com/",
			expectedCallback: "https://example.com/api/auth/oauth/github/callback",
		},
		{
			name:             "with multiple trailing slashes",
			baseURL:          "https://example.com///",
			expectedCallback: "https://example.com/api/auth/oauth/github/callback",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			reg := provider.NewRegistry(provider.Config{
				BaseURL:            tc.baseURL,
				GitHubClientID:     "client-id",
				GitHubClientSecret: "client-secret",
			})

			p, ok := reg.Get("github")
			if !ok {
				t.Fatal("expected github provider to be registered")
			}

			ghp, ok := p.(*provider.GitHubProvider)
			if !ok {
				t.Fatalf("expected *provider.GitHubProvider, got %T", p)
			}

			if ghp.OAuthConfig().RedirectURL != tc.expectedCallback {
				t.Errorf("got redirect url %q, want %q", ghp.OAuthConfig().RedirectURL, tc.expectedCallback)
			}
		})
	}
}

func TestRegistry_NilAndMissingProvider(t *testing.T) {
	var nilReg *provider.Registry
	if p, ok := nilReg.Get("github"); ok || p != nil {
		t.Error("expected nil registry Get to return false, nil")
	}
	if enabled := nilReg.Enabled(); enabled != nil {
		t.Errorf("expected nil registry Enabled to return nil, got %v", enabled)
	}

	emptyReg := provider.NewRegistry(provider.Config{})
	if p, ok := emptyReg.Get("nonexistent"); ok || p != nil {
		t.Error("expected non-existent provider to return false, nil")
	}
}

func TestProvider_AuthCodeURL_PKCE(t *testing.T) {
	reg := provider.NewRegistry(provider.Config{
		BaseURL:            "http://localhost:7880",
		GitHubClientID:     "gh-id",
		GitHubClientSecret: "gh-secret",
	})

	p, ok := reg.Get("github")
	if !ok {
		t.Fatal("expected github provider")
	}

	// 1. Without verifier
	authURL := p.AuthCodeURL("random-state", "")
	parsed, err := url.Parse(authURL)
	if err != nil {
		t.Fatalf("failed to parse auth url: %v", err)
	}
	q := parsed.Query()
	if q.Get("state") != "random-state" {
		t.Errorf("got state %q, want %q", q.Get("state"), "random-state")
	}
	if q.Get("code_challenge") != "" {
		t.Error("expected no code_challenge when verifier is empty")
	}

	// 2. With PKCE verifier
	authURLPKCE := p.AuthCodeURL("random-state", "test-verifier-string")
	parsedPKCE, err := url.Parse(authURLPKCE)
	if err != nil {
		t.Fatalf("failed to parse auth url: %v", err)
	}
	qPKCE := parsedPKCE.Query()
	if qPKCE.Get("code_challenge") == "" {
		t.Error("expected code_challenge when verifier is provided")
	}
	if qPKCE.Get("code_challenge_method") != "S256" {
		t.Errorf("got code_challenge_method %q, want %q", qPKCE.Get("code_challenge_method"), "S256")
	}
}

func TestProvider_Exchange(t *testing.T) {
	reg := provider.NewRegistry(provider.Config{
		BaseURL:            "http://localhost:7880",
		GoogleClientID:     "google-id",
		GoogleClientSecret: "google-secret",
	})

	p, ok := reg.Get("google")
	if !ok {
		t.Fatal("expected google provider")
	}

	// Default implementation returns ErrNotImplemented
	_, err := p.Exchange(context.Background(), "code123", "verifier123")
	if err != provider.ErrNotImplemented {
		t.Errorf("got error %v, want %v", err, provider.ErrNotImplemented)
	}

	// Custom Exchange hook
	bp := p.(*provider.BaseProvider)
	bp.SetExchangeFunc(func(ctx context.Context, code, verifier string) (provider.Profile, error) {
		return provider.Profile{
			ProviderUserID: "12345",
			Email:          "user@google.com",
			EmailVerified:  true,
			Name:           "Google User",
			AvatarURL:      "https://google.com/avatar.png",
		}, nil
	})

	profile, err := p.Exchange(context.Background(), "code123", "verifier123")
	if err != nil {
		t.Fatalf("unexpected exchange error: %v", err)
	}
	if profile.ProviderUserID != "12345" || profile.Email != "user@google.com" || !profile.EmailVerified {
		t.Errorf("unexpected profile: %+v", profile)
	}
}
