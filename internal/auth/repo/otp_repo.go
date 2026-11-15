package repo

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/kilip/opus/ent"
	entOTP "github.com/kilip/opus/ent/otp"
	entUser "github.com/kilip/opus/ent/user"
	"github.com/kilip/opus/internal/auth/domain"
)

// OTPRepo is the OTP repository interface.
type OTPRepo interface {
	// CreateOTP deletes any existing OTP for email, then creates a new one.
	CreateOTP(ctx context.Context, email, codeHash string, expiresAt time.Time) (*domain.OTP, error)
	// FindByEmail returns the active OTP for email.
	FindByEmail(ctx context.Context, email string) (*domain.OTP, error)
	// DeleteByEmail removes the OTP for email (single-use on verify).
	DeleteByEmail(ctx context.Context, email string) error
	// IncrementAttempts adds 1 to attempts; returns updated attempts count.
	IncrementAttempts(ctx context.Context, id uuid.UUID) (int, error)
	// FindUserByEmail returns a domain.User if owner email matches.
	FindUserByEmail(ctx context.Context, email string) (*domain.User, error)
}

// EntOTPRepo implements OTPRepo using an Ent client.
type EntOTPRepo struct {
	db *ent.Client
}

// NewOTPRepo returns an EntOTPRepo backed by db.
func NewOTPRepo(db *ent.Client) *EntOTPRepo {
	return &EntOTPRepo{db: db}
}

func (r *EntOTPRepo) CreateOTP(ctx context.Context, email, codeHash string, expiresAt time.Time) (*domain.OTP, error) {
	// Delete any existing OTP for this email first (invalidate old ones).
	_, _ = r.db.OTP.Delete().Where(entOTP.EmailEQ(email)).Exec(ctx)

	o, err := r.db.OTP.Create().
		SetEmail(email).
		SetCodeHash(codeHash).
		SetExpiresAt(expiresAt).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return toOTPDomain(o), nil
}

func (r *EntOTPRepo) FindByEmail(ctx context.Context, email string) (*domain.OTP, error) {
	o, err := r.db.OTP.Query().Where(entOTP.EmailEQ(email)).Only(ctx)
	if err != nil {
		return nil, err
	}
	return toOTPDomain(o), nil
}

func (r *EntOTPRepo) DeleteByEmail(ctx context.Context, email string) error {
	_, err := r.db.OTP.Delete().Where(entOTP.EmailEQ(email)).Exec(ctx)
	return err
}

func (r *EntOTPRepo) IncrementAttempts(ctx context.Context, id uuid.UUID) (int, error) {
	o, err := r.db.OTP.UpdateOneID(id).AddAttempts(1).Save(ctx)
	if err != nil {
		return 0, err
	}
	return o.Attempts, nil
}

func (r *EntOTPRepo) FindUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	u, err := r.db.User.Query().Where(entUser.EmailEQ(email)).Only(ctx)
	if err != nil {
		return nil, err
	}
	return &domain.User{ID: u.ID, Email: u.Email}, nil
}

func toOTPDomain(o *ent.OTP) *domain.OTP {
	return &domain.OTP{
		ID:        o.ID,
		Email:     o.Email,
		CodeHash:  o.CodeHash,
		ExpiresAt: o.ExpiresAt,
		Attempts:  o.Attempts,
		CreatedAt: o.CreatedAt,
	}
}
