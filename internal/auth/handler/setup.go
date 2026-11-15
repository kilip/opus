package handler

// ponytail: onboarding open to first visitor; add setup token before exposing to the internet

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

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
// @Success 200 {object} httpx.Response[SetupStatusResponse]
// @Router /setup/status [get]
func (h *Handler) setupStatus(c fiber.Ctx) error {
	n, err := h.repo.CountUsers(c.Context())
	if err != nil {
		return err
	}
	return httpx.OK(c, SetupStatusResponse{NeedsSetup: n == 0})
}

// setupRequestOTP godoc
// @Summary Setup: request OTP
// @Description Sends a one-time code to the given email for initial owner setup
// @ID setupRequestOTP
// @Tags setup
// @Accept json
// @Produce json
// @Param body body SetupOTPBody true "Email"
// @Success 202 "Accepted"
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 409 {object} httpx.ErrorResponse
// @Router /setup/otp [post]
func (h *Handler) setupRequestOTP(c fiber.Ctx) error {
	var body SetupOTPBody
	if err := c.Bind().JSON(&body); err != nil || body.Email == "" {
		return httpx.BadRequest(httpx.ErrBadRequest)
	}

	n, err := h.repo.CountUsers(c.Context())
	if err != nil {
		return err
	}
	if n > 0 {
		return httpx.Conflict(domain.ErrSetupClosed)
	}

	code, err := generateOTPCode()
	if err != nil {
		return err
	}
	expiresAt := h.now().Add(otpTTLMinutes * 60 * 1e9)
	codeHash := hashToken(code, h.secret)
	if _, err := h.otpRepo.CreateOTP(c.Context(), body.Email, codeHash, expiresAt); err != nil {
		return err
	}
	_ = h.mailer.Send(c.Context(), body.Email, "Your setup OTP code",
		fmt.Sprintf("Your one-time code is: %s (valid for %d minutes)", code, otpTTLMinutes))

	return c.SendStatus(fiber.StatusAccepted)
}

// setupComplete godoc
// @Summary Setup: complete owner creation
// @Description Verifies the OTP, creates the owner user and session in one transaction, sets cookie
// @ID setupComplete
// @Tags setup
// @Accept json
// @Produce json
// @Param body body SetupCompleteBody true "Email and code"
// @Success 200 {object} httpx.Response[SetupUserResponse]
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 409 {object} httpx.ErrorResponse
// @Router /setup [post]
func (h *Handler) setupComplete(c fiber.Ctx) error {
	var body SetupCompleteBody
	if err := c.Bind().JSON(&body); err != nil || body.Email == "" || body.Code == "" {
		return httpx.BadRequest(httpx.ErrBadRequest)
	}

	n, err := h.repo.CountUsers(c.Context())
	if err != nil {
		return err
	}
	if n > 0 {
		return httpx.Conflict(domain.ErrSetupClosed)
	}

	otp, err := h.otpRepo.FindByEmail(c.Context(), body.Email)
	if err != nil || otp == nil {
		return httpx.NewError(fiber.StatusUnauthorized, domain.ErrInvalidOTP)
	}
	if h.now().After(otp.ExpiresAt) {
		return httpx.NewError(fiber.StatusUnauthorized, domain.ErrOTPExpired)
	}
	incoming := hashToken(body.Code, h.secret)
	if otp.CodeHash != incoming {
		return httpx.NewError(fiber.StatusUnauthorized, domain.ErrInvalidOTP)
	}
	_ = h.otpRepo.DeleteByEmail(c.Context(), body.Email)

	rawBytes := make([]byte, 32)
	if _, err = rand.Read(rawBytes); err != nil {
		return err
	}
	rawToken := base64.RawURLEncoding.EncodeToString(rawBytes)
	tokenHash := hashToken(rawToken, h.secret)
	sessionTTL := time.Duration(h.cookieCfg.MaxAge) * time.Second
	if sessionTTL <= 0 {
		sessionTTL = 24 * time.Hour
	}
	sess, err := h.repo.CreateUserAndSession(c.Context(), body.Email, tokenHash, h.now().Add(sessionTTL))
	if err != nil {
		// Second concurrent call hits unique constraint → treat as setup closed
		n2, _ := h.repo.CountUsers(c.Context())
		if n2 > 0 {
			return httpx.Conflict(domain.ErrSetupClosed)
		}
		return err
	}

	c.Cookie(&fiber.Cookie{
		Name:     "opus_session",
		Value:    rawToken,
		Path:     "/",
		MaxAge:   h.cookieCfg.MaxAge,
		Secure:   h.cookieCfg.Secure,
		HTTPOnly: true,
		SameSite: "Lax",
	})

	return httpx.OK(c, SetupUserResponse{ID: sess.User.ID, Email: sess.User.Email})
}
