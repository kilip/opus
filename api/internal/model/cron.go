package model

import "time"

// CronSchedule defines a recurring job trigger stored in the database.
type CronSchedule struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CronExpr  string    `json:"cron_expression"` // standard 5-field cron expression
	JobType   string    `json:"job_type"`         // matches registered HandlerFunc key
	Payload   []byte    `json:"payload"`
	IsActive  bool      `json:"is_active"`
	LastRunAt time.Time `json:"last_run_at"`
	NextRunAt time.Time `json:"next_run_at"`
	UserID    string    `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
