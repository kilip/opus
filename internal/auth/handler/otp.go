package handler

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/opus/internal/auth/domain"
	"github.com/kilip/opus/internal/platform/httpx"
)

const otpTTLMinutes = 10

// generateOTPCode returns a random 6-digit string from crypto/rand.
func generateOTPCode() (string, error) {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	n := binary.BigEndian.Uint32(b) % 1_000_000
	return fmt.Sprintf("%06d", n), nil
}

// requestOTP godoc
// @Summary Request OTP
// @Description Sends a one-time code to the given email if it matches the owner
// @ID authOTPRequest
// @Tags auth
// @Accept json
// @Produce json
// @Param body body OTPRequestBody true "Email"
// @Success 202 "Accepted"
// @Failure 400 {object} httpx.ErrorResponse
// @Router /auth/otp/request [post]
func (h *Handler) requestOTP(c fiber.Ctx) error {
	var body OTPRequestBody
	if err := c.Bind().JSON(&body); err != nil || body.Email == "" {
		return httpx.BadRequest(httpx.ErrBadRequest)
	}

	// AC1: always 202; only send mail if user exists.
	user, userErr := h.otpRepo.FindUserByEmail(c.Context(), body.Email)
	if userErr == nil && user != nil {
		code, err := generateOTPCode()
		if err != nil {
			return err
		}
		expiresAt := h.now().Add(otpTTLMinutes * 60 * 1e9) // 10 min
		codeHash := hashToken(code, h.secret)

		if _, err := h.otpRepo.CreateOTP(c.Context(), body.Email, codeHash, expiresAt); err != nil {
			return err
		}
		_ = h.mailer.Send(c.Context(), body.Email, "Your OTP code",
			fmt.Sprintf("Your one-time code is: %s (valid for %d minutes)", code, otpTTLMinutes))
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
// @Param body body OTPVerifyBody true "Email and code"
// @Success 200 {object} httpx.Response[OTPVerifyResponse]
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 429 {object} httpx.ErrorResponse
// @Router /auth/otp/verify [post]
func (h *Handler) verifyOTP(c fiber.Ctx) error {
	var body OTPVerifyBody
	if err := c.Bind().JSON(&body); err != nil || body.Email == "" || body.Code == "" {
		return httpx.BadRequest(httpx.ErrBadRequest)
	}

	otp, err := h.otpRepo.FindByEmail(c.Context(), body.Email)
	if err != nil || otp == nil {
		return httpx.NewError(fiber.StatusUnauthorized, domain.ErrInvalidOTP)
	}

	// AC5: lockout before checking code.
	if otp.Attempts >= domain.MaxOTPAttempts {
		_ = h.otpRepo.DeleteByEmail(c.Context(), body.Email)
		return httpx.NewError(fiber.StatusTooManyRequests, domain.ErrTooManyAttempts)
	}

	// AC5: expiry check.
	if h.now().After(otp.ExpiresAt) {
		return httpx.NewError(fiber.StatusUnauthorized, domain.ErrOTPExpired)
	}

	// AC6: constant-time compare.
	incoming := hashToken(body.Code, h.secret)
	if subtle.ConstantTimeCompare([]byte(otp.CodeHash), []byte(incoming)) != 1 {
		attempts, _ := h.otpRepo.IncrementAttempts(c.Context(), otp.ID)
		if attempts >= domain.MaxOTPAttempts {
			_ = h.otpRepo.DeleteByEmail(c.Context(), body.Email)
			return httpx.NewError(fiber.StatusTooManyRequests, domain.ErrTooManyAttempts)
		}
		return httpx.NewError(fiber.StatusUnauthorized, domain.ErrInvalidOTP)
	}

	// AC4: single-use — delete OTP before creating session.
	_ = h.otpRepo.DeleteByEmail(c.Context(), body.Email)

	// Look up user.
	user, err := h.otpRepo.FindUserByEmail(c.Context(), body.Email)
	if err != nil || user == nil {
		return httpx.NewError(fiber.StatusUnauthorized, domain.ErrInvalidOTP)
	}

	// Create session token.
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
	sessionExpiresAt := h.now().Add(sessionTTL)
	if _, err = h.repo.CreateSession(c.Context(), user.ID, tokenHash, sessionExpiresAt); err != nil {
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

	return httpx.OK(c, OTPVerifyResponse{ID: user.ID, Email: user.Email})
}
