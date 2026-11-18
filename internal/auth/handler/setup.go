package handler

import (
	"net/mail"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/opus/internal/auth/domain"
	"github.com/kilip/opus/internal/platform/httpx"
	"github.com/kilip/opus/internal/user"
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
	return httpx.OK(c, domain.SetupStatusResponse{
		NeedsSetup:    needs,
		RequiresToken: needs,
	})
}

func (h *Handler) checkSetupOpenAndToken(c fiber.Ctx) error {
	needs, err := h.svc.NeedsSetup(c.Context())
	if err != nil {
		return mapErr(err)
	}
	if !needs {
		return mapErr(domain.ErrSetupClosed)
	}
	if !h.svc.ValidateSetupToken(c.Get("X-Setup-Token")) {
		return mapErr(domain.ErrSetupTokenInvalid)
	}
	return nil
}

// setupRequestOTP godoc
// @Summary Setup: request OTP
// @Description Sends a one-time code to the given email for initial owner setup
// @ID setupRequestOTP
// @Tags setup
// @Accept json
// @Produce json
// @Param X-Setup-Token header string false "First-run setup token"
// @Param body body domain.OTPRequestBody true "Email"
// @Success 202 "Accepted"
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Failure 409 {object} httpx.ErrorResponse
// @Failure 429 {object} httpx.ErrorResponse
// @Router /setup/otp [post]
func (h *Handler) setupRequestOTP(c fiber.Ctx) error {
	if err := h.checkSetupOpenAndToken(c); err != nil {
		return err
	}
	var body domain.OTPRequestBody
	if err := c.Bind().JSON(&body); err != nil {
		return httpx.BadRequest(httpx.ErrBadRequest)
	}
	email := user.NormalizeEmail(body.Email)
	if email == "" || len(email) > 254 {
		return httpx.BadRequest(httpx.ErrBadRequest)
	}
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
		return httpx.BadRequest(httpx.ErrBadRequest)
	}
	if err := h.svc.RequestSetupOTP(c.Context(), email); err != nil {
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
// @Param X-Setup-Token header string false "First-run setup token"
// @Param body body domain.OTPVerifyBody true "Email and code"
// @Success 200 {object} httpx.Response[domain.UserResponse]
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Failure 409 {object} httpx.ErrorResponse
// @Failure 429 {object} httpx.ErrorResponse
// @Router /setup [post]
func (h *Handler) setupComplete(c fiber.Ctx) error {
	if err := h.checkSetupOpenAndToken(c); err != nil {
		return err
	}
	var body domain.OTPVerifyBody
	if err := c.Bind().JSON(&body); err != nil || body.Code == "" {
		return httpx.BadRequest(httpx.ErrBadRequest)
	}
	email := user.NormalizeEmail(body.Email)
	if email == "" || len(email) > 254 {
		return httpx.BadRequest(httpx.ErrBadRequest)
	}
	usr, token, err := h.svc.CompleteSetup(c.Context(), email, body.Code, clientAuditMeta(c))
	if err != nil {
		return mapErr(err)
	}
	h.setSessionCookie(c, token, h.cookie.MaxAge)
	return httpx.OK(c, domain.UserResponse{ID: usr.ID, Email: usr.Email})
}
