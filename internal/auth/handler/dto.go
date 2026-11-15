package handler

import "github.com/google/uuid"

// MeResponse is the response body for GET /auth/me.
// swagger:model MeResponse
type MeResponse struct {
	// ID is the authenticated user's UUID.
	ID uuid.UUID `json:"id"`
	// Email is the authenticated user's email address.
	Email string `json:"email"`
}

// OTPRequestBody is the request body for POST /auth/otp/request.
type OTPRequestBody struct {
	Email string `json:"email"`
}

// OTPVerifyBody is the request body for POST /auth/otp/verify.
type OTPVerifyBody struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

// OTPVerifyResponse is the response body for POST /auth/otp/verify.
type OTPVerifyResponse struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email"`
}

// SetupStatusResponse is the response body for GET /setup/status.
type SetupStatusResponse struct {
	NeedsSetup bool `json:"needs_setup"`
}

// SetupOTPBody is the request body for POST /setup/otp.
type SetupOTPBody struct {
	Email string `json:"email"`
}

// SetupCompleteBody is the request body for POST /setup.
type SetupCompleteBody struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

// SetupUserResponse is the response body for POST /setup.
type SetupUserResponse struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email"`
}
