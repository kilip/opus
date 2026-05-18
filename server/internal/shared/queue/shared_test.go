package queue_test

import (
	"context"
	"testing"
	"time"

	"github.com/kilip/opus/server/internal/shared/queue"
)

func TestJobOptions(t *testing.T) {
	job := &queue.Job{}

	queue.WithPriority(10)(job)
	if job.Priority != 10 {
		t.Errorf("expected priority 10, got %d", job.Priority)
	}

	queue.WithQueue("high-priority")(job)
	if job.Queue != "high-priority" {
		t.Errorf("expected queue high-priority, got %s", job.Queue)
	}

	queue.WithMaxRetries(5)(job)
	if job.MaxRetries != 5 {
		t.Errorf("expected max retries 5, got %d", job.MaxRetries)
	}

	now := time.Now()
	queue.WithProcessAt(now)(job)
	if !job.ProcessAt.Equal(now) {
		t.Errorf("expected process at %v, got %v", now, job.ProcessAt)
	}

	delay := 5 * time.Minute
	queue.WithDelay(delay)(job)
	if job.ProcessAt.Before(now.Add(delay)) {
		t.Errorf("expected process at after delay, got %v", job.ProcessAt)
	}
}

func TestNoop(t *testing.T) {
	q := queue.NewNoopQueue()
	id, err := q.Enqueue(context.Background(), "type", nil)
	if err != nil || id == "" {
		t.Errorf("noop enqueue failed")
	}
	q.RegisterHandler("type", nil)
	if err := q.Start(context.Background()); err != nil {
		t.Errorf("noop start failed")
	}
	if err := q.Shutdown(context.Background()); err != nil {
		t.Errorf("noop shutdown failed")
	}
	j, _ := q.Inspect(context.Background(), id)
	if j == nil {
		t.Errorf("noop inspect failed")
	}

	bus := queue.NewNoopEventBus()
	if err := bus.Publish(context.Background(), queue.Event{}); err != nil {
		t.Errorf("noop publish failed")
	}
	bus.Subscribe("topic", nil)
}
