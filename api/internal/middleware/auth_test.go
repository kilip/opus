package middleware

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/opus/api/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestAuth(t *testing.T) {
	app := fiber.New()

	t.Run("MissingHeader", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockAuthService := mocks.NewMockAuthServiceInterface(ctrl)
		app.Get("/test", Auth(mockAuthService), func(c fiber.Ctx) error {
			return c.SendString("OK")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		json.Unmarshal(body, &result)

		assert.False(t, result["success"].(bool))
		errMap := result["error"].(map[string]interface{})
		assert.Equal(t, "UNAUTHORIZED", errMap["code"])
		assert.Equal(t, "Missing authorization header", errMap["message"])
	})

	t.Run("InvalidFormat", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockAuthService := mocks.NewMockAuthServiceInterface(ctrl)
		// We don't re-use the same app to avoid middleware stack pollution if any
		app := fiber.New()
		app.Get("/test", Auth(mockAuthService), func(c fiber.Ctx) error {
			return c.SendString("OK")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		json.Unmarshal(body, &result)

		assert.False(t, result["success"].(bool))
		errMap := result["error"].(map[string]interface{})
		assert.Equal(t, "UNAUTHORIZED", errMap["code"])
		assert.Equal(t, "Invalid authorization header format", errMap["message"])
	})

	t.Run("InvalidToken", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockAuthService := mocks.NewMockAuthServiceInterface(ctrl)
		mockAuthService.EXPECT().ValidateAccessToken("invalid-token").Return("", errors.New("invalid token"))

		app := fiber.New()
		app.Get("/test", Auth(mockAuthService), func(c fiber.Ctx) error {
			return c.SendString("OK")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		json.Unmarshal(body, &result)

		assert.False(t, result["success"].(bool))
		errMap := result["error"].(map[string]interface{})
		assert.Equal(t, "UNAUTHORIZED", errMap["code"])
		assert.Equal(t, "Invalid or expired access token", errMap["message"])
	})

	t.Run("Success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockAuthService := mocks.NewMockAuthServiceInterface(ctrl)
		mockAuthService.EXPECT().ValidateAccessToken("valid-token").Return("user_123", nil)

		app := fiber.New()
		app.Get("/test", Auth(mockAuthService), func(c fiber.Ctx) error {
			userID := c.Locals("userID").(string)
			assert.Equal(t, "user_123", userID)
			return c.SendString("OK")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer valid-token")
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		assert.Equal(t, "OK", string(body))
	})
}
