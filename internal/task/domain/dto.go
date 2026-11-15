package domain

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
