package handler

import "time"

// TaskSource identifies the tracker a task originates from. Tasks are issues
// normalised from external systems, or created in Opus itself ("custom").
type TaskSource string

const (
	TaskSourceGitHub TaskSource = "github"
	TaskSourceJira   TaskSource = "jira"
	TaskSourceSentry TaskSource = "sentry"
	TaskSourceCustom TaskSource = "custom"
)

// TaskStatus is the normalised workflow stage. The status reported by the
// source tracker is kept verbatim in Task.SourceStatus.
type TaskStatus string

const (
	TaskStatusBacklog    TaskStatus = "backlog"
	TaskStatusSpec       TaskStatus = "spec"
	TaskStatusInProgress TaskStatus = "inProgress"
	TaskStatusInReview   TaskStatus = "inReview"
	TaskStatusDone       TaskStatus = "done"
)

// TaskPriority is the normalised priority, lowest first.
type TaskPriority string

const (
	TaskPriorityNone   TaskPriority = "none"
	TaskPriorityLow    TaskPriority = "low"
	TaskPriorityMedium TaskPriority = "medium"
	TaskPriorityHigh   TaskPriority = "high"
)

// TaskType is the normalised kind of work.
type TaskType string

const (
	TaskTypeFeature  TaskType = "feature"
	TaskTypeBug      TaskType = "bug"
	TaskTypeChore    TaskType = "chore"
	TaskTypeIncident TaskType = "incident"
)

// ActivityKind distinguishes the entries of a task's chatter timeline.
type ActivityKind string

const (
	ActivityKindMessage  ActivityKind = "message"
	ActivityKindNote     ActivityKind = "note"
	ActivityKindTracking ActivityKind = "tracking"
)

// Assignee is the person a task is assigned to.
type Assignee struct {
	ID   string `json:"id" validate:"required"`
	Name string `json:"name" validate:"required"`
}

// Task is an issue normalised from its source tracker.
type Task struct {
	// ID is the Opus identifier.
	ID string `json:"id" validate:"required"`
	// Key is the human-readable reference, e.g. OPUS-12.
	Key    string     `json:"key" validate:"required"`
	Source TaskSource `json:"source" validate:"required"`
	// ExternalID is the identifier in the source tracker.
	ExternalID string `json:"externalId,omitempty"`
	// ExternalURL links to the issue in the source tracker.
	ExternalURL string     `json:"externalUrl,omitempty"`
	Title       string     `json:"title" validate:"required"`
	Description string     `json:"description" validate:"required"`
	Status      TaskStatus `json:"status" validate:"required"`
	// SourceStatus is the status name as reported by the source tracker.
	SourceStatus string       `json:"sourceStatus,omitempty"`
	Priority     TaskPriority `json:"priority" validate:"required"`
	Type         TaskType     `json:"type" validate:"required"`
	Assignee     *Assignee    `json:"assignee,omitempty"`
	// DueDate is a calendar date (YYYY-MM-DD).
	DueDate   string    `json:"dueDate,omitempty" format:"date"`
	Labels    []string  `json:"labels" validate:"required"`
	CreatedAt time.Time `json:"createdAt" validate:"required"`
	UpdatedAt time.Time `json:"updatedAt" validate:"required"`
}

// TaskList is the response of the task collection endpoint.
type TaskList struct {
	Items []Task `json:"items" validate:"required"`
	Total int    `json:"total" validate:"required"`
}

// Activity is one entry of a task's chatter timeline.
type Activity struct {
	ID        string       `json:"id" validate:"required"`
	Kind      ActivityKind `json:"kind" validate:"required"`
	Author    string       `json:"author" validate:"required"`
	Body      string       `json:"body" validate:"required"`
	CreatedAt time.Time    `json:"createdAt" validate:"required"`
}

// ActivityList is the response of the task activity endpoint, oldest first.
type ActivityList struct {
	Items []Activity `json:"items" validate:"required"`
}

// ErrorResponse is the body of every non-2xx response.
type ErrorResponse struct {
	Message string `json:"message" validate:"required"`
}
