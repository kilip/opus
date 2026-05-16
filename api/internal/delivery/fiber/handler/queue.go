// api/internal/delivery/fiber/handler/queue.go
package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/opus/api/internal/repository/queue"
)

type QueueHandler struct {
	driver queue.QueueDriver
}

func NewQueueHandler(driver queue.QueueDriver) *QueueHandler {
	return &QueueHandler{driver: driver}
}

// ListDeadLetters returns a paginated list of dead letter jobs.
// GET /queue/dead
func (h *QueueHandler) ListDeadLetters(c fiber.Ctx) error {
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	items, err := h.driver.ListDeadLetters(c.Context(), limit, offset)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    items,
	})
}

// RetryDeadLetter moves a dead letter job back to the pending queue.
// POST /queue/dead/:id/retry
func (h *QueueHandler) RetryDeadLetter(c fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Missing job ID")
	}

	if err := h.driver.RetryDeadLetter(c.Context(), id); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Job successfully moved back to pending queue",
	})
}

// DeleteDeadLetter removes a dead letter job without retrying.
// DELETE /queue/dead/:id
func (h *QueueHandler) DeleteDeadLetter(c fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Missing job ID")
	}

	if err := h.driver.DeleteDeadLetter(c.Context(), id); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Job successfully removed from dead letter queue",
	})
}
