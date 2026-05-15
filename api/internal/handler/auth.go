package handler

import (
	"fmt"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/opus/api/internal/config"
	"github.com/kilip/opus/api/internal/model"
	"github.com/kilip/opus/api/internal/service"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	googleoauth2 "google.golang.org/api/oauth2/v2"
	"google.golang.org/api/option"
)

type AuthHandler struct {
	authService *service.AuthService
	userService *service.UserService
	cfg         *config.Config
}

func NewAuthHandler(authService *service.AuthService, userService *service.UserService, cfg *config.Config) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		userService: userService,
		cfg:         cfg,
	}
}

func (h *AuthHandler) getGoogleConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     h.cfg.Auth.Google.ClientID,
		ClientSecret: h.cfg.Auth.Google.ClientSecret,
		RedirectURL:  h.cfg.Auth.Google.RedirectURL,
		Endpoint:     google.Endpoint,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
	}
}

func (h *AuthHandler) getGitHubConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     h.cfg.Auth.GitHub.ClientID,
		ClientSecret: h.cfg.Auth.GitHub.ClientSecret,
		RedirectURL:  h.cfg.Auth.GitHub.RedirectURL,
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://github.com/login/oauth/authorize",
			TokenURL: "https://github.com/login/oauth/access_token",
		},
		Scopes: []string{"user:email"},
	}
}

// Login handles email/password login (dev only)
func (h *AuthHandler) Login(c fiber.Ctx) error {
	if h.cfg.Server.Env != "development" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "FORBIDDEN",
				"message": "Email/password login is disabled in production.",
			},
		})
	}

	var req model.LoginRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	// Simple mock login for development
	user, err := h.userService.GetUserByEmail(c.Context(), req.Email)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Invalid credentials")
	}

	accessToken, refreshToken, err := h.authService.IssueTokens(c.Context(), user.ID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to issue tokens")
	}

	h.setRefreshTokenCookie(c, refreshToken)

	return c.JSON(fiber.Map{
		"success": true,
		"data": model.AuthResponse{
			AccessToken: accessToken,
			User:        user,
		},
	})
}

// Refresh handles token refresh
func (h *AuthHandler) Refresh(c fiber.Ctx) error {
	rawRefreshToken := c.Cookies("refresh_token")
	if rawRefreshToken == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "Missing refresh token")
	}

	accessToken, refreshToken, err := h.authService.RefreshTokens(c.Context(), rawRefreshToken)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Invalid refresh token")
	}

	userID, _ := h.authService.ValidateAccessToken(accessToken)
	user, err := h.userService.GetUserByID(c.Context(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to fetch user")
	}

	h.setRefreshTokenCookie(c, refreshToken)

	return c.JSON(fiber.Map{
		"success": true,
		"data": model.AuthResponse{
			AccessToken: accessToken,
			User:        user,
		},
	})
}

// Logout handles user logout
func (h *AuthHandler) Logout(c fiber.Ctx) error {
	rawRefreshToken := c.Cookies("refresh_token")
	if rawRefreshToken != "" {
		_ = h.authService.Logout(c.Context(), rawRefreshToken)
	}

	c.ClearCookie("refresh_token")

	return c.JSON(fiber.Map{
		"success": true,
		"data":    nil,
	})
}

// GoogleLogin redirects to Google OAuth
func (h *AuthHandler) GoogleLogin(c fiber.Ctx) error {
	url := h.getGoogleConfig().AuthCodeURL("state")
	return c.Redirect().To(url)
}

// GoogleCallback handles Google OAuth callback
func (h *AuthHandler) GoogleCallback(c fiber.Ctx) error {
	code := c.Query("code")
	if code == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Missing code")
	}

	tok, err := h.getGoogleConfig().Exchange(c.Context(), code)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to exchange code")
	}

	client := h.getGoogleConfig().Client(c.Context(), tok)
	svc, err := googleoauth2.NewService(c.Context(), option.WithHTTPClient(client))
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to create oauth2 service")
	}

	info, err := svc.Userinfo.Get().Do()
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to get user info")
	}

	user, err := h.authService.UpsertOAuthUser(c.Context(), "google", info.Id, info.Email, info.Name, info.Picture)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to upsert user")
	}

	accessToken, refreshToken, err := h.authService.IssueTokens(c.Context(), user.ID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to issue tokens")
	}

	h.setRefreshTokenCookie(c, refreshToken)

	// Redirect to dash with access token
	return c.Redirect().To(fmt.Sprintf("%s/auth/callback?token=%s", "http://localhost:3000", accessToken))
}

// GitHubLogin redirects to GitHub OAuth
func (h *AuthHandler) GitHubLogin(c fiber.Ctx) error {
	url := h.getGitHubConfig().AuthCodeURL("state")
	return c.Redirect().To(url)
}

// GitHubCallback handles GitHub OAuth callback
func (h *AuthHandler) GitHubCallback(c fiber.Ctx) error {
	code := c.Query("code")
	if code == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Missing code")
	}

	tok, err := h.getGitHubConfig().Exchange(c.Context(), code)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to exchange code")
	}

	// Fetch GitHub user info (basic implementation)
	client := h.getGitHubConfig().Client(c.Context(), tok)
	resp, err := client.Get("https://api.github.com/user")
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to get user info")
	}
	defer resp.Body.Close()

	var ghUser struct {
		ID     int    `json:"id"`
		Login  string `json:"login"`
		Email  string `json:"email"`
		Avatar string `json:"avatar_url"`
	}
	if err := fiber.Unmarshal(resp.Body, &ghUser); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to parse user info")
	}

	user, err := h.authService.UpsertOAuthUser(c.Context(), "github", fmt.Sprintf("%d", ghUser.ID), ghUser.Email, ghUser.Login, ghUser.Avatar)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to upsert user")
	}

	accessToken, refreshToken, err := h.authService.IssueTokens(c.Context(), user.ID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to issue tokens")
	}

	h.setRefreshTokenCookie(c, refreshToken)

	return c.Redirect().To(fmt.Sprintf("%s/auth/callback?token=%s", "http://localhost:3000", accessToken))
}


func (h *AuthHandler) setRefreshTokenCookie(c fiber.Ctx, refreshToken string) {
	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		HTTPOnly: true,
		Secure:   h.cfg.Server.Env == "production",
		SameSite: "Strict",
		MaxAge:   h.cfg.Auth.RefreshTokenTTL * 60,
	})
}
