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
	// ReserveAttempt atomically increments attempts if attempts < max and returns the new count.
	ReserveAttempt(ctx context.Context, id uuid.UUID, max int) (attempts int, ok bool, err error)
	// ConsumeOTP atomically deletes the OTP by id and returns true iff this caller deleted it.
	ConsumeOTP(ctx context.Context, id uuid.UUID) (bool, error)
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

func (r *EntOTPRepo) ReserveAttempt(ctx context.Context, id uuid.UUID, max int) (int, bool, error) {
	n, err := r.db.OTP.Update().
		Where(entOTP.IDEQ(id), entOTP.AttemptsLT(max)).
		AddAttempts(1).
		Save(ctx)
	if err != nil {
		return 0, false, err
	}
	if n == 0 {
		return max, false, nil
	}
	o, err := r.db.OTP.Get(ctx, id)
	if err != nil {
		return max, false, nil
	}
	return o.Attempts, true, nil
}

func (r *EntOTPRepo) ConsumeOTP(ctx context.Context, id uuid.UUID) (bool, error) {
	n, err := r.db.OTP.Delete().Where(entOTP.IDEQ(id)).Exec(ctx)
	if err != nil {
		return false, err
	}
	return n == 1, nil
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
