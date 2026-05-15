package service_test

import (
	"context"
	"testing"

	"github.com/kilip/opus/api/internal/config"
	"github.com/kilip/opus/api/internal/model"
	"github.com/kilip/opus/api/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockUserRepo struct {
	mock.Mock
}

func (m *MockUserRepo) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserRepo) FindByProviderID(ctx context.Context, provider, providerID string) (*model.User, error) {
	args := m.Called(ctx, provider, providerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserRepo) Create(ctx context.Context, user *model.User) (*model.User, error) {
	args := m.Called(ctx, user)
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserRepo) Update(ctx context.Context, user *model.User) (*model.User, error) {
	args := m.Called(ctx, user)
	return args.Get(0).(*model.User), args.Error(1)
}

type MockSessionRepo struct {
	mock.Mock
}

func (m *MockSessionRepo) Create(ctx context.Context, session *model.Session) (*model.Session, error) {
	args := m.Called(ctx, session)
	return args.Get(0).(*model.Session), args.Error(1)
}

func (m *MockSessionRepo) FindByTokenHash(ctx context.Context, hash string) (*model.Session, error) {
	args := m.Called(ctx, hash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Session), args.Error(1)
}

func (m *MockSessionRepo) RevokeByID(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockSessionRepo) RevokeAllByUserID(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func TestAuthService_IssueTokens(t *testing.T) {
	userRepo := new(MockUserRepo)
	sessionRepo := new(MockSessionRepo)
	cfg := &config.Config{
		Auth: config.AuthConfig{
			Secret:         "testsecret",
			AccessTokenTTL: 15,
		},
	}

	svc := service.NewAuthService(userRepo, sessionRepo, cfg)

	sessionRepo.On("Create", mock.Anything, mock.Anything).Return(&model.Session{}, nil)

	accessToken, refreshToken, err := svc.IssueTokens(context.Background(), "user_1")

	assert.NoError(t, err)
	assert.NotEmpty(t, accessToken)
	assert.NotEmpty(t, refreshToken)
	sessionRepo.AssertExpectations(t)
}

func TestAuthService_ValidateAccessToken(t *testing.T) {
	userRepo := new(MockUserRepo)
	sessionRepo := new(MockSessionRepo)
	cfg := &config.Config{
		Auth: config.AuthConfig{
			Secret:         "testsecret",
			AccessTokenTTL: 15,
		},
	}

	svc := service.NewAuthService(userRepo, sessionRepo, cfg)
	sessionRepo.On("Create", mock.Anything, mock.Anything).Return(&model.Session{}, nil)

	accessToken, _, _ := svc.IssueTokens(context.Background(), "user_1")

	userID, err := svc.ValidateAccessToken(accessToken)

	assert.NoError(t, err)
	assert.Equal(t, "user_1", userID)
}
