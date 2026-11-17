package repo

import "github.com/kilip/opus/internal/task/domain"

type CreateInput struct {
	ProjectID   string
	Source      domain.TaskSource
	ExternalID  string
	ExternalURL string
	Title       string
	Description string
	Status      *domain.TaskStatus
	Priority    *domain.TaskPriority
	Type        *domain.TaskType
	Assignee    *domain.Assignee
	DueDate     string
	Labels      []string
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
