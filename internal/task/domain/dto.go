package domain

import "time"

// CreateTaskRequest is payload for creating task.
type CreateTaskRequest struct {
	ProjectID   string        `json:"projectId" validate:"required"`
	Title       string        `json:"title" validate:"required"`
	Description string        `json:"description,omitempty"`
	Status      *TaskStatus   `json:"status,omitempty"`
	Priority    *TaskPriority `json:"priority,omitempty"`
	Type        *TaskType     `json:"type,omitempty"`
	Assignee    *Assignee     `json:"assignee,omitempty"`
	DueDate     string        `json:"dueDate,omitempty" format:"date"`
	Labels      []string      `json:"labels,omitempty"`
}

// UpdateTaskRequest is payload for updating task.
type UpdateTaskRequest struct {
	Title         *string       `json:"title,omitempty"`
	Description   *string       `json:"description,omitempty"`
	Status        *TaskStatus   `json:"status,omitempty"`
	Priority      *TaskPriority `json:"priority,omitempty"`
	Type          *TaskType     `json:"type,omitempty"`
	Assignee      *Assignee     `json:"assignee,omitempty"`
	ClearAssignee bool          `json:"clearAssignee,omitempty"`
	DueDate       *string       `json:"dueDate,omitempty" format:"date"`
	Labels        *[]string     `json:"labels,omitempty"`
}

// CreateActivityRequest is payload for adding activity to task.
type CreateActivityRequest struct {
	Kind *ActivityKind `json:"kind,omitempty"`
	Body string        `json:"body" validate:"required"`
}

// ImportTaskRequest is payload for upserting an external task via the task facade.
type ImportTaskRequest struct {
	ProjectID    string       `json:"projectId"`
	Title        string       `json:"title"`
	Description  string       `json:"description,omitempty"`
	Source       TaskSource   `json:"source"`
	ExternalID   string       `json:"externalId"`
	ExternalURL  string       `json:"externalUrl,omitempty"`
	SourceStatus string       `json:"sourceStatus,omitempty"`
	Key          string       `json:"key,omitempty"`
	Labels       []string     `json:"labels,omitempty"`
	Type         TaskType     `json:"type,omitempty"`
	Priority     TaskPriority `json:"priority,omitempty"`
	Status       TaskStatus   `json:"status,omitempty"`
	SyncedAt     *time.Time   `json:"syncedAt,omitempty"`
}
