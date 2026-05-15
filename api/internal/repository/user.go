package repository

import (
	"context"
	"fmt"

	"github.com/kilip/opus/api/ent"
	"github.com/kilip/opus/api/ent/user"
	"github.com/kilip/opus/api/internal/model"
)

type UserRepository interface {
	FindByID(ctx context.Context, id string) (*model.User, error)
	FindByEmail(ctx context.Context, email string) (*model.User, error)
	FindByProviderID(ctx context.Context, provider, providerID string) (*model.User, error)
	Create(ctx context.Context, u *model.User) (*model.User, error)
	Update(ctx context.Context, u *model.User) (*model.User, error)
}

type userRepository struct {
	client *ent.Client
}

func NewUserRepository(client *ent.Client) UserRepository {
	return &userRepository{client: client}
}

func (r *userRepository) FindByID(ctx context.Context, id string) (*model.User, error) {
	entUser, err := r.client.User.Query().Where(user.ID(id)).Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, model.ErrUserNotFound
		}
		return nil, fmt.Errorf("userRepository.FindByID: %w", err)
	}
	return r.mapToModel(entUser), nil
}

func (r *userRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	entUser, err := r.client.User.Query().Where(user.Email(email)).Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, model.ErrUserNotFound
		}
		return nil, fmt.Errorf("userRepository.FindByEmail: %w", err)
	}
	return r.mapToModel(entUser), nil
}

func (r *userRepository) FindByProviderID(ctx context.Context, providerName, providerID string) (*model.User, error) {
	entUser, err := r.client.User.Query().
		Where(user.Provider(providerName), user.ProviderID(providerID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, model.ErrUserNotFound
		}
		return nil, fmt.Errorf("userRepository.FindByProviderID: %w", err)
	}
	return r.mapToModel(entUser), nil
}

func (r *userRepository) Create(ctx context.Context, u *model.User) (*model.User, error) {
	entUser, err := r.client.User.Create().
		SetID(u.ID).
		SetEmail(u.Email).
		SetName(u.Name).
		SetAvatarURL(u.AvatarURL).
		SetProvider(u.Provider).
		SetProviderID(u.ProviderID).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("userRepository.Create: %w", err)
	}
	return r.mapToModel(entUser), nil
}

func (r *userRepository) Update(ctx context.Context, u *model.User) (*model.User, error) {
	entUser, err := r.client.User.UpdateOneID(u.ID).
		SetEmail(u.Email).
		SetName(u.Name).
		SetAvatarURL(u.AvatarURL).
		SetProvider(u.Provider).
		SetProviderID(u.ProviderID).
		Save(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, model.ErrUserNotFound
		}
		return nil, fmt.Errorf("userRepository.Update: %w", err)
	}
	return r.mapToModel(entUser), nil
}

func (r *userRepository) mapToModel(u *ent.User) *model.User {
	return &model.User{
		ID:         u.ID,
		Email:      u.Email,
		Name:       u.Name,
		AvatarURL:  u.AvatarURL,
		Provider:   u.Provider,
		ProviderID: u.ProviderID,
		CreatedAt:  u.CreatedAt,
		UpdatedAt:  u.UpdatedAt,
	}
}
