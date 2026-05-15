package handler

import (
	"github.com/gofiber/fiber/v3"
)

type HealthHandler struct{}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

func (h *HealthHandler) Check(c fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"status": "up",
		},
	})
}
