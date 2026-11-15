package handler

import (
	"errors"
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/kilip/opus/internal/auth/domain"
	"github.com/kilip/opus/internal/platform/httpx"
)

// OAuthStateCookie is the name of the signed cookie holding OAuth state and PKCE verifier.
const OAuthStateCookie = "opus_oauth"

const oauthStateTTL = 10 * 60 // 10 minutes in seconds

// listProviders godoc
// @Summary List enabled OAuth providers
// @Description Returns a list of configured and enabled OAuth identity providers
// @ID authListProviders
// @Tags auth
// @Produce json
// @Success 200 {object} httpx.Response[[]domain.ProviderItem]
// @Router /auth/providers [get]
func (h *Handler) listProviders(c fiber.Ctx) error {
	return httpx.OK(c, h.svc.EnabledProviders())
}

// oauthAuthorize godoc
// @Summary OAuth authorize
// @Description Initiates OAuth login flow, redirecting to provider
// @ID authOAuthAuthorize
// @Tags auth
// @Param provider path string true "Provider name (e.g. github, google, facebook)"
// @Param link query string false "Set to 1 to link provider to current authenticated user"
// @Success 302 "Redirect to provider auth URL"
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Router /auth/oauth/{provider} [get]
func (h *Handler) oauthAuthorize(c fiber.Ctx) error {
	providerName := strings.ToLower(c.Params("provider"))
	var linkUserID string
	if c.Query("link") == "1" {
		sess, err := h.svc.Session(c.Context(), c.Cookies(SessionCookie))
		if err != nil {
			return mapErr(err)
		}
		linkUserID = sess.UserID.String()
	}

	authURL, state, verifier, err := h.svc.OAuthAuthorize(providerName)
	if err != nil {
		return mapErr(err)
	}

	cookieVal := h.svc.SignOAuthCookie(state, verifier, linkUserID)
	c.Cookie(&fiber.Cookie{
		Name:     OAuthStateCookie,
		Value:    cookieVal,
		Path:     "/",
		MaxAge:   oauthStateTTL,
		Secure:   h.cookie.Secure,
		HTTPOnly: true,
		SameSite: "Lax",
	})

	return c.Redirect().Status(fiber.StatusFound).To(authURL)
}

// oauthCallback godoc
// @Summary OAuth callback
// @Description Handles OAuth provider callback, sets session cookie and redirects
// @ID authOAuthCallback
// @Tags auth
// @Param provider path string true "Provider name"
// @Param code query string false "Authorization code"
// @Param state query string false "OAuth state parameter"
// @Success 302 "Redirect to / on success or /login?error=<code> on failure"
// @Failure 404 {object} httpx.ErrorResponse
// @Failure 409 {object} httpx.ErrorResponse
// @Router /auth/oauth/{provider}/callback [get]
func (h *Handler) oauthCallback(c fiber.Ctx) error {
	providerName := strings.ToLower(c.Params("provider"))

	if !h.svc.HasProvider(providerName) {
		return httpx.NotFound(domain.ErrProviderNotFound)
	}

	// Always clear the OAuth state cookie after use
	c.Cookie(&fiber.Cookie{
		Name:     OAuthStateCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Secure:   h.cookie.Secure,
		HTTPOnly: true,
		SameSite: "Lax",
	})

	if qErr := c.Query("error"); qErr != "" {
		return c.Redirect().Status(fiber.StatusFound).To("/login?error=" + url.QueryEscape(qErr))
	}

	cookieVal := c.Cookies(OAuthStateCookie)
	cookieState, verifier, linkUserID, ok := h.svc.VerifyOAuthCookie(cookieVal)
	if !ok || cookieState == "" {
		return c.Redirect().Status(fiber.StatusFound).To("/login?error=STATE_MISMATCH")
	}

	queryState := c.Query("state")
	code := c.Query("code")

	if linkUserID != "" {
		parsedUID, err := uuid.Parse(linkUserID)
		if err != nil {
			return httpx.NewError(fiber.StatusBadRequest, domain.ErrUnauthorized)
		}
		if err := h.svc.OAuthLinkCallback(c.Context(), parsedUID, providerName, code, cookieState, queryState, verifier, clientAuditMeta(c)); err != nil {
			return mapErr(err)
		}
		return c.Redirect().Status(fiber.StatusFound).To("/")
	}

	token, err := h.svc.OAuthCallback(c.Context(), providerName, code, cookieState, queryState, verifier, clientAuditMeta(c))
	if err != nil {
		if errors.Is(err, domain.ErrProviderNotFound) {
			return httpx.NotFound(err)
		}
		return c.Redirect().Status(fiber.StatusFound).To("/login?error=" + url.QueryEscape(err.Error()))
	}

	h.setSessionCookie(c, token, h.cookie.MaxAge)
	return c.Redirect().Status(fiber.StatusFound).To("/")
}
