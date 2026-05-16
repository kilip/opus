package handler

import (
	"bufio"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/opus/api/internal/delivery/fiber/middleware"
	"github.com/kilip/opus/api/internal/service"
)

type SSEHandler struct {
	hub service.SSEHub
	log *slog.Logger
}

func NewSSEHandler(hub service.SSEHub, log *slog.Logger) *SSEHandler {
	return &SSEHandler{hub: hub, log: log}
}

func (h *SSEHandler) Stream(c fiber.Ctx) error {
	userID := c.Locals("userID").(string)
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

	return c.SendStreamWriter(func(w *bufio.Writer) {
		ch := hub.Register(userID)
		defer hub.Unregister(userID, ch)

		// Initial connection event
		err := middleware.WriteEvent(w, service.SSEEvent{
			Type:    "connected",
			Payload: map[string]string{"time": time.Now().Format(time.RFC3339)},
		})
		if err != nil {
			h.log.Error("Failed to write initial connection event", "userID", userID, "error", err)
			return
		}
		if err := w.Flush(); err != nil {
			h.log.Error("Failed to flush initial connection event", "userID", userID, "error", err)
			return
		}

		ticker := time.NewTicker(30 * time.Second) // Heartbeat
		defer ticker.Stop()

		for {
			select {
			case <-c.Context().Done():
				return
			case event, ok := <-ch:
				if !ok {
					h.log.Debug("SSE event channel closed", "userID", userID)
					return
				}
				h.log.Debug("Sending SSE event", "userID", userID, "type", event.Type)
				_ = middleware.WriteEvent(w, event)
				if err := w.Flush(); err != nil {
					h.log.Error("Failed to flush event", "userID", userID, "error", err)
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
