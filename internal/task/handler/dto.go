package handler

import "time"

// TaskSource identifies tracker task originates from.
type TaskSource string

const (
	TaskSourceGitHub TaskSource = "github"
	TaskSourceLinear TaskSource = "linear"
	TaskSourceSentry TaskSource = "sentry"
	TaskSourceAgent  TaskSource = "agent"
	TaskSourceManual TaskSource = "manual"
)

// TaskStatus is normalised workflow stage.
type TaskStatus string

const (
	TaskStatusBacklog    TaskStatus = "backlog"
	TaskStatusTodo       TaskStatus = "todo"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusReview     TaskStatus = "review"
	TaskStatusDone       TaskStatus = "done"
)

// TaskPriority is normalised priority, lowest first.
type TaskPriority string

const (
	TaskPriorityP0 TaskPriority = "0"
	TaskPriorityP1 TaskPriority = "1"
	TaskPriorityP2 TaskPriority = "2"
	TaskPriorityP3 TaskPriority = "3"
)

// TaskType is normalised kind of work.
type TaskType string

const (
	TaskTypeFeat     TaskType = "feat"
	TaskTypeBug      TaskType = "bug"
	TaskTypeChore    TaskType = "chore"
	TaskTypeRefactor TaskType = "refactor"
	TaskTypeDocs     TaskType = "docs"
)

// ActivityKind distinguishes entries of task chatter timeline.
type ActivityKind string

const (
	ActivityKindStatus  ActivityKind = "status"
	ActivityKindComment ActivityKind = "comment"
	ActivityKindAssign  ActivityKind = "assign"
	ActivityKindCommit  ActivityKind = "commit"
	ActivityKindReview  ActivityKind = "review"
)

// Assignee is person or agent task assigned to.
type Assignee struct {
	ID   string `json:"id" validate:"required"`
	Name string `json:"name" validate:"required"`
}

// Task is issue normalised from source tracker.
type Task struct {
	// ID is Opus identifier.
	ID string `json:"id" validate:"required"`
	// Key is human-readable reference, e.g. OP-12.
	Key          string       `json:"key" validate:"required"`
	Source       TaskSource   `json:"source" validate:"required"`
	ExternalID   string       `json:"externalId,omitempty"`
	ExternalURL  string       `json:"externalUrl,omitempty"`
	Title        string       `json:"title" validate:"required"`
	Description  string       `json:"description" validate:"required"`
	Status       TaskStatus   `json:"status" validate:"required"`
	SourceStatus string       `json:"sourceStatus,omitempty"`
	Priority     TaskPriority `json:"priority" validate:"required"`
	Type         TaskType     `json:"type" validate:"required"`
	Assignee     *Assignee    `json:"assignee,omitempty"`
	// DueDate is calendar date (YYYY-MM-DD).
	DueDate   string    `json:"dueDate,omitempty" format:"date"`
	Labels    []string  `json:"labels" validate:"required"`
	CreatedAt time.Time `json:"createdAt" validate:"required"`
	UpdatedAt time.Time `json:"updatedAt" validate:"required"`
}

// Activity is one entry of task chatter timeline.
type Activity struct {
	ID        string       `json:"id" validate:"required"`
	Kind      ActivityKind `json:"kind" validate:"required"`
	Author    string       `json:"author" validate:"required"`
	Body      string       `json:"body" validate:"required"`
	CreatedAt time.Time    `json:"createdAt" validate:"required"`
}

// CreateTaskRequest is payload for creating task.
type CreateTaskRequest struct {
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
	Kind   *ActivityKind `json:"kind,omitempty"`
	Author string        `json:"author,omitempty"`
	Body   string        `json:"body" validate:"required"`
}
