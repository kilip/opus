package handler

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/kilip/opus/server/internal/auth"
	"github.com/kilip/opus/server/internal/delivery/gofiber"
)

// AuthHandler manages registration and logout routes.
type AuthHandler struct {
	svc  *auth.Service
	repo auth.Repository
}

// NewAuthHandler maps service interfaces.
func NewAuthHandler(svc *auth.Service, repo auth.Repository) *AuthHandler {
	return &AuthHandler{
		svc:  svc,
		repo: repo,
	}
}

// Register registers a credential user.
func (h *AuthHandler) Register(c fiber.Ctx) error {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Name     string `json:"name"`
	}
	if err := c.Bind().Body(&req); err != nil {
		return gofiber.Error(c, fiber.StatusBadRequest, "invalid_request_body", "Invalid Request Body", err.Error())
	}

	user, tokens, err := h.svc.Register(c.Context(), req.Email, req.Password, req.Name)
	if err != nil {
		return gofiber.Error(c, fiber.StatusConflict, "registration_conflict", "Registration Conflict", err.Error())
	}

	h.setCookies(c, tokens)
	return gofiber.OK(c, user)
}

// Login maps user login.
func (h *AuthHandler) Login(c fiber.Ctx) error {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.Bind().Body(&req); err != nil {
		return gofiber.Error(c, fiber.StatusBadRequest, "invalid_request_body", "Invalid Request Body", err.Error())
	}

	user, tokens, err := h.svc.Login(c.Context(), req.Email, req.Password)
	if err != nil {
		return gofiber.Error(c, fiber.StatusUnauthorized, "invalid_credentials", "Invalid Credentials", "Invalid email or password")
	}

	h.setCookies(c, tokens)
	return gofiber.OK(c, user)
}

// Refresh rotates active tokens.
func (h *AuthHandler) Refresh(c fiber.Ctx) error {
	refreshToken := c.Cookies("opus_refresh_token")
	if refreshToken == "" {
		return gofiber.Error(c, fiber.StatusBadRequest, "missing_refresh_token", "Missing Refresh Token", "Missing refresh token cookie")
	}

	tokens, err := h.svc.Refresh(c.Context(), refreshToken)
	if err != nil {
		return gofiber.Error(c, fiber.StatusUnauthorized, "token_rotation_failed", "Token Rotation Failed", err.Error())
	}

	h.setCookies(c, tokens)
	return gofiber.OK(c, fiber.Map{"status": "rotated"})
}

// Logout revokes active records.
func (h *AuthHandler) Logout(c fiber.Ctx) error {
	claims, ok := c.Locals("auth_claims").(*auth.Claims)
	if !ok {
		return gofiber.Error(c, fiber.StatusInternalServerError, "invalid_session_context", "Invalid Session Context", "Failed to retrieve session claims from locals")
	}

	if err := h.svc.Logout(c.Context(), claims.SessionID); err != nil {
		return gofiber.Error(c, fiber.StatusInternalServerError, "logout_failed", "Logout Failed", err.Error())
	}

	h.clearCookies(c)
	return gofiber.OK(c, fiber.Map{"status": "logged out"})
}

// Me retrieves current user profile.
func (h *AuthHandler) Me(c fiber.Ctx) error {
	claims, ok := c.Locals("auth_claims").(*auth.Claims)
	if !ok {
		return gofiber.Error(c, fiber.StatusInternalServerError, "invalid_session_context", "Invalid Session Context", "Failed to retrieve session claims from locals")
	}

	user, err := h.repo.FindUserByID(c.Context(), claims.Subject)
	if err != nil {
		return gofiber.Error(c, fiber.StatusNotFound, "user_not_found", "User Not Found", "Failed to find user profile matching subject ID")
	}

	return gofiber.OK(c, user)
}

// OAuthRedirect handles redirect paths.
func (h *AuthHandler) OAuthRedirect(c fiber.Ctx) error {
	providerID := c.Params("provider")
	u, err := uuid.NewV7()
	if err != nil {
		return gofiber.Error(c, fiber.StatusInternalServerError, "oauth_state_generation_failed", "OAuth State Generation Failed", err.Error())
	}
	state := u.String()

	err = h.repo.CreateOAuthState(c.Context(), state, providerID, time.Now().Add(10*time.Minute))
	if err != nil {
		return gofiber.Error(c, fiber.StatusInternalServerError, "oauth_state_save_failed", "OAuth State Save Failed", err.Error())
	}

	provider, err := h.svc.Registry().Get(providerID)
	if err != nil {
		return gofiber.Error(c, fiber.StatusBadRequest, "oauth_provider_not_found", "OAuth Provider Not Found", err.Error())
	}

	return c.Redirect().To(provider.AuthURL(state))
}

// OAuthCallback captures third-party profiles.
func (h *AuthHandler) OAuthCallback(c fiber.Ctx) error {
	providerID := c.Params("provider")
	code := c.Query("code")
	state := c.Query("state")

	user, tokens, err := h.svc.OAuthCallback(c.Context(), providerID, code, state)
	if err != nil {
		return gofiber.Error(c, fiber.StatusUnauthorized, "oauth_callback_unauthorized", "OAuth Callback Unauthorized", err.Error())
	}

	h.setCookies(c, tokens)
	return gofiber.OK(c, user)
}

func (h *AuthHandler) setCookies(c fiber.Ctx, tokens *auth.TokenPair) {
	c.Cookie(&fiber.Cookie{
		Name:     "opus_access_token",
		Value:    tokens.AccessToken,
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Strict",
		Path:     "/",
	})
	c.Cookie(&fiber.Cookie{
		Name:     "opus_refresh_token",
		Value:    tokens.RefreshToken,
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Strict",
		Path:     "/auth/refresh",
	})
}

func (h *AuthHandler) clearCookies(c fiber.Ctx) {
	c.Cookie(&fiber.Cookie{
		Name:     "opus_access_token",
		Value:    "",
		Expires:  time.Unix(0, 0),
		HTTPOnly: true,
		Path:     "/",
	})
	c.Cookie(&fiber.Cookie{
		Name:     "opus_refresh_token",
		Value:    "",
		Expires:  time.Unix(0, 0),
		HTTPOnly: true,
		Path:     "/auth/refresh",
	})
}
