package middleware

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/kilip/opus/server/internal/auth"
)

// Auth checks stateful access token validity in the database before processing requests.
func Auth(cfg auth.Config, repo auth.Repository) fiber.Handler {
	return func(c fiber.Ctx) error {
		tokenStr := c.Cookies("opus_access_token")
		if tokenStr == "" {
			authHeader := c.Get("Authorization")
			if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
				tokenStr = authHeader[7:]
			}
		}

		if tokenStr == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "unauthorized: missing access token",
			})
		}

		hashedToken := auth.HashToken(tokenStr)
		tokenRecord, err := repo.FindTokenByHash(c.Context(), hashedToken)
		if err != nil || tokenRecord.RevokedAt != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "unauthorized: invalid or revoked token",
			})
		}

		if tokenRecord.Type != "access" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "unauthorized: invalid token type",
			})
		}

		if time.Now().After(tokenRecord.ExpiresAt) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "unauthorized: token expired",
			})
		}

		token, err := jwt.ParseWithClaims(tokenStr, &auth.Claims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(cfg.JWTSecret), nil
		})

		if err != nil || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "unauthorized: invalid signature",
			})
		}

		claims, ok := token.Claims.(*auth.Claims)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "unauthorized: invalid claim type",
			})
		}

		c.Locals("auth_claims", claims)
		return c.Next()
	}
}
