package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// SessionResponse is the API wire representation of a Session.
type SessionResponse struct {
	ID              uuid.UUID     `json:"id"`
	TaskID          uuid.UUID     `json:"taskId"`
	NodeID          uuid.UUID     `json:"nodeId"`
	Provider        string        `json:"provider"`
	ProviderAccount string        `json:"providerAccount"`
	Status          SessionStatus `json:"status"`
	FailureReason   *string       `json:"failureReason,omitempty"`
	CreatedAt       time.Time     `json:"createdAt"`
	StartedAt       *time.Time    `json:"startedAt,omitempty"`
	EndedAt         *time.Time    `json:"endedAt,omitempty"`
	UpdatedAt       time.Time     `json:"updatedAt"`
}

// SessionEventResponse is the API/SSE wire representation of a SessionEvent.
type SessionEventResponse struct {
	ID        uuid.UUID       `json:"id"`
	SessionID uuid.UUID       `json:"sessionId"`
	Seq       int             `json:"seq"`
	Kind      string          `json:"kind"`
	Payload   json.RawMessage `json:"payload,omitempty" swaggertype:"object"`
	At        time.Time       `json:"at"`
}

// ToSessionResponse converts a domain.Session to SessionResponse.
func ToSessionResponse(s Session) SessionResponse {
	return SessionResponse(s)
}

// ToSessionEventResponse converts a domain.SessionEvent to SessionEventResponse.
func ToSessionEventResponse(e SessionEvent) SessionEventResponse {
	return SessionEventResponse(e)
}
