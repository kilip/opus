package domain

import (
	"encoding/json"
	"time"
)

// RunResponse is the HTTP API representation of a Run.
type RunResponse struct {
	ID            string          `json:"id"`
	TaskID        string          `json:"taskId"`
	ProjectID     string          `json:"projectId"`
	NodeID        *string         `json:"nodeId,omitempty"`
	SessionID     *string         `json:"sessionId,omitempty"`
	Mode          RunMode         `json:"mode"`
	State         RunState        `json:"state"`
	StageAttempts int             `json:"stageAttempts"`
	Branch        string          `json:"branch,omitempty"`
	PRNumber      *int            `json:"prNumber,omitempty"`
	Report        json.RawMessage `json:"report,omitempty" swaggertype:"object"`
	PausedUntil   *time.Time      `json:"pausedUntil,omitempty"`
	FailureReason *string         `json:"failureReason,omitempty"`
	CreatedAt     time.Time       `json:"createdAt"`
	StartedAt     *time.Time      `json:"startedAt,omitempty"`
	EndedAt       *time.Time      `json:"endedAt,omitempty"`
	UpdatedAt     time.Time       `json:"updatedAt"`
}

// CancelRunRequest is the optional request body for cancelling a Run.
type CancelRunRequest struct {
	Reason string `json:"reason,omitempty"`
}

// ToRunResponse maps a domain.Run to its API DTO.
func ToRunResponse(r Run) RunResponse {
	var nodeID *string
	if r.NodeID != nil {
		s := r.NodeID.String()
		nodeID = &s
	}
	var sessionID *string
	if r.SessionID != nil {
		s := r.SessionID.String()
		sessionID = &s
	}
	return RunResponse{
		ID:            r.ID.String(),
		TaskID:        r.TaskID.String(),
		ProjectID:     r.ProjectID.String(),
		NodeID:        nodeID,
		SessionID:     sessionID,
		Mode:          r.Mode,
		State:         r.State,
		StageAttempts: r.StageAttempts,
		Branch:        r.Branch,
		PRNumber:      r.PRNumber,
		Report:        r.Report,
		PausedUntil:   r.PausedUntil,
		FailureReason: r.FailureReason,
		CreatedAt:     r.CreatedAt,
		StartedAt:     r.StartedAt,
		EndedAt:       r.EndedAt,
		UpdatedAt:     r.UpdatedAt,
	}
}
