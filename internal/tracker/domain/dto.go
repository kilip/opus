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
