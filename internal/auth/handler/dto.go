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
