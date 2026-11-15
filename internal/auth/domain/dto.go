package domain

import "github.com/google/uuid"

// UserResponse is the response body for GET /auth/me, POST /auth/otp/verify
// and POST /setup.
type UserResponse struct {
	// ID is the authenticated user's UUID.
	ID uuid.UUID `json:"id"`
	// Email is the authenticated user's email address.
	Email string `json:"email"`
}

// OTPRequestBody is the request body for POST /auth/otp/request and
// POST /setup/otp.
type OTPRequestBody struct {
	Email string `json:"email"`
}

// OTPVerifyBody is the request body for POST /auth/otp/verify and POST /setup.
type OTPVerifyBody struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

// SetupStatusResponse is the response body for GET /setup/status.
type SetupStatusResponse struct {
	NeedsSetup bool `json:"needs_setup"`
}

// ProviderItem describes an enabled OAuth identity provider.
type ProviderItem struct {
	Name  string `json:"name"`
	Label string `json:"label"`
}
