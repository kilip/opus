package repo

import (
	"time"

	"github.com/kilip/opus/internal/task/domain"
)

type CreateInput struct {
	ProjectID    string
	Key          string
	Source       domain.TaskSource
	ExternalID   string
	ExternalURL  string
	SourceStatus string
	Title        string
	Description  string
	Status       *domain.TaskStatus
	Priority     *domain.TaskPriority
	Type         *domain.TaskType
	Assignee     *domain.Assignee
	DueDate      string
	Labels       []string
	SyncedAt     *time.Time
}

// ImportResult holds the upserted task and whether the operation created, modified, or transitioned it.
type ImportResult struct {
	Task          domain.Task
	PrevStatus    domain.TaskStatus
	Created       bool
	Changed       bool
	StatusChanged bool
}

type UpdateInput struct {
	Title         *string
	Description   *string
	Status        *domain.TaskStatus
	Priority      *domain.TaskPriority
	Type          *domain.TaskType
	Assignee      *domain.Assignee
	ClearAssignee bool
	DueDate       *string
	Labels        *[]string
}

// UpdateResult holds the updated task, the previous status, and whether the status changed.
type UpdateResult struct {
	Task          domain.Task
	PrevStatus    domain.TaskStatus
	StatusChanged bool
}
