package domain

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
	ID   string
	Name string
}

// Task is issue normalised from source tracker.
type Task struct {
	ID           string
	Key          string
	Source       TaskSource
	ExternalID   string
	ExternalURL  string
	Title        string
	Description  string
	Status       TaskStatus
	SourceStatus string
	Priority     TaskPriority
	Type         TaskType
	Assignee     *Assignee
	DueDate      string
	Labels       []string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Activity is entry in task timeline.
type Activity struct {
	ID        string
	Kind      ActivityKind
	Author    string
	Body      string
	CreatedAt time.Time
}
