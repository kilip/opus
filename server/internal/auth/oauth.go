package auth

import (
	"context"
	"sync"
)

// OAuthProvider wraps authorization routing for federated callbacks.
type OAuthProvider interface {
	// Name returns the identifier of the identity provider.
	Name() string

	// AuthURL computes the secure redirect URL with dynamic state.
	AuthURL(state string) string

	// Exchange exchanges the authorization code for the OAuthProfile.
	Exchange(ctx context.Context, code string) (*OAuthProfile, error)
}

// ProviderRegistry acts as a safe, concurrent lookup engine for identity endpoints.
type ProviderRegistry struct {
	mu        sync.RWMutex
	providers map[string]OAuthProvider
}

// NewProviderRegistry compiles a concurrency-safe lookup map.
func NewProviderRegistry(providers ...OAuthProvider) *ProviderRegistry {
	reg := &ProviderRegistry{
		providers: make(map[string]OAuthProvider),
	}
	for _, p := range providers {
		reg.providers[p.Name()] = p
	}
	return reg
}

// Register maps an active provider instance dynamically.
func (r *ProviderRegistry) Register(p OAuthProvider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[p.Name()] = p
}

// Get resolves a registered provider or returns ErrProviderNotFound.
func (r *ProviderRegistry) Get(name string) (OAuthProvider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, exists := r.providers[name]
	if !exists {
		return nil, ErrProviderNotFound
	}
	return p, nil
}
