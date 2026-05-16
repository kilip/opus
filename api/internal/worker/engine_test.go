package worker

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"github.com/kilip/opus/api/internal/model"
	"github.com/kilip/opus/api/mocks"
)

func TestWorkerEngine_Process(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mDriver := mocks.NewMockQueueDriver(ctrl)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	engine := NewWorkerEngine(mDriver, 1, logger)

	ctx := context.Background()
	job := &model.Job{
		ID:         "job-1",
		Type:       "test_job",
		MaxRetries: 3,
		UserID:     "user-1",
	}

	t.Run("Success Path", func(t *testing.T) {
		handlerCalled := false
		engine.Register("test_job", func(ctx context.Context, j *model.Job) error {
			handlerCalled = true
			return nil
		})

		mDriver.EXPECT().UpdateStatus(gomock.Any(), "job-1", model.StatusCompleted, "").Return(nil)

		engine.(*workerEngine).process(ctx, job)
		assert.True(t, handlerCalled)
	})

	t.Run("Handler Fails - Reschedule", func(t *testing.T) {
		job.Retries = 0
		engine.Register("fail_job", func(ctx context.Context, j *model.Job) error {
			return errors.New("boom")
		})
		job.Type = "fail_job"

		mDriver.EXPECT().Push(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, j *model.Job) error {
			assert.Equal(t, 1, j.Retries)
			assert.Equal(t, "boom", j.Error)
			assert.Equal(t, model.StatusPending, j.Status)
			return nil
		}).Return(nil)

		engine.(*workerEngine).process(ctx, job)
	})

	t.Run("Max Retries Reached - Move to Dead", func(t *testing.T) {
		job.Retries = 2 // Will become 3
		job.MaxRetries = 3
		job.Type = "fail_job"

		mDriver.EXPECT().MoveToDead(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, j *model.Job) error {
			assert.Equal(t, 3, j.Retries)
			return nil
		}).Return(nil)

		engine.(*workerEngine).process(ctx, job)
	})
}

func TestWorkerEngine_StartStop(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mDriver := mocks.NewMockQueueDriver(ctrl)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	engine := NewWorkerEngine(mDriver, 2, logger)

	mDriver.EXPECT().Pop(gomock.Any()).Return(nil, nil).AnyTimes()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := engine.Start(ctx)
	assert.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	err = engine.Stop()
	assert.NoError(t, err)
}
