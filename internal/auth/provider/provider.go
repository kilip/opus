package provider

import (
	"context"
	"errors"

	"golang.org/x/oauth2"
)

// ErrNotImplemented is returned when Exchange is called without an active implementation.
var ErrNotImplemented = errors.New("provider exchange not implemented")

// Profile holds user profile information returned from an OAuth2 provider.
type Profile struct {
	ProviderUserID string
	Email          string
	EmailVerified  bool
	Name           string
	AvatarURL      string
}

// Provider represents an OAuth2 identity provider.
type Provider interface {
	Name() string
	AuthCodeURL(state, verifier string) string
	Exchange(ctx context.Context, code, verifier string) (Profile, error)
}

// BaseProvider is an OAuth2 provider implementation wrapping oauth2.Config.
type BaseProvider struct {
	name       string
	cfg        oauth2.Config
	exchangeFn func(ctx context.Context, code, verifier string) (Profile, error)
}

// NewBaseProvider creates a new BaseProvider with the specified name and OAuth2 configuration.
func NewBaseProvider(name string, cfg oauth2.Config) *BaseProvider {
	return &BaseProvider{
		name: name,
		cfg:  cfg,
	}
}

// Name returns the provider identifier.
func (p *BaseProvider) Name() string {
	return p.name
}

// AuthCodeURL returns the provider authorization code URL, optionally configuring PKCE challenge.
func (p *BaseProvider) AuthCodeURL(state, verifier string) string {
	var opts []oauth2.AuthCodeOption
	if verifier != "" {
		opts = append(opts, oauth2.S256ChallengeOption(verifier))
	}
	return p.cfg.AuthCodeURL(state, opts...)
}

// Exchange exchanges authorization code and verifier for a user profile.
// ponytail: provider-specific Exchange profile fetching is implemented in subsequent issues.
func (p *BaseProvider) Exchange(ctx context.Context, code, verifier string) (Profile, error) {
	if p.exchangeFn != nil {
		return p.exchangeFn(ctx, code, verifier)
	}
	return Profile{}, ErrNotImplemented
}

// OAuthConfig returns the underlying oauth2.Config.
func (p *BaseProvider) OAuthConfig() oauth2.Config {
	return p.cfg
}

// SetExchangeFunc sets a custom exchange function for testing or extension.
func (p *BaseProvider) SetExchangeFunc(fn func(ctx context.Context, code, verifier string) (Profile, error)) {
	p.exchangeFn = fn
}
