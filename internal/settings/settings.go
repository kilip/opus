// Package settings manages encrypted runtime settings. Other packages depend on this package only,
// never on its subpackages.
package settings

import (
	"context"
	"net/http"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/opus/ent"
	"github.com/kilip/opus/internal/settings/domain"
	"github.com/kilip/opus/internal/settings/handler"
	"github.com/kilip/opus/internal/settings/repo"
	"github.com/kilip/opus/internal/settings/service"
)

// Sentinel errors re-exported from domain.
var (
	ErrEmptyGitHubToken   = domain.ErrEmptyGitHubToken
	ErrInvalidGitHubToken = domain.ErrInvalidGitHubToken
)

// Re-exported domain types.
type (
	GitHubTokenStatusResponse = domain.GitHubTokenStatusResponse
	SetGitHubTokenRequest     = domain.SetGitHubTokenRequest
)

// Config configures the Settings feature.
type Config struct {
	Secret           string
	GitHubAPIBaseURL string
	HTTPClient       *http.Client
	Getenv           func(string) string
}

// Settings is the settings feature facade.
type Settings struct {
	svc     *service.Settings
	handler *handler.Handler
}

// New creates a new Settings feature facade backed by Ent.
func New(client *ent.Client, cfg Config) (*Settings, error) {
	r := repo.New(client)
	svc, err := service.New(r, service.Config{
		Secret:           cfg.Secret,
		GitHubAPIBaseURL: cfg.GitHubAPIBaseURL,
		HTTPClient:       cfg.HTTPClient,
		Getenv:           cfg.Getenv,
	})
	if err != nil {
		return nil, err
	}
	return &Settings{
		svc:     svc,
		handler: handler.New(svc),
	}, nil
}

// Register mounts settings HTTP routes on r.
func (s *Settings) Register(r fiber.Router) {
	s.handler.Register(r)
}

// EffectiveGitHubToken returns GH_TOKEN if set, otherwise the decrypted stored token.
func (s *Settings) EffectiveGitHubToken(ctx context.Context) string {
	return s.svc.EffectiveGitHubToken(ctx)
}

// StoredGitHubToken returns the decrypted stored token, or "" if unconfigured or unreadable.
func (s *Settings) StoredGitHubToken(ctx context.Context) string {
	return s.svc.StoredGitHubToken(ctx)
}
