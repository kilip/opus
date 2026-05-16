//go:build integration

package queue

import (
	"context"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/kilip/opus/api/ent"
	"github.com/kilip/opus/api/internal/model"
)

func TestEntGoDriver_Integration(t *testing.T) {
	ctx := context.Background()
	
	// Setup in-memory SQLite
	client, err := ent.Open("sqlite3", "file::memory:?cache=shared&_fk=1")
	if err != nil {
		t.Fatalf("failed opening connection to sqlite: %v", err)
	}
	defer client.Close()

	// Run migration
	if err := client.Schema.Create(ctx); err != nil {
		t.Fatalf("failed creating schema resources: %v", err)
	}

	driver := NewEntGoDriver(client, "sqlite")

	t.Run("Push and Pop Job", func(t *testing.T) {
		jobID := "job-1"
		m := &model.Job{
			ID:          jobID,
			Type:        "test_task",
			Payload:     []byte(`{"foo":"bar"}`),
			Priority:    10,
			Status:      model.StatusPending,
			ScheduledAt: time.Now().Add(-1 * time.Minute),
		}

		err := driver.Push(ctx, m)
		assert.NoError(t, err)

		// Pop it
		popped, err := driver.Pop(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, popped)
		assert.Equal(t, jobID, popped.ID)
		assert.Equal(t, "test_task", popped.Type)
		assert.Equal(t, 10, popped.Priority)
	})

	t.Run("Update Job Status", func(t *testing.T) {
		jobID := "job-2"
		m := &model.Job{
			ID:          jobID,
			Type:        "test_task",
			Payload:     []byte(`{}`),
			Priority:    5,
			Status:      model.StatusPending,
			ScheduledAt: time.Now(),
		}

		_ = driver.Push(ctx, m)

		err := driver.UpdateStatus(ctx, jobID, model.StatusRunning, "")
		assert.NoError(t, err)

		// Verify status
		popped, _ := driver.Pop(ctx)
		assert.Nil(t, popped, "should not pop running job")
		
		j, _ := client.Job.Get(ctx, jobID)
		assert.Equal(t, "running", j.Status)
	})

	t.Run("Move to Dead Letter", func(t *testing.T) {
		jobID := "job-3"
		m := &model.Job{
			ID:          jobID,
			Type:        "test_task",
			Payload:     []byte(`{"fail":true}`),
			Priority:    1,
			Status:      model.StatusFailed,
			Retries:     3,
			Error:       "max retries exceeded",
			ScheduledAt: time.Now(),
		}

		// Push first so we have a job to "move"
		_ = driver.Push(ctx, m)

		err := driver.MoveToDead(ctx, m)
		assert.NoError(t, err)

		// Check dead letter entry
		dl, err := client.DeadLetter.Get(ctx, jobID)
		assert.NoError(t, err)
		assert.Equal(t, jobID, dl.JobID)
		assert.Equal(t, "max retries exceeded", dl.LastError)

		// Check original job is gone
		_, err = client.Job.Get(ctx, jobID)
		assert.True(t, ent.IsNotFound(err))
	})

	t.Run("Cron Upsert and List", func(t *testing.T) {
		cronID := "cron-1"
		m := &model.CronSchedule{
			ID:       cronID,
			Name:     "daily_cleanup",
			CronExpr: "0 0 * * *",
			JobType:  "cleanup",
			IsActive: true,
		}

		err := driver.UpsertCron(ctx, m)
		assert.NoError(t, err)

		crons, err := driver.ListPendingCrons(ctx)
		assert.NoError(t, err)
		assert.Len(t, crons, 1)
		assert.Equal(t, cronID, crons[0].ID)
		assert.Equal(t, "daily_cleanup", crons[0].Name)
	})
}
