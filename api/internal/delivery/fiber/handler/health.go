package handler

import (
	"github.com/gofiber/fiber/v3"
	"github.com/kilip/opus/api/internal/model"
)

type HealthHandler struct{}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

func (h *HealthHandler) Check(c fiber.Ctx) error {
	return c.JSON(model.ApiResponse{
		Success: true,
		Data: map[string]string{
			"status":  "ok",
			"version": "1.0.1",
			"db":      "sqlite",
		},
		Error: nil,
	})
}
