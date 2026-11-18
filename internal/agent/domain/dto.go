package domain

import "time"

// GetTaskInput is the input for the get_task MCP tool.
type GetTaskInput struct {
	Ref string `json:"ref,omitempty" jsonschema:"Task reference: OP-<n>, #<issue>, or task UUID. Defaults to the bound run task when omitted."`
}

// TaskActivityItem represents a single entry on a task's activity timeline.
type TaskActivityItem struct {
	ID        string    `json:"id" jsonschema:"Activity UUID"`
	Kind      string    `json:"kind" jsonschema:"Activity kind"`
	Author    string    `json:"author,omitempty" jsonschema:"Activity author (e.g. human, vibe)"`
	Body      string    `json:"body" jsonschema:"Untrusted activity body wrapped in untrusted-task-data tags"`
	CreatedAt time.Time `json:"created_at" jsonschema:"Creation timestamp"`
}

// GetTaskOutput is the output for the get_task MCP tool.
type GetTaskOutput struct {
	ID          string             `json:"id" jsonschema:"Task UUID"`
	Key         string             `json:"key" jsonschema:"Task key (e.g. OP-12)"`
	ProjectID   string             `json:"project_id" jsonschema:"Project UUID"`
	ExternalID  string             `json:"external_id,omitempty" jsonschema:"External tracker reference"`
	ExternalURL string             `json:"external_url,omitempty" jsonschema:"External tracker URL"`
	Title       string             `json:"title" jsonschema:"Untrusted task title wrapped in untrusted-task-data tags"`
	Description string             `json:"description" jsonschema:"Untrusted task description wrapped in untrusted-task-data tags"`
	Truncated   bool               `json:"truncated" jsonschema:"True when description exceeded 8KB and was truncated"`
	Status      string             `json:"status" jsonschema:"Current task status"`
	Priority    string             `json:"priority" jsonschema:"Task priority"`
	Type        string             `json:"type" jsonschema:"Task type"`
	Labels      []string           `json:"labels" jsonschema:"Task labels"`
	Activities  []TaskActivityItem `json:"activities,omitempty" jsonschema:"Task activity timeline"`
}

// ListTasksInput is the input for the list_tasks MCP tool.
type ListTasksInput struct {
	ProjectID string `json:"project_id,omitempty" jsonschema:"Optional project UUID filter (must be within allowed scope)"`
	Status    string `json:"status,omitempty" jsonschema:"Optional task status filter"`
	Limit     int    `json:"limit,omitempty" jsonschema:"Maximum number of tasks to return (max 50)"`
}

// TaskSummary is a compact task representation returned by list_tasks.
type TaskSummary struct {
	ID         string   `json:"id" jsonschema:"Task UUID"`
	Key        string   `json:"key" jsonschema:"Task key (e.g. OP-12)"`
	ProjectID  string   `json:"project_id" jsonschema:"Project UUID"`
	ExternalID string   `json:"external_id,omitempty" jsonschema:"External tracker reference"`
	Title      string   `json:"title" jsonschema:"Untrusted task title wrapped in untrusted-task-data tags"`
	Status     string   `json:"status" jsonschema:"Current task status"`
	Priority   string   `json:"priority" jsonschema:"Task priority"`
	Type       string   `json:"type" jsonschema:"Task type"`
	Labels     []string `json:"labels" jsonschema:"Task labels"`
}

// ListTasksOutput is the output for the list_tasks MCP tool.
type ListTasksOutput struct {
	Tasks []TaskSummary `json:"tasks" jsonschema:"Tasks within allowed project scope"`
	Total int           `json:"total" jsonschema:"Total matching tasks before applying limit"`
}

// SetStatusInput is the input for the set_status MCP tool.
type SetStatusInput struct {
	Ref    string `json:"ref,omitempty" jsonschema:"Task reference: OP-<n>, #<issue>, or task UUID. Defaults to the bound run task when omitted."`
	Status string `json:"status" jsonschema:"Target task status (backlog, todo, in-progress, review, blocked, done)"`
}

// SetStatusOutput is the output for the set_status MCP tool.
type SetStatusOutput struct {
	ID     string `json:"id" jsonschema:"Task UUID"`
	Key    string `json:"key" jsonschema:"Task key"`
	Status string `json:"status" jsonschema:"Updated task status"`
}

// AddActivityInput is the input for the add_activity MCP tool.
type AddActivityInput struct {
	Ref  string `json:"ref,omitempty" jsonschema:"Task reference: OP-<n>, #<issue>, or task UUID. Defaults to the bound run task when omitted."`
	Kind string `json:"kind,omitempty" jsonschema:"Activity kind (comment, commit, review, status, assign); defaults to comment"`
	Body string `json:"body" jsonschema:"Activity body text"`
}

// AddActivityOutput is the output for the add_activity MCP tool.
type AddActivityOutput struct {
	ID        string    `json:"id" jsonschema:"Created activity UUID"`
	TaskID    string    `json:"task_id" jsonschema:"Task UUID"`
	Kind      string    `json:"kind" jsonschema:"Activity kind"`
	Author    string    `json:"author" jsonschema:"Activity author"`
	Body      string    `json:"body" jsonschema:"Sanitized activity body"`
	CreatedAt time.Time `json:"created_at" jsonschema:"Creation timestamp"`
}

// SubmitReportInput is the input for the submit_report MCP tool.
type SubmitReportInput struct {
	Report
}

// SubmitReportOutput is the output for the submit_report MCP tool.
type SubmitReportOutput struct {
	RunID      string `json:"run_id" jsonschema:"Bound Run UUID"`
	TaskID     string `json:"task_id" jsonschema:"Bound Task UUID"`
	ActivityID string `json:"activity_id" jsonschema:"Recorded summary activity UUID"`
	Report     Report `json:"report" jsonschema:"Sanitized and persisted report"`
}
