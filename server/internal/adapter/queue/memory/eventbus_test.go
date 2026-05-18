package memory_test

import (
	"context"
	"testing"

	"github.com/kilip/opus/server/internal/adapter/queue/memory"
	"github.com/kilip/opus/server/internal/shared/queue"
)

func TestInMemoryEventBus(t *testing.T) {
	bus := memory.NewInMemoryEventBus()
	received := false

	bus.Subscribe("agent.*", func(ctx context.Context, event queue.Event) error {
		if event.Topic == "agent.completed" {
			received = true
		}
		return nil
	})

	err := bus.Publish(context.Background(), queue.Event{Topic: "agent.completed", Source: "agent"})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if !received {
		t.Errorf("expected to receive event")
	}

	received = false
	err = bus.Publish(context.Background(), queue.Event{Topic: "vault.written", Source: "vault"})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if received {
		t.Errorf("expected not to receive event for non-matching topic")
	}
}
