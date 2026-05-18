// Package memory provides an in-memory implementation of the EventBus.
package memory

import (
	"context"
	"path"
	"sync"

	"github.com/kilip/opus/server/internal/shared/queue"
)

type subscription struct {
	pattern string
	handler queue.EventHandler
}

// InMemoryEventBus is a synchronous, in-process EventBus implementation.
type InMemoryEventBus struct {
	mu   sync.RWMutex
	subs []subscription
}

// NewInMemoryEventBus creates a new InMemoryEventBus.
func NewInMemoryEventBus() queue.EventBus {
	return &InMemoryEventBus{}
}

// Publish distributes the event to all matching subscribers synchronously.
func (b *InMemoryEventBus) Publish(ctx context.Context, event queue.Event) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, sub := range b.subs {
		matched, err := path.Match(sub.pattern, event.Topic)
		if err != nil || !matched {
			continue
		}
		// Errors are intentionally swallowed here. Handler implementations are
		// responsible for logging their own errors (ADR-006).
		_ = sub.handler(ctx, event)
	}
	return nil
}

// Subscribe registers a handler for a topic pattern.
func (b *InMemoryEventBus) Subscribe(topic string, handler queue.EventHandler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.subs = append(b.subs, subscription{pattern: topic, handler: handler})
}
