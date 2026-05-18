//go:build integration

package sqlite_test

import (
	"context"
	"testing"
	"time"

	"github.com/kilip/opus/server/internal/adapter/entgo"
	"github.com/kilip/opus/server/internal/adapter/queue/sqlite"
	"github.com/kilip/opus/server/internal/shared/logger"
	"github.com/kilip/opus/server/internal/shared/queue"
	"github.com/kilip/opus/server/internal/testutil"
)

func TestSQLiteQueue_Integration(t *testing.T) {
	client := testutil.NewTestEntClient(t)
	db, err := entgo.DB(client)
	if err != nil {
		t.Fatalf("failed to get db: %v", err)
	}

	q, err := sqlite.NewSQLiteQueue(db, 2, &logger.NoopLogger{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	ctx := context.Background()

	jobID, err := q.Enqueue(ctx, "test:job", []byte("payload"))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	job, err := q.Inspect(ctx, jobID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if job.Status != queue.JobStatusPending {
		t.Errorf("expected status pending, got %v", job.Status)
	}

	handled := make(chan bool, 1)
	q.RegisterHandler("test:job", func(c context.Context, j *queue.Job) error {
		handled <- true
		return nil
	})

	if err := q.Start(ctx); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	select {
	case <-handled:
	case <-time.After(2 * time.Second):
		t.Errorf("timeout waiting for job to be handled")
	}

	var finalJob *queue.Job
	for i := 0; i < 50; i++ {
		finalJob, err = q.Inspect(ctx, jobID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if finalJob.Status == queue.JobStatusCompleted {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if finalJob.Status != queue.JobStatusCompleted {
		t.Errorf("expected status completed, got %v", finalJob.Status)
	}

	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("expected panic on RegisterHandler after Start")
			}
		}()
		q.RegisterHandler("late:job", func(c context.Context, j *queue.Job) error { return nil })
	}()

	if err := q.Shutdown(ctx); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}
