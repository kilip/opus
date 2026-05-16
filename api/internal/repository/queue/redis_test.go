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
}
