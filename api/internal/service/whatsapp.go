package service

import (
	"context"
)

// WhatsAppService defines the interface for WhatsApp related operations.
type WhatsAppService interface {
	// GetStatus returns the current connection status and JID for the user.
	GetStatus(ctx context.Context, userID string) (status string, jid string, err error)
	// Connect initiates the WhatsApp connection process (e.g., generates QR code via SSE).
	Connect(ctx context.Context, userID string) error
	// Disconnect closes the active WhatsApp connection for the user.
	Disconnect(ctx context.Context, userID string) error
	// SendMessage sends a text message to a target JID.
	SendMessage(ctx context.Context, userID string, targetJID string, message string) error
}
