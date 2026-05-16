package service

import "context"

// SSEEvent represents a Server-Sent Event payload
type SSEEvent struct {
	Type    string `json:"type"`
	Payload any    `json:"payload"`
}

// SSEHub defines the interface for managing SSE connections and broadcasting events.
// It allows both global broadcasts and user-targeted publishing.
type SSEHub interface {
	// Publish sends an event to a specific user
	Publish(ctx context.Context, userID string, event SSEEvent) error
	// Broadcast sends an event to all connected clients
	Broadcast(ctx context.Context, event SSEEvent) error
}
