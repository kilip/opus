package queue_test

import (
	"testing"

	"github.com/kilip/opus/server/internal/adapter/queue"
	"github.com/kilip/opus/server/internal/shared/logger"
	sharedq "github.com/kilip/opus/server/internal/shared/queue"
	"github.com/kilip/opus/server/internal/testutil"
)

func TestFactory(t *testing.T) {
	client := testutil.NewTestEntClient(t)

	cfg := sharedq.Config{Driver: sharedq.DriverDatabase, Concurrency: 2}
	q, err := queue.NewQueue(cfg, client, &logger.NoopLogger{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if q == nil {
		t.Errorf("expected non-nil queue")
	}

	bus := queue.NewEventBus()
	if bus == nil {
		t.Errorf("expected non-nil eventbus")
	}

	cfg.Driver = "invalid"
	_, err = queue.NewQueue(cfg, client, &logger.NoopLogger{})
	if err == nil {
		t.Errorf("expected error for unsupported driver")
	}
}
