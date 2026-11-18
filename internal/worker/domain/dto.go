package domain

import (
	"encoding/json"

	"github.com/kilip/opus/internal/node"
)

// Command type constants carried inside command envelope payloads.
const (
	CommandStartRun        = "start_run"
	CommandCancelRun       = "cancel_run"
	CommandCleanupWorktree = "cleanup_worktree"
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

// Policy defines the execution, tool, and question constraints for a run.
type Policy struct {
	WorktreeRoot      string   `json:"worktreeRoot"`
	AllowedSubagents  []string `json:"allowedSubagents,omitempty"`
	AllowedMCPTools   []string `json:"allowedMcpTools,omitempty"`
	QuestionAutoReply string   `json:"questionAutoReply,omitempty"`
	MaxQuestions      int      `json:"maxQuestions"`
	AllowMerge        bool     `json:"allowMerge"`
}

// StartRun is the command payload for starting a run on a worker.
type StartRun struct {
	Type          string  `json:"type,omitempty"`
	RunID         string  `json:"runId"`
	SessionID     string  `json:"sessionId,omitempty"`
	TaskID        string  `json:"taskId,omitempty"`
	TaskKey       string  `json:"taskKey,omitempty"`
	ProjectID     string  `json:"projectId,omitempty"`
	ProjectSlug   string  `json:"projectSlug,omitempty"`
	RepoURL       string  `json:"repoUrl,omitempty"`
	DefaultBranch string  `json:"defaultBranch,omitempty"`
	MappedPath    string  `json:"mappedPath,omitempty"`
	Branch        string  `json:"branch,omitempty"`
	Prompt        string  `json:"prompt,omitempty"`
	Policy        *Policy `json:"policy,omitempty"`
}

// CancelRun is the command payload for cancelling an active run on a worker.
type CancelRun struct {
	Type   string `json:"type,omitempty"`
	RunID  string `json:"runId"`
	Reason string `json:"reason,omitempty"`
}

// CleanupWorktree is the command payload for removing a task's worktree and branch on a worker.
type CleanupWorktree struct {
	Type        string `json:"type,omitempty"`
	ProjectSlug string `json:"projectSlug"`
	TaskKey     string `json:"taskKey"`
	Branch      string `json:"branch,omitempty"`
	MappedPath  string `json:"mappedPath,omitempty"`
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

// NewCleanupWorktreeEnvelope encodes a CleanupWorktree command into a node.Envelope.
func NewCleanupWorktreeEnvelope(id string, cmd CleanupWorktree) (node.Envelope, error) {
	cmd.Type = CommandCleanupWorktree
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
