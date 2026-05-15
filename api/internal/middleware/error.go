package middleware

import (
	"errors"
	"fmt"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/opus/api/internal/model"
)

func ErrorHandler(c fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	var e *fiber.Error
	if errors.As(err, &e) {
		code = e.Code
	}

	return c.Status(code).JSON(model.ApiResponse{
		Success: false,
		Data:    nil,
		Error: &model.ApiError{
			Code:    fmt.Sprintf("ERR_%d", code),
			Message: err.Error(),
		},
	})
}
