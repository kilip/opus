package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidOTP      = errors.New("INVALID_OTP")
	ErrOTPExpired      = errors.New("OTP_EXPIRED")
	ErrTooManyAttempts = errors.New("TOO_MANY_ATTEMPTS")
	ErrRateLimited     = errors.New("RATE_LIMITED")
)

// MaxOTPAttempts is the number of wrong-code attempts before the OTP is invalidated.
const MaxOTPAttempts = 5

// OTP represents a pending one-time password.
type OTP struct {
	ID        uuid.UUID
	Email     string
	CodeHash  string
	ExpiresAt time.Time
	Attempts  int
	CreatedAt time.Time
}
