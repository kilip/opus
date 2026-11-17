package domain

import "github.com/google/uuid"

// Event bus topic names for run lifecycle facts.
const (
	EventStarted  = "run.started"
	EventFinished = "run.finished"
)

// RunStartedPayload is published to the event bus when a run starts executing.
type RunStartedPayload struct {
	RunID     uuid.UUID `json:"runId"`
	TaskID    uuid.UUID `json:"taskId"`
	ProjectID uuid.UUID `json:"projectId"`
	Mode      RunMode   `json:"mode"`
}

// RunFinishedPayload is published to the event bus when a run reaches a finished outcome.
type RunFinishedPayload struct {
	RunID   uuid.UUID `json:"runId"`
	TaskID  uuid.UUID `json:"taskId"`
	Mode    RunMode   `json:"mode"`
	Outcome RunState  `json:"outcome"`
	Reason  string    `json:"reason,omitempty"`
}
