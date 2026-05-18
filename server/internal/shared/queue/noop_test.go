package queue_test

import (
	"context"
	"testing"

	"github.com/kilip/opus/server/internal/shared/queue"
)

func TestNoopQueue(t *testing.T) {
	q := queue.NewNoopQueue()
	id, err := q.Enqueue(context.Background(), "test:job", nil)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if id != "noop-job-id" {
		t.Errorf("expected noop-job-id, got %v", id)
	}

	job, err := q.Inspect(context.Background(), "any-id")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if job == nil || job.ID != "noop-job-id" || job.Status != queue.JobStatusCompleted {
		t.Errorf("expected completed noop job, got %v", job)
	}

	q.RegisterHandler("test:job", func(ctx context.Context, job *queue.Job) error { return nil })
	if err := q.Start(context.Background()); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if err := q.Shutdown(context.Background()); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestNoopEventBus(t *testing.T) {
	bus := queue.NewNoopEventBus()
	err := bus.Publish(context.Background(), queue.Event{})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	bus.Subscribe("test.*", func(ctx context.Context, event queue.Event) error { return nil })
}
