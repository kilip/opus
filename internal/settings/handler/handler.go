package handler

import (
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/opus/internal/platform/httpx"
	"github.com/kilip/opus/internal/settings/domain"
	"github.com/kilip/opus/internal/settings/service"
)

// Handler exposes HTTP endpoints for settings.
type Handler struct {
	svc *service.Settings
}

// New returns a new settings Handler.
func New(svc *service.Settings) *Handler {
	return &Handler{svc: svc}
}

// Register mounts settings routes under r.
func (h *Handler) Register(r fiber.Router) {
	r.Get("/settings/github-token", h.getGitHubToken)
	r.Put("/settings/github-token", h.setGitHubToken)
	r.Delete("/settings/github-token", h.deleteGitHubToken)
}

// @Summary Get GitHub token status
// @Description Returns whether a GitHub token is configured, its last 4 characters, update timestamp, env override flag, and whether re-entry is required.
// @ID getGitHubTokenStatus
// @Tags settings
// @Produce json
// @Success 200 {object} httpx.Response[domain.GitHubTokenStatusResponse]
// @Failure 500 {object} httpx.ErrorResponse
// @Router /settings/github-token [get]
func (h *Handler) getGitHubToken(c fiber.Ctx) error {
	status, err := h.svc.GetGitHubTokenStatus(c.Context())
	if err != nil {
		return err
	}
	return httpx.OK(c, status)
}

// @Summary Set GitHub token
// @Description Validates a GitHub personal access token against the GitHub API and stores it encrypted at rest.
// @ID setGitHubToken
// @Tags settings
// @Accept json
// @Produce json
// @Param body body domain.SetGitHubTokenRequest true "GitHub token payload"
// @Success 200 {object} httpx.Response[domain.GitHubTokenStatusResponse]
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /settings/github-token [put]
func (h *Handler) setGitHubToken(c fiber.Ctx) error {
	var req domain.SetGitHubTokenRequest
	if err := c.Bind().Body(&req); err != nil {
		return httpx.BadRequest(errors.New("invalid JSON body"))
	}

	status, err := h.svc.SetGitHubToken(c.Context(), req.Token)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrEmptyGitHubToken):
			return httpx.BadRequest(err)
		case errors.Is(err, domain.ErrInvalidGitHubToken):
			return httpx.BadRequest(errors.New("GitHub rejected the provided token"))
		default:
			return err
		}
	}
	return httpx.OK(c, status)
}

// @Summary Delete GitHub token
// @Description Removes the stored GitHub token from the database.
// @ID deleteGitHubToken
// @Tags settings
// @Produce json
// @Success 204
// @Failure 500 {object} httpx.ErrorResponse
// @Router /settings/github-token [delete]
func (h *Handler) deleteGitHubToken(c fiber.Ctx) error {
	if err := h.svc.DeleteGitHubToken(c.Context()); err != nil {
		return err
	}
	return c.SendStatus(fiber.StatusNoContent)
}
