package auth_test

import (
	"context"
	"testing"

	"github.com/kilip/opus/server/internal/auth"
)

type dummyProvider struct{}

func (d *dummyProvider) Name() string                { return "dummy" }
func (d *dummyProvider) AuthURL(state string) string { return "http://dummy.com/auth?state=" + state }
func (d *dummyProvider) Exchange(ctx context.Context, code string) (*auth.OAuthProfile, error) {
	return &auth.OAuthProfile{Provider: "dummy", ProviderID: "d123", Email: "dummy@test.com"}, nil
}

func TestOAuthProviderRegistry(t *testing.T) {
	p := &dummyProvider{}
	registry := auth.NewProviderRegistry(p)

	resolved, err := registry.Get("dummy")
	if err != nil {
		t.Fatalf("failed to resolve dummy provider: %v", err)
	}

	if resolved.Name() != "dummy" {
		t.Errorf("resolved invalid provider name")
	}

	_, err = registry.Get("nonexistent")
	if err != auth.ErrProviderNotFound {
		t.Errorf("expected ErrProviderNotFound, got %v", err)
	}
}
