package csrf

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/opus/internal/platform/httpx"
)

// Config defines the configuration for the CSRF origin-check middleware.
type Config struct {
	BaseURL        string
	TrustedOrigins []string
}

// normalizeOrigin extracts scheme://host[:port] from a raw origin or URL string.
func normalizeOrigin(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return strings.TrimRight(strings.ToLower(raw), "/")
	}

	scheme := strings.ToLower(u.Scheme)
	hostname := strings.ToLower(u.Hostname())
	port := u.Port()

	// Normalize default HTTP/HTTPS ports
	if (scheme == "http" && port == "80") || (scheme == "https" && port == "443") {
		port = ""
	}

	if port != "" {
		return fmt.Sprintf("%s://%s:%s", scheme, hostname, port)
	}
	return fmt.Sprintf("%s://%s", scheme, hostname)
}

// isMutating returns true for state-changing HTTP methods.
func isMutating(method string) bool {
	switch method {
	case fiber.MethodPost, fiber.MethodPut, fiber.MethodPatch, fiber.MethodDelete:
		return true
	default:
		return false
	}
}

// New returns a Fiber middleware that checks Origin and Sec-Fetch-Site headers
// to protect cookie-authenticated sessions from Cross-Site Request Forgery (CSRF).
//
// ponytail: origin-check CSRF protection for cookie sessions; upgrade to Double Submit Cookie or Synchronizer Token Pattern if cross-site embedding is ever needed.
func New(cfg Config) fiber.Handler {
	allowed := make(map[string]struct{})
	if base := normalizeOrigin(cfg.BaseURL); base != "" {
		allowed[base] = struct{}{}
	}
	for _, entry := range cfg.TrustedOrigins {
		for _, origin := range strings.Split(entry, ",") {
			if norm := normalizeOrigin(origin); norm != "" {
				allowed[norm] = struct{}{}
			}
		}
	}

	return func(c fiber.Ctx) error {
		if !isMutating(c.Method()) {
			return c.Next()
		}

		origin := strings.TrimSpace(c.Get(fiber.HeaderOrigin))
		if origin != "" {
			norm := normalizeOrigin(origin)
			if _, ok := allowed[norm]; !ok {
				return httpx.CSRFForbidden()
			}
			return c.Next()
		}

		secFetchSite := strings.ToLower(strings.TrimSpace(c.Get(fiber.HeaderSecFetchSite)))
		if secFetchSite == "cross-site" {
			return httpx.CSRFForbidden()
		}

		return c.Next()
	}
}
