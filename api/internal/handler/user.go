package handler

import (
	"github.com/gofiber/fiber/v3"
	"github.com/kilip/opus/api/internal/service"
)

type UserHandler struct {
	userService service.UserServiceInterface
}

func NewUserHandler(userService service.UserServiceInterface) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

// Me returns the currently authenticated user
func (h *UserHandler) Me(c fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	if userID == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "User not authenticated")
	}

	user, err := h.userService.GetUserByID(c.Context(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to fetch user")
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    user,
	})
}
