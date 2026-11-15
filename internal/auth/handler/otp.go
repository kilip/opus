package handler

import (
	"github.com/gofiber/fiber/v3"
	"github.com/kilip/opus/internal/auth/domain"
	"github.com/kilip/opus/internal/platform/httpx"
)

// requestOTP godoc
// @Summary Request OTP
// @Description Sends a one-time code to the given email if it matches the owner
// @ID authOTPRequest
// @Tags auth
// @Accept json
// @Produce json
// @Param body body domain.OTPRequestBody true "Email"
// @Success 202 "Accepted"
// @Failure 400 {object} httpx.ErrorResponse
// @Router /auth/otp/request [post]
func (h *Handler) requestOTP(c fiber.Ctx) error {
	var body domain.OTPRequestBody
	if err := c.Bind().JSON(&body); err != nil || body.Email == "" {
		return httpx.BadRequest(httpx.ErrBadRequest)
	}
	// always 202: response must not reveal whether the email exists
	if err := h.svc.RequestOTP(c.Context(), body.Email); err != nil {
		return mapErr(err)
	}
	return c.SendStatus(fiber.StatusAccepted)
}

// verifyOTP godoc
// @Summary Verify OTP
// @Description Verifies the one-time code, creates a session, sets the session cookie, and returns the user
// @ID authOTPVerify
// @Tags auth
// @Accept json
// @Produce json
// @Param body body domain.OTPVerifyBody true "Email and code"
// @Success 200 {object} httpx.Response[domain.UserResponse]
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 429 {object} httpx.ErrorResponse
// @Router /auth/otp/verify [post]
func (h *Handler) verifyOTP(c fiber.Ctx) error {
	var body domain.OTPVerifyBody
	if err := c.Bind().JSON(&body); err != nil || body.Email == "" || body.Code == "" {
		return httpx.BadRequest(httpx.ErrBadRequest)
	}
	user, token, err := h.svc.VerifyOTP(c.Context(), body.Email, body.Code)
	if err != nil {
		return mapErr(err)
	}
	h.setSessionCookie(c, token, h.cookie.MaxAge)
	return httpx.OK(c, domain.UserResponse{ID: user.ID, Email: user.Email})
}
