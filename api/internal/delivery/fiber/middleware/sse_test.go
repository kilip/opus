package middleware_test

import (
	"context"
	"testing"
	"time"

	"github.com/kilip/opus/api/internal/delivery/fiber/middleware"
	"github.com/kilip/opus/api/internal/service"
	"github.com/stretchr/testify/assert"
)

func TestSSEHub_PublishToUser(t *testing.T) {
	hub := middleware.NewSSEHub()
	userID := "user-1"
	
	// Type assertion to access internal Register method
	concreteHub := hub.(interface {
		Register(string) chan service.SSEEvent
	})

	ch := concreteHub.Register(userID)
	
	event := service.SSEEvent{
		Type:    "test_event",
		Payload: "hello",
	}

	// Publish in a separate goroutine to avoid blocking
	go func() {
		time.Sleep(100 * time.Millisecond)
		err := hub.Publish(context.Background(), userID, event)
		assert.NoError(t, err)
	}()

	select {
	case received := <-ch:
		assert.Equal(t, event.Type, received.Type)
		assert.Equal(t, event.Payload, received.Payload)
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestSSEHub_PublishNoClient(t *testing.T) {
	hub := middleware.NewSSEHub()
	err := hub.Publish(context.Background(), "non-existent", service.SSEEvent{Type: "test"})
	assert.Error(t, err)
}
