package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/opus/api/internal/model"
	"github.com/kilip/opus/api/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestUserHandler_Me(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserService := mocks.NewMockUserServiceInterface(ctrl)
	handler := NewUserHandler(mockUserService)

	app := fiber.New()
	// Middleware to set userID in Locals
	app.Use(func(c fiber.Ctx) error {
		c.Locals("userID", "user_123")
		return c.Next()
	})
	app.Get("/me", handler.Me)

	t.Run("success", func(t *testing.T) {
		mockUser := &model.User{ID: "user_123", Email: "test@example.com"}
		mockUserService.EXPECT().GetUserByID(gomock.Any(), "user_123").Return(mockUser, nil)

		req := httptest.NewRequest(http.MethodGet, "/me", nil)
		resp, _ := app.Test(req)

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("not_found", func(t *testing.T) {
		mockUserService.EXPECT().GetUserByID(gomock.Any(), "user_123").Return(nil, model.ErrUserNotFound)

		req := httptest.NewRequest(http.MethodGet, "/me", nil)
		resp, _ := app.Test(req)

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	})
}
