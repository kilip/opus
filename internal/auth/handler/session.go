// Package handler implements the auth HTTP handlers.
package handler

import (
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/opus/internal/auth/domain"
	"github.com/kilip/opus/internal/auth/service"
	"github.com/kilip/opus/internal/platform/httpx"
)

// SessionCookie is the name of the session cookie.
const SessionCookie = "opus_session"

// CookieConfig holds cookie parameters derived from app config.
type CookieConfig struct {
	Secure bool
	MaxAge int
}

// Handler holds auth HTTP handlers.
type Handler struct {
	svc          *service.Auth
	cookie       CookieConfig
	emailLimiter fiber.Handler
	ipLimiter    fiber.Handler
}

// New returns a Handler backed by the auth service.
func New(svc *service.Auth, cc CookieConfig, rl ...RateLimitConfig) *Handler {
	var rlCfg RateLimitConfig
	if len(rl) > 0 {
		rlCfg = rl[0]
	}
	return &Handler{
		svc:          svc,
		cookie:       cc,
		emailLimiter: newEmailLimiter(rlCfg),
		ipLimiter:    newIPLimiter(rlCfg),
	}
}

// Register mounts protected auth routes on r.
func (h *Handler) Register(r fiber.Router) {
	r.Get("/me", h.me)
	r.Post("/logout", h.logout)
}

// RegisterPublic mounts public OTP routes (before auth middleware).
func (h *Handler) RegisterPublic(r fiber.Router) {
	r.Post("/otp/request", h.ipLimiter, h.emailLimiter, h.requestOTP)
	r.Post("/otp/verify", h.verifyOTP)
}

// RegisterSetup mounts public setup routes (before auth middleware).
func (h *Handler) RegisterSetup(r fiber.Router) {
	r.Get("/status", h.setupStatus)
	r.Post("/otp", h.ipLimiter, h.emailLimiter, h.setupRequestOTP)
	r.Post("", h.setupComplete)
}

// me godoc
// @Summary Get current user
// @Description Returns the authenticated user's profile
// @ID authMe
// @Tags auth
// @Produce json
// @Success 200 {object} httpx.Response[domain.UserResponse]
// @Failure 401 {object} httpx.ErrorResponse
// @Router /auth/me [get]
func (h *Handler) me(c fiber.Ctx) error {
	sess, err := h.svc.Session(c.Context(), c.Cookies(SessionCookie))
	if err != nil {
		return mapErr(err)
	}
	return httpx.OK(c, domain.UserResponse{ID: sess.User.ID, Email: sess.User.Email})
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
	h.svc.Logout(c.Context(), c.Cookies(SessionCookie))
	h.setSessionCookie(c, "", -1)
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) setSessionCookie(c fiber.Ctx, value string, maxAge int) {
	c.Cookie(&fiber.Cookie{
		Name:     SessionCookie,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		Secure:   h.cookie.Secure,
		HTTPOnly: true,
		SameSite: "Lax",
	})
}

// mapErr maps domain sentinel errors to httpx status errors.
func mapErr(err error) error {
	switch {
	case errors.Is(err, domain.ErrUnauthorized):
		return httpx.Unauthorized()
	case errors.Is(err, domain.ErrInvalidOTP), errors.Is(err, domain.ErrOTPExpired):
		return httpx.NewError(fiber.StatusUnauthorized, err)
	case errors.Is(err, domain.ErrTooManyAttempts), errors.Is(err, domain.ErrRateLimited):
		return httpx.NewError(fiber.StatusTooManyRequests, err)
	case errors.Is(err, domain.ErrSetupClosed):
		return httpx.Conflict(err)
	default:
		return err
	}
}
