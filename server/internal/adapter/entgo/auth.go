package entgo

import (
	"context"

	"github.com/kilip/opus/server/ent"
	"github.com/kilip/opus/server/ent/user"
	"github.com/kilip/opus/server/internal/auth"
)

// AuthRepo is the concrete Ent implementation of the auth Repository interface.
type AuthRepo struct {
	client *ent.Client
}

// NewAuthRepo creates a new instance of AuthRepo.
func NewAuthRepo(client *ent.Client) *AuthRepo {
	return &AuthRepo{client: client}
}

// FindByEmail retrieves a user by email from the Ent database.
func (r *AuthRepo) FindByEmail(ctx context.Context, email string) (*auth.User, error) {
	u, err := r.client.User.Query().Where(user.Email(email)).Only(ctx)
	if err != nil {
		return nil, err
	}
	return &auth.User{Email: u.Email}, nil
}
