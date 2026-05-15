package service_test

import (
	"context"
	"testing"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"github.com/kilip/opus/api/internal/model"
	"github.com/kilip/opus/api/internal/service"
	"github.com/kilip/opus/api/mocks"
)

func TestUserService_GetUserByID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockUserRepository(ctrl)
	svc := service.NewUserService(mockRepo, nil) // Config not needed for this
	ctx := context.Background()

	u := &model.User{ID: "user_1", Email: "test@example.com"}
	mockRepo.EXPECT().FindByID(ctx, "user_1").Return(u, nil)

	result, err := svc.GetUserByID(ctx, "user_1")
	assert.NoError(t, err)
	assert.Equal(t, u, result)
}

func TestUserService_GetUserByEmail(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockUserRepository(ctrl)
	svc := service.NewUserService(mockRepo, nil)
	ctx := context.Background()

	u := &model.User{ID: "user_1", Email: "test@example.com"}
	mockRepo.EXPECT().FindByEmail(ctx, "test@example.com").Return(u, nil)

	result, err := svc.GetUserByEmail(ctx, "test@example.com")
	assert.NoError(t, err)
	assert.Equal(t, u, result)
}
