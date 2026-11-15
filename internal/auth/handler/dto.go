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
