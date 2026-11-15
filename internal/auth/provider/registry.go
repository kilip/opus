package provider

import (
	"sort"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/endpoints"
	"golang.org/x/oauth2/github"
)

// Config holds OAuth provider credentials and base URL.
type Config struct {
	BaseURL              string
	GitHubClientID       string
	GitHubClientSecret   string
	GoogleClientID       string
	GoogleClientSecret   string
	FacebookClientID     string
	FacebookClientSecret string
}

// Registry manages configured OAuth2 providers.
type Registry struct {
	providers map[string]Provider
}

// NewRegistry initializes an OAuth2 provider registry, building only providers
// whose client ID and client secret are both non-empty.
func NewRegistry(cfg Config) *Registry {
	r := &Registry{
		providers: make(map[string]Provider),
	}

	base := strings.TrimRight(cfg.BaseURL, "/")

	if cfg.GitHubClientID != "" && cfg.GitHubClientSecret != "" {
		r.providers["github"] = NewGitHubProvider(oauth2.Config{
			ClientID:     cfg.GitHubClientID,
			ClientSecret: cfg.GitHubClientSecret,
			Endpoint:     github.Endpoint,
			RedirectURL:  base + "/api/auth/oauth/github/callback",
			Scopes:       []string{"read:user", "user:email"},
		})
	}

	if cfg.GoogleClientID != "" && cfg.GoogleClientSecret != "" {
		r.providers["google"] = NewBaseProvider("google", oauth2.Config{
			ClientID:     cfg.GoogleClientID,
			ClientSecret: cfg.GoogleClientSecret,
			Endpoint:     endpoints.Google,
			RedirectURL:  base + "/api/auth/oauth/google/callback",
			Scopes:       []string{"openid", "profile", "email"},
		})
	}

	if cfg.FacebookClientID != "" && cfg.FacebookClientSecret != "" {
		r.providers["facebook"] = NewBaseProvider("facebook", oauth2.Config{
			ClientID:     cfg.FacebookClientID,
			ClientSecret: cfg.FacebookClientSecret,
			Endpoint:     endpoints.Facebook,
			RedirectURL:  base + "/api/auth/oauth/facebook/callback",
			Scopes:       []string{"email", "public_profile"},
		})
	}

	return r
}

// Get returns the provider by name, if registered.
func (r *Registry) Get(name string) (Provider, bool) {
	if r == nil || r.providers == nil {
		return nil, false
	}
	p, ok := r.providers[name]
	return p, ok
}

// Enabled returns a sorted list of names of all registered providers.
func (r *Registry) Enabled() []string {
	if r == nil || r.providers == nil {
		return nil
	}
	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Register adds or replaces a provider in the registry.
func (r *Registry) Register(p Provider) {
	if r.providers == nil {
		r.providers = make(map[string]Provider)
	}
	r.providers[p.Name()] = p
}
