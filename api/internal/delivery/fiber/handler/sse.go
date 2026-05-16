package handler

import (
	"bufio"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/opus/api/internal/delivery/fiber/middleware"
	"github.com/kilip/opus/api/internal/service"
)

type SSEHandler struct {
	hub service.SSEHub
}

func NewSSEHandler(hub service.SSEHub) *SSEHandler {
	return &SSEHandler{hub: hub}
}

func (h *SSEHandler) Stream(c fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	if userID == "" {
		return fiber.ErrUnauthorized
	}

	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")

	// We know the implementation is *middleware.sseHub
	hub := h.hub.(interface {
		Register(string) chan service.SSEEvent
		Unregister(string, chan service.SSEEvent)
	})

	ch := hub.Register(userID)
	defer hub.Unregister(userID, ch)

	return c.SendStreamWriter(func(w *bufio.Writer) {
		// Initial connection event
		_ = middleware.WriteEvent(w, service.SSEEvent{
			Type:    "connected",
			Payload: map[string]string{"time": time.Now().Format(time.RFC3339)},
		})
		_ = w.Flush()

		ticker := time.NewTicker(30 * time.Second) // Heartbeat
		defer ticker.Stop()

		for {
			select {
			case <-c.Context().Done():
				return
			case event, ok := <-ch:
				if !ok {
					return
				}
				_ = middleware.WriteEvent(w, event)
				if err := w.Flush(); err != nil {
					return
				}
			case <-ticker.C:
				_ = middleware.WriteEvent(w, service.SSEEvent{Type: "heartbeat", Payload: "ping"})
				if err := w.Flush(); err != nil {
					return
				}
			}
		}
	})
}
