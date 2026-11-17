package domain

import (
	"time"

	"github.com/kilip/opus/internal/task"
)

// TrackerSync represents per-project tracker sync state.
type TrackerSync struct {
	ID         string
	ProjectID  string
	Cursor     string
	LastSyncAt *time.Time
	LastError  string
}

// TrackerStatusResponse is the response payload for GET /projects/{id}/tracker.
type TrackerStatusResponse struct {
	LastSyncAt *time.Time `json:"lastSyncAt"`
	LastError  string     `json:"lastError"`
}

// TrackerSettingsResponse is the response payload for GET/PUT /tracker/settings.
type TrackerSettingsResponse struct {
	StatusLabels map[string]string `json:"statusLabels" validate:"required"`
}

// UpdateTrackerSettingsRequest is the request payload for PUT /tracker/settings.
type UpdateTrackerSettingsRequest struct {
	StatusLabels map[string]string `json:"statusLabels" validate:"required"`
}

// EffectiveLabelMap contains resolved bidirectional status-label mappings for a project.
type EffectiveLabelMap struct {
	StatusToLabel map[string]string
	LabelToStatus map[string]task.TaskStatus
}

// PRState represents normalized pull request lifecycle state.
type PRState string

const (
	PRStateOpen   PRState = "open"
	PRStateClosed PRState = "closed"
	PRStateMerged PRState = "merged"
)

// ChecksSummary represents aggregated CI check status for a pull request.
type ChecksSummary string

const (
	ChecksPassing ChecksSummary = "passing"
	ChecksFailing ChecksSummary = "failing"
	ChecksPending ChecksSummary = "pending"
	ChecksNone    ChecksSummary = "none"
)

// TrackerPR is the tracked pull request snapshot exposed via API and domain.
type TrackerPR struct {
	ID            string        `json:"id" validate:"required"`
	ProjectID     string        `json:"projectId" validate:"required"`
	TaskID        *string       `json:"taskId,omitempty"`
	Number        int           `json:"number" validate:"required"`
	State         PRState       `json:"state" validate:"required"`
	IsDraft       bool          `json:"isDraft"`
	ReviewState   string        `json:"reviewState"`
	ChecksSummary ChecksSummary `json:"checksSummary" validate:"required"`
	URL           string        `json:"url"`
	CreatedAt     time.Time     `json:"createdAt" validate:"required"`
	UpdatedAt     time.Time     `json:"updatedAt" validate:"required"`
}
