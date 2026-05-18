package queue

import "context"

// Event is an immutable domain event emitted by a feature domain.
type Event struct {
	// Topic is a dot-separated string identifying the event category.
	// Convention: "<domain>.<action>" (e.g. "agent.completed", "vault.written").
	Topic string
	// Payload is the serialised event data. JSON encoding is recommended.
	Payload []byte
	// Source is the domain package that emitted the event, used for logging and tracing.
	Source string
}

// EventHandler processes an event.
type EventHandler func(ctx context.Context, event Event) error

//go:generate mockgen -destination=mock_eventbus.go -package=queue . EventBus

// EventBus defines the contract for in-process publish/subscribe messaging.
// All subscribers are notified synchronously in the order they were registered.
// Implementations must be safe for concurrent use.
type EventBus interface {
	Publish(ctx context.Context, event Event) error
	Subscribe(topic string, handler EventHandler)
}
