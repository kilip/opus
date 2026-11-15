package handler

// ponytail: onboarding open to first visitor; add setup token before exposing to the internet

import (
	"github.com/gofiber/fiber/v3"
	"github.com/kilip/opus/internal/auth/domain"
	"github.com/kilip/opus/internal/platform/httpx"
)

// setupStatus godoc
// @Summary Setup status
// @Description Returns whether initial owner setup is required
// @ID setupStatus
// @Tags setup
// @Produce json
// @Success 200 {object} httpx.Response[domain.SetupStatusResponse]
// @Router /setup/status [get]
func (h *Handler) setupStatus(c fiber.Ctx) error {
	needs, err := h.svc.NeedsSetup(c.Context())
	if err != nil {
		return mapErr(err)
	}
	return httpx.OK(c, domain.SetupStatusResponse{NeedsSetup: needs})
}

// setupRequestOTP godoc
// @Summary Setup: request OTP
// @Description Sends a one-time code to the given email for initial owner setup
// @ID setupRequestOTP
// @Tags setup
// @Accept json
// @Produce json
// @Param body body domain.OTPRequestBody true "Email"
// @Success 202 "Accepted"
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 409 {object} httpx.ErrorResponse
// @Router /setup/otp [post]
func (h *Handler) setupRequestOTP(c fiber.Ctx) error {
	var body domain.OTPRequestBody
	if err := c.Bind().JSON(&body); err != nil || body.Email == "" {
		return httpx.BadRequest(httpx.ErrBadRequest)
	}
	if err := h.svc.RequestSetupOTP(c.Context(), body.Email); err != nil {
		return mapErr(err)
	}
	return c.SendStatus(fiber.StatusAccepted)
}

// setupComplete godoc
// @Summary Setup: complete owner creation
// @Description Verifies the OTP, creates the owner user and session in one transaction, sets cookie
// @ID setupComplete
// @Tags setup
// @Accept json
// @Produce json
// @Param body body domain.OTPVerifyBody true "Email and code"
// @Success 200 {object} httpx.Response[domain.UserResponse]
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 409 {object} httpx.ErrorResponse
// @Router /setup [post]
func (h *Handler) setupComplete(c fiber.Ctx) error {
	var body domain.OTPVerifyBody
	if err := c.Bind().JSON(&body); err != nil || body.Email == "" || body.Code == "" {
		return httpx.BadRequest(httpx.ErrBadRequest)
	}
	user, token, err := h.svc.CompleteSetup(c.Context(), body.Email, body.Code)
	if err != nil {
		return mapErr(err)
	}
	h.setSessionCookie(c, token, h.cookie.MaxAge)
	return httpx.OK(c, domain.UserResponse{ID: user.ID, Email: user.Email})
}
