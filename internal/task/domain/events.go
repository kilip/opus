package domain

// Event type constants for task domain events.
const (
	EventReceived      = "task.received"
	EventStatusChanged = "task.status_changed"
)

// TaskReceivedPayload is the event payload published when a task is created.
type TaskReceivedPayload struct {
	ID         string     `json:"id"`
	Key        string     `json:"key"`
	ProjectID  string     `json:"projectId"`
	Source     TaskSource `json:"source"`
	ExternalID string     `json:"externalId,omitempty"`
	Status     TaskStatus `json:"status"`
}

// TaskStatusChangedPayload is the event payload published when a task stage transitions.
type TaskStatusChangedPayload struct {
	ID        string     `json:"id"`
	ProjectID string     `json:"projectId"`
	From      TaskStatus `json:"from"`
	To        TaskStatus `json:"to"`
}
