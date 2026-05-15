package service

import (
	"context"

	"github.com/kilip/opus/api/internal/config"
	"github.com/kilip/opus/api/internal/model"
)

type UserReader interface {
	FindByID(ctx context.Context, id string) (*model.User, error)
	FindByEmail(ctx context.Context, email string) (*model.User, error)
}

type UserService struct {
	userRepo UserReader
	cfg      *config.Config
}

func NewUserService(userRepo UserReader, cfg *config.Config) *UserService {
	return &UserService{
		userRepo: userRepo,
		cfg:      cfg,
	}
}

func (s *UserService) GetUserByID(ctx context.Context, userID string) (*model.User, error) {
	return s.userRepo.FindByID(ctx, userID)
}

func (s *UserService) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	return s.userRepo.FindByEmail(ctx, email)
}
