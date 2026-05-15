package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/opus/api/internal/service"
)

func Auth(authService *service.AuthService) fiber.Handler {
	return func(c fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"data":    nil,
				"error": fiber.Map{
					"code":    "UNAUTHORIZED",
					"message": "Missing authorization header",
				},
			})
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"data":    nil,
				"error": fiber.Map{
					"code":    "UNAUTHORIZED",
					"message": "Invalid authorization header format",
				},
			})
		}

		userID, err := authService.ValidateAccessToken(parts[1])
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"data":    nil,
				"error": fiber.Map{
					"code":    "UNAUTHORIZED",
					"message": "Invalid or expired access token",
				},
			})
		}

		c.Locals("userID", userID)
		return c.Next()
	}
}
