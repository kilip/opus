package queue

import "context"

type noopQueue struct{}

// NewNoopQueue creates a new Queue that does nothing, for use in tests.
func NewNoopQueue() Queue {
	return &noopQueue{}
}

func (q *noopQueue) Enqueue(ctx context.Context, jobType string, payload []byte, opts ...JobOption) (string, error) {
	return "noop-job-id", nil
}

func (q *noopQueue) RegisterHandler(jobType string, handler Handler) {}

func (q *noopQueue) Start(ctx context.Context) error { return nil }

func (q *noopQueue) Shutdown(ctx context.Context) error { return nil }

func (q *noopQueue) Inspect(ctx context.Context, jobID string) (*Job, error) {
	return &Job{ID: "noop-job-id", Status: JobStatusCompleted}, nil
}

type noopEventBus struct{}

// NewNoopEventBus creates a new EventBus that does nothing, for use in tests.
func NewNoopEventBus() EventBus {
	return &noopEventBus{}
}

func (b *noopEventBus) Publish(ctx context.Context, event Event) error {
	return nil
}

func (b *noopEventBus) Subscribe(topic string, handler EventHandler) {}
