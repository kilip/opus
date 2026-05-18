package database_test

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kilip/opus/server/internal/adapter/queue/database"
	"github.com/kilip/opus/server/internal/shared/logger"
	"github.com/kilip/opus/server/internal/shared/queue"
	"github.com/kilip/opus/server/internal/testutil"
)

func TestDatabaseQueue(t *testing.T) {
	client := testutil.NewTestEntClient(t)
	log := &logger.NoopLogger{}
	q := database.NewDatabaseQueue(client, 2, log)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	processed := make(chan string, 1)
	q.RegisterHandler("test:job", func(ctx context.Context, j *queue.Job) error {
		processed <- j.ID
		return nil
	})

	if err := q.Start(ctx); err != nil {
		t.Fatalf("failed to start queue: %v", err)
	}

	payload := []byte("hello")
	id, err := q.Enqueue(ctx, "test:job", payload)
	if err != nil {
		t.Fatalf("failed to enqueue: %v", err)
	}

	select {
	case pid := <-processed:
		if pid != id {
			t.Errorf("expected job ID %s, got %s", id, pid)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for job processing")
	}

	// Verify job status with wait loop
	var j *queue.Job
	for i := 0; i < 20; i++ {
		j, err = q.Inspect(ctx, id)
		if err != nil {
			t.Logf("Inspect error: %v", err)
		} else if j.Status == queue.JobStatusCompleted {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}

	if j == nil {
		t.Fatalf("job not found")
	}
	if j.Status != queue.JobStatusCompleted {
		t.Errorf("expected job status completed, got %s", j.Status)
	}

	// Case 2: RegisterHandler panic after start
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic when calling RegisterHandler after Start")
		}
	}()
	q.RegisterHandler("late:job", func(ctx context.Context, j *queue.Job) error { return nil })
}

func TestDatabaseQueue_FailureAndRetry(t *testing.T) {
	client := testutil.NewTestEntClient(t)
	log, _ := logger.NewSlogLogger(logger.DefaultConfig())
	q := database.NewDatabaseQueue(client, 1, log)
	q.RetryBackoff = 100 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var failCount atomic.Int32
	q.RegisterHandler("fail:job", func(ctx context.Context, j *queue.Job) error {
		newCount := failCount.Add(1)
		if newCount == 1 {
			return fmt.Errorf("temporary failure")
		}
		return nil
	})

	if err := q.Start(ctx); err != nil {
		t.Fatalf("failed to start queue: %v", err)
	}

	id, err := q.Enqueue(ctx, "fail:job", []byte{}, queue.WithMaxRetries(2))
	if err != nil {
		t.Fatalf("failed to enqueue: %v", err)
	}

	// Wait for retry
	var j *queue.Job
	for i := 0; i < 30; i++ {
		j, err = q.Inspect(ctx, id)
		if err == nil && j.Status == queue.JobStatusCompleted {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}

	if j == nil {
		t.Fatalf("job not found after waiting")
	}
	if failCount.Load() != 2 {
		t.Errorf("expected 2 attempts, got %d", failCount.Load())
	}
	if j.Status != queue.JobStatusCompleted {
		t.Errorf("expected job status completed after retry, got %s", j.Status)
	}
}

func TestDatabaseQueue_DeadLetter(t *testing.T) {
	client := testutil.NewTestEntClient(t)
	log := &logger.NoopLogger{}
	q := database.NewDatabaseQueue(client, 1, log)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	q.RegisterHandler("dead:job", func(ctx context.Context, j *queue.Job) error {
		return fmt.Errorf("permanent failure")
	})

	_ = q.Start(ctx)

	id, err := q.Enqueue(ctx, "dead:job", []byte{}, queue.WithMaxRetries(1))
	if err != nil {
		t.Fatalf("failed to enqueue: %v", err)
	}

	var j *queue.Job
	for i := 0; i < 10; i++ {
		j, err = q.Inspect(ctx, id)
		if err == nil && j.Status == queue.JobStatusDead {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	if j == nil {
		t.Fatalf("job not found")
	}
	if j.Status != queue.JobStatusDead {
		t.Errorf("expected job status dead, got %s", j.Status)
	}
}

func TestDatabaseQueue_InspectNotFound(t *testing.T) {
	client := testutil.NewTestEntClient(t)
	q := database.NewDatabaseQueue(client, 1, &logger.NoopLogger{})
	_, err := q.Inspect(context.Background(), "non-existent")
	if err == nil {
		t.Errorf("expected error for non-existent job")
	}
}

func TestDatabaseQueue_Shutdown(t *testing.T) {
	client := testutil.NewTestEntClient(t)
	q := database.NewDatabaseQueue(client, 1, &logger.NoopLogger{})
	_ = q.Start(context.Background())
	err := q.Shutdown(context.Background())
	if err != nil {
		t.Errorf("expected no error on shutdown, got %v", err)
	}
}
