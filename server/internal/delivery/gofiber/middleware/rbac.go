package middleware

import (
	"github.com/gofiber/fiber/v3"
	"github.com/kilip/opus/server/internal/auth"
)

// Require evaluates domain authorization utilizing workspace claims.
func Require(policy auth.PolicyService, action string) fiber.Handler {
	return func(c fiber.Ctx) error {
		claims, ok := c.Locals("auth_claims").(*auth.Claims)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "unauthorized: missing token claims",
			})
		}

		domain := claims.WorkspaceID
		userID := claims.Subject
		resource := c.Path()

		allowed, err := policy.Enforce(userID, domain, resource, action)
		if err != nil || !allowed {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "forbidden: insufficient workspace permissions",
			})
		}

		return c.Next()
	}
}
