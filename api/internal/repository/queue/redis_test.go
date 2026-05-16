package queue

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/kilip/opus/api/internal/model"
)

func TestRedisDriver(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Close()

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	driver := NewRedisDriver(client, "test")
	ctx := context.Background()

	t.Run("Push and Pop", func(t *testing.T) {
		job := &model.Job{
			ID:          "job-1",
			Type:        "test",
			ScheduledAt: time.Now().Add(-1 * time.Minute),
		}

		err := driver.Push(ctx, job)
		assert.NoError(t, err)

		popped, err := driver.Pop(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, popped)
		assert.Equal(t, "job-1", popped.ID)

		// Pop again should be nil
		popped, _ = driver.Pop(ctx)
		assert.Nil(t, popped)
	})

	t.Run("Cron Upsert and List", func(t *testing.T) {
		cron := &model.CronSchedule{
			ID:       "cron-1",
			Name:     "daily",
			CronExpr: "0 0 * * *",
			IsActive: true,
		}

		err := driver.UpsertCron(ctx, cron)
		assert.NoError(t, err)

		crons, err := driver.ListPendingCrons(ctx)
		assert.NoError(t, err)
		assert.Len(t, crons, 1)
		assert.Equal(t, "daily", crons[0].Name)
	})

	t.Run("Dead Letter and Multi-User Isolation", func(t *testing.T) {
		jobID := "job-dl"
		m := &model.Job{
			ID:      jobID,
			Type:    "test",
			UserID:  "user-1",
			Payload: []byte("{}"),
		}

		// Move to dead
		err := driver.MoveToDead(ctx, m)
		assert.NoError(t, err)

		// List for user-1
		dl, err := driver.ListDeadLetters(ctx, "user-1", 10, 0)
		assert.NoError(t, err)
		assert.Len(t, dl, 1)
		assert.Equal(t, jobID, dl[0].JobID)

		// List for user-2
		dl, err = driver.ListDeadLetters(ctx, "user-2", 10, 0)
		assert.NoError(t, err)
		assert.Len(t, dl, 0)

		// Retry with wrong user
		err = driver.RetryDeadLetter(ctx, "user-2", "dl-"+jobID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unauthorized")

		// Retry with correct user
		err = driver.RetryDeadLetter(ctx, "user-1", "dl-"+jobID)
		assert.NoError(t, err)

		// Verify popped
		popped, _ := driver.Pop(ctx)
		assert.NotNil(t, popped)
		assert.Equal(t, jobID, popped.ID)

		// Move back to dead then delete
		_ = driver.MoveToDead(ctx, m)
		err = driver.DeleteDeadLetter(ctx, "user-2", "dl-"+jobID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unauthorized")

		err = driver.DeleteDeadLetter(ctx, "user-1", "dl-"+jobID)
		assert.NoError(t, err)

		dl, _ = driver.ListDeadLetters(ctx, "user-1", 10, 0)
		assert.Len(t, dl, 0)
	})
}
