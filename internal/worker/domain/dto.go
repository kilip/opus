package domain

import (
	"encoding/json"

	"github.com/kilip/opus/internal/node"
)

// Command type constants carried inside command envelope payloads.
const (
	CommandStartRun  = "start_run"
	CommandCancelRun = "cancel_run"
)

// Event type constants carried inside worker event envelope payloads.
const (
	EventRunCancelled = "run_cancelled"
	EventRunError     = "run_error"
	EventCapabilities = "capabilities_updated"
)

// Re-export node capability types for worker domain consumers.
type (
	ToolStatus   = node.ToolStatus
	Capabilities = node.Capabilities
	HelloPayload = node.HelloPayload
)

// StartRun is the command payload for starting a run on a worker.
type StartRun struct {
	Type      string `json:"type,omitempty"`
	RunID     string `json:"runId"`
	TaskID    string `json:"taskId,omitempty"`
	ProjectID string `json:"projectId,omitempty"`
	RepoURL   string `json:"repoUrl,omitempty"`
	Branch    string `json:"branch,omitempty"`
	Prompt    string `json:"prompt,omitempty"`
}

// CancelRun is the command payload for cancelling an active run on a worker.
type CancelRun struct {
	Type   string `json:"type,omitempty"`
	RunID  string `json:"runId"`
	Reason string `json:"reason,omitempty"`
}

// Event is the payload for worker-to-master event envelopes.
type Event struct {
	Type         string        `json:"type"`
	RunID        string        `json:"runId,omitempty"`
	Reason       string        `json:"reason,omitempty"`
	Error        string        `json:"error,omitempty"`
	Capabilities *Capabilities `json:"capabilities,omitempty"`
}

// NewStartRunEnvelope encodes a StartRun command into a node.Envelope.
func NewStartRunEnvelope(id string, cmd StartRun) (node.Envelope, error) {
	cmd.Type = CommandStartRun
	raw, err := json.Marshal(cmd)
	if err != nil {
		return node.Envelope{}, err
	}
	return node.Envelope{
		Type:    node.EnvelopeTypeCommand,
		ID:      id,
		Payload: raw,
	}, nil
}

// NewCancelRunEnvelope encodes a CancelRun command into a node.Envelope.
func NewCancelRunEnvelope(id string, cmd CancelRun) (node.Envelope, error) {
	cmd.Type = CommandCancelRun
	raw, err := json.Marshal(cmd)
	if err != nil {
		return node.Envelope{}, err
	}
	return node.Envelope{
		Type:    node.EnvelopeTypeCommand,
		ID:      id,
		Payload: raw,
	}, nil
}
