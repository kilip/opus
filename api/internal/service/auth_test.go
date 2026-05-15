package service_test

import (
	"context"
	"testing"

	"github.com/kilip/opus/api/internal/config"
	"github.com/kilip/opus/api/internal/model"
	"github.com/kilip/opus/api/internal/service"
	"github.com/kilip/opus/api/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestAuthService_IssueTokens(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mocks.NewMockUserRepository(ctrl)
	sessionRepo := mocks.NewMockSessionRepository(ctrl)
	cfg := &config.Config{
		Auth: config.AuthConfig{
			Secret:         "testsecret",
			AccessTokenTTL: 15,
		},
	}

	svc := service.NewAuthService(userRepo, sessionRepo, cfg)

	sessionRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(&model.Session{}, nil)

	accessToken, refreshToken, err := svc.IssueTokens(context.Background(), "user_1")

	assert.NoError(t, err)
	assert.NotEmpty(t, accessToken)
	assert.NotEmpty(t, refreshToken)
}

func TestAuthService_ValidateAccessToken(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mocks.NewMockUserRepository(ctrl)
	sessionRepo := mocks.NewMockSessionRepository(ctrl)
	cfg := &config.Config{
		Auth: config.AuthConfig{
			Secret:         "testsecret",
			AccessTokenTTL: 15,
		},
	}

	svc := service.NewAuthService(userRepo, sessionRepo, cfg)
	sessionRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(&model.Session{}, nil)

	accessToken, _, _ := svc.IssueTokens(context.Background(), "user_1")

	userID, err := svc.ValidateAccessToken(accessToken)

	assert.NoError(t, err)
	assert.Equal(t, "user_1", userID)
}
