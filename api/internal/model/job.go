package model

import "time"

// JobStatus represents the current lifecycle state of a job.
type JobStatus string

const (
	StatusPending   JobStatus = "pending"
	StatusRunning   JobStatus = "running"
	StatusCompleted JobStatus = "completed"
	StatusFailed    JobStatus = "failed"
)

// Job represents a unit of background work to be executed by the worker engine.
type Job struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`            // matches registered HandlerFunc key
	Payload     []byte    `json:"payload"`         // JSON-encoded handler input
	Priority    int       `json:"priority"`        // 0-10, higher = more urgent
	Status      JobStatus `json:"status"`
	Retries     int       `json:"retries"`
	MaxRetries  int       `json:"max_retries"`
	ScheduledAt time.Time `json:"scheduled_at"`    // zero = immediate
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Error       string    `json:"error,omitempty"` // last error message
}

// DeadLetter represents a job that has exceeded its MaxRetries threshold.
type DeadLetter struct {
	ID        string    `json:"id"`
	JobID     string    `json:"job_id"`
	Type      string    `json:"type"`
	Payload   []byte    `json:"payload"`
	LastError string    `json:"last_error"`
	Retries   int       `json:"retries"`
	CreatedAt time.Time `json:"created_at"`
}
