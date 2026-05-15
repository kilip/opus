package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/opus/api/internal/config"
	"github.com/kilip/opus/api/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestAuthHandler_Logout(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAuthService := mocks.NewMockAuthServiceInterface(ctrl)
	mockUserService := mocks.NewMockUserServiceInterface(ctrl)
	cfg := &config.Config{} // Simplified config

	handler := NewAuthHandler(mockAuthService, mockUserService, cfg)

	app := fiber.New()
	app.Get("/logout", handler.Logout)

	t.Run("success_with_cookie", func(t *testing.T) {
		cookie := "refresh_token_value"
		mockAuthService.EXPECT().Logout(gomock.Any(), cookie).Return(nil)

		req := httptest.NewRequest(http.MethodGet, "/logout", nil)
		req.Header.Set("Cookie", "refresh_token="+cookie)
		resp, _ := app.Test(req)

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("success_without_cookie", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/logout", nil)
		resp, _ := app.Test(req)

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}
