package worker

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"github.com/kilip/opus/api/internal/model"
	"github.com/kilip/opus/api/mocks"
)

func TestCronScheduler_FireCron(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mDriver := mocks.NewMockQueueDriver(ctrl)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	sched := NewScheduler(mDriver, logger)

	ctx := context.Background()
	cronSched := &model.CronSchedule{
		ID:       "cron-1",
		Name:     "test_cron",
		CronExpr: "*/5 * * * *",
		JobType:  "scheduled_task",
		IsActive: true,
		UserID:   "user-1",
	}

	t.Run("Fire and Reschedule", func(t *testing.T) {
		mDriver.EXPECT().Push(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, j *model.Job) error {
			assert.Equal(t, "scheduled_task", j.Type)
			assert.Equal(t, model.StatusPending, j.Status)
			assert.Equal(t, "user-1", j.UserID)
			return nil
		})

		mDriver.EXPECT().UpdateCronNextRun(gomock.Any(), "cron-1", gomock.Any(), gomock.Any()).Return(nil)

		sched.(*cronScheduler).fireCron(ctx, cronSched)
	})
}

func TestCronScheduler_AddCron(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mDriver := mocks.NewMockQueueDriver(ctrl)
	sched := NewScheduler(mDriver, slog.Default())

	ctx := context.Background()

	t.Run("Invalid Expression", func(t *testing.T) {
		err := sched.AddCron(ctx, &model.CronSchedule{
			CronExpr: "invalid",
		})
		assert.Error(t, err)
	})

	t.Run("Valid Expression", func(t *testing.T) {
		mDriver.EXPECT().UpsertCron(ctx, gomock.Any()).Return(nil)
		err := sched.AddCron(ctx, &model.CronSchedule{
			Name:     "daily",
			CronExpr: "0 0 * * *",
			JobType:  "clean",
		})
		assert.NoError(t, err)
	})
}
