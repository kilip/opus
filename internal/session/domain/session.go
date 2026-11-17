package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// SessionStatus represents the lifecycle state of an agent session.
type SessionStatus string

const (
	StatusQueued    SessionStatus = "queued"
	StatusRunning   SessionStatus = "running"
	StatusCompleted SessionStatus = "completed"
	StatusFailed    SessionStatus = "failed"
	StatusCancelled SessionStatus = "cancelled"
)

// IsValidStatus reports whether s is a known session status.
func IsValidStatus(s SessionStatus) bool {
	switch s {
	case StatusQueued, StatusRunning, StatusCompleted, StatusFailed, StatusCancelled:
		return true
	default:
		return false
	}
}

// IsTerminalStatus reports whether s is a terminal session status.
func IsTerminalStatus(s SessionStatus) bool {
	switch s {
	case StatusCompleted, StatusFailed, StatusCancelled:
		return true
	default:
		return false
	}
}

// Session represents an agent execution session bound to a task and node.
type Session struct {
	ID              uuid.UUID
	TaskID          uuid.UUID
	NodeID          uuid.UUID
	Provider        string
	ProviderAccount string
	Status          SessionStatus
	FailureReason   *string
	CreatedAt       time.Time
	StartedAt       *time.Time
	EndedAt         *time.Time
	UpdatedAt       time.Time
}

// SessionEvent represents an ordered event emitted by a Session.
type SessionEvent struct {
	ID        uuid.UUID
	SessionID uuid.UUID
	Seq       int
	Kind      string
	Payload   json.RawMessage
	At        time.Time
}

// CreateInput holds parameters for creating/enqueuing a session.
type CreateInput struct {
	TaskID          uuid.UUID
	NodeID          uuid.UUID
	Provider        string
	ProviderAccount string
}

// ListFilter filters sessions in List queries.
type ListFilter struct {
	TaskID *uuid.UUID
	NodeID *uuid.UUID
	Status *SessionStatus
}
