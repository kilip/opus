package domain

import "github.com/google/uuid"

// Event bus topic names for session lifecycle facts.
const (
	EventQueued    = "session.queued"
	EventStarted   = "session.started"
	EventCompleted = "session.completed"
	EventFailed    = "session.failed"
	EventCancelled = "session.cancelled"
)

// SessionLifecyclePayload is published to the event bus when session state changes.
type SessionLifecyclePayload struct {
	SessionID uuid.UUID     `json:"sessionId"`
	TaskID    uuid.UUID     `json:"taskId"`
	NodeID    uuid.UUID     `json:"nodeId"`
	Status    SessionStatus `json:"status"`
}
