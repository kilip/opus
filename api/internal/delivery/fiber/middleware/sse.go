package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/kilip/opus/api/internal/service"
)

type sseHub struct {
	// userID -> map[chan service.SSEEvent]bool
	clients    map[string]map[chan service.SSEEvent]bool
	clientsMu  sync.RWMutex
	broadcast  chan service.SSEEvent
	register   chan clientRegistration
	unregister chan clientRegistration
}

type clientRegistration struct {
	userID string
	ch     chan service.SSEEvent
}

func NewSSEHub() service.SSEHub {
	h := &sseHub{
		clients:    make(map[string]map[chan service.SSEEvent]bool),
		broadcast:  make(chan service.SSEEvent),
		register:   make(chan clientRegistration),
		unregister: make(chan clientRegistration),
	}
	go h.run()
	return h
}

func (h *sseHub) run() {
	for {
		select {
		case reg := <-h.register:
			h.clientsMu.Lock()
			if _, ok := h.clients[reg.userID]; !ok {
				h.clients[reg.userID] = make(map[chan service.SSEEvent]bool)
			}
			h.clients[reg.userID][reg.ch] = true
			h.clientsMu.Unlock()

		case unreg := <-h.unregister:
			h.clientsMu.Lock()
			if _, ok := h.clients[unreg.userID]; ok {
				delete(h.clients[unreg.userID], unreg.ch)
				close(unreg.ch)
				if len(h.clients[unreg.userID]) == 0 {
					delete(h.clients, unreg.userID)
				}
			}
			h.clientsMu.Unlock()

		case event := <-h.broadcast:
			h.clientsMu.RLock()
			for _, userClients := range h.clients {
				for ch := range userClients {
					select {
					case ch <- event:
					default:
						// Buffer full, skip or close? For SSE, we usually just skip if client is slow
					}
				}
			}
			h.clientsMu.RUnlock()
		}
	}
}

func (h *sseHub) Publish(ctx context.Context, userID string, event service.SSEEvent) error {
	h.clientsMu.RLock()
	defer h.clientsMu.RUnlock()

	userClients, ok := h.clients[userID]
	if !ok || len(userClients) == 0 {
		return fmt.Errorf("no active connections for user %s", userID)
	}

	for ch := range userClients {
		select {
		case ch <- event:
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Client slow, skip
		}
	}
	return nil
}

func (h *sseHub) Broadcast(ctx context.Context, event service.SSEEvent) error {
	select {
	case h.broadcast <- event:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Internal methods for Handler to use
func (h *sseHub) Register(userID string) chan service.SSEEvent {
	ch := make(chan service.SSEEvent, 10)
	h.register <- clientRegistration{userID: userID, ch: ch}
	return ch
}

func (h *sseHub) Unregister(userID string, ch chan service.SSEEvent) {
	h.unregister <- clientRegistration{userID: userID, ch: ch}
}

// WriteEvent helper to format SSE message
func WriteEvent(w io.Writer, event service.SSEEvent) error {
	data, err := json.Marshal(event.Payload)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, string(data))
	return err
}
