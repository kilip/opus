package handler

import (
	"bufio"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"
)

type SSEHandler struct{}

func NewSSEHandler() *SSEHandler {
	return &SSEHandler{}
}

func (h *SSEHandler) Stream(c fiber.Ctx) error {
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")

	return c.SendStreamWriter(func(w *bufio.Writer) {
		_, _ = fmt.Fprintf(w, "event: connected\ndata: %s\n\n", time.Now().Format(time.RFC3339))
		_ = w.Flush()

		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-c.Context().Done():
				return
			case t := <-ticker.C:
				_, _ = fmt.Fprintf(w, "event: heartbeat\ndata: %s\n\n", t.Format(time.RFC3339))
				if err := w.Flush(); err != nil {
					return
				}
			}
		}
	})
}
