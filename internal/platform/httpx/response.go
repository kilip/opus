package httpx

import (
	"github.com/gofiber/fiber/v3"
)

type Meta struct {
	Total      *int   `json:"total,omitempty"`
	NextCursor string `json:"nextCursor,omitempty"`
}

type Response[T any] struct {
	Data  T      `json:"data"`
	Meta  *Meta  `json:"meta,omitempty"`
	Error string `json:"error,omitempty"`
}

type ErrorResponse struct {
	Data  any    `json:"data"`
	Meta  *Meta  `json:"meta,omitempty"`
	Error string `json:"error" validate:"required"`
}

func OK[T any](c fiber.Ctx, data T) error {
	return c.Status(fiber.StatusOK).JSON(Response[T]{
		Data: data,
	})
}

func Created[T any](c fiber.Ctx, data T) error {
	return c.Status(fiber.StatusCreated).JSON(Response[T]{
		Data: data,
	})
}

func List[T any](c fiber.Ctx, items []T, meta *Meta) error {
	if items == nil {
		items = []T{}
	}
	return c.Status(fiber.StatusOK).JSON(Response[[]T]{
		Data: items,
		Meta: meta,
	})
}
