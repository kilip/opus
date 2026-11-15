// Package handler implements the auth HTTP handlers.
package handler

import (
	"github.com/gofiber/fiber/v3"
	"github.com/kilip/opus/internal/auth/repo"
	"github.com/kilip/opus/internal/platform/httpx"
)

// Handler holds auth HTTP handlers.
type Handler struct {
	repo      repo.Repo
	secret    string
	cookieCfg CookieConfig
}

// CookieConfig holds cookie parameters derived from app config.
type CookieConfig struct {
	Secure  bool
	MaxAge  int
	BaseURL string
}

// New returns a Handler.
func New(r repo.Repo, secret string, cc CookieConfig) *Handler {
	return &Handler{repo: r, secret: secret, cookieCfg: cc}
}

// Register mounts auth routes on r.
func (h *Handler) Register(r fiber.Router) {
	r.Get("/me", h.me)
	r.Post("/logout", h.logout)
}

// me godoc
// @Summary Get current user
// @Description Returns the authenticated user's profile
// @ID authMe
// @Tags auth
// @Produce json
// @Success 200 {object} httpx.Response[MeResponse]
// @Failure 401 {object} httpx.ErrorResponse
// @Router /auth/me [get]
func (h *Handler) me(c fiber.Ctx) error {
	sess, err := resolveSession(c, h.repo, h.secret)
	if err != nil {
		return httpx.Unauthorized()
	}
	return httpx.OK(c, MeResponse{
		ID:    sess.User.ID,
		Email: sess.User.Email,
	})
}

// logout godoc
// @Summary Logout
// @Description Deletes the current session and clears the session cookie
// @ID authLogout
// @Tags auth
// @Produce json
// @Success 204 "No Content"
// @Router /auth/logout [post]
func (h *Handler) logout(c fiber.Ctx) error {
	raw := c.Cookies("opus_session")
	if raw != "" {
		hash := hashToken(raw, h.secret)
		// noop if session not found
		_ = h.repo.DeleteByTokenHash(c.Context(), hash)
	}
	// clear cookie
	c.Cookie(&fiber.Cookie{
		Name:     "opus_session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Secure:   h.cookieCfg.Secure,
		HTTPOnly: true,
		SameSite: "Lax",
	})
	return c.SendStatus(fiber.StatusNoContent)
}
