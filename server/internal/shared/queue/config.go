package queue

// Driver identifies the backing engine for the job queue.
type Driver string

const (
	// DriverDatabase selects the Ent-backed queue, which works with any Ent-supported database.
	DriverDatabase Driver = "database"
	// DriverRedis selects the Redis-backed queue via Asynq.
	DriverRedis Driver = "redis"
)

// Config holds all queue configuration.
// It is owned by the queue package and composed into the root config.Config
// by internal/config/model.go.
//
// Environment variable overrides follow the OPUS_ prefix convention:
//
//	OPUS_QUEUE_DRIVER       — sets Driver
//	OPUS_QUEUE_DSN          — sets DSN
//	OPUS_QUEUE_CONCURRENCY  — sets Concurrency
type Config struct {
	// Driver selects the queue backend.
	// Valid values: "database" (default), "redis".
	Driver Driver `mapstructure:"driver" json:"driver" jsonschema:"enum=database,enum=redis,default=database,description=Queue backend driver"`

	// DSN is the data source name for the selected driver.
	// database: unused (the global Ent client is used)
	// redis:    Redis URL (e.g. "redis://localhost:6379")
	DSN string `mapstructure:"dsn" json:"dsn" jsonschema:"description=Data source name for the queue backend. Use env var OPUS_QUEUE_DSN for secrets."`

	// Concurrency is the number of concurrent job workers.
	// Default: 10.
	Concurrency int `mapstructure:"concurrency" json:"concurrency" jsonschema:"default=10,description=Number of concurrent job processing workers"`

	// RetentionHours is the number of hours to retain completed/failed jobs
	// in the database before pruning. Default: 168 (7 days).
	RetentionHours int `mapstructure:"retention_hours" json:"retention_hours" jsonschema:"default=168,description=Hours to retain completed and failed jobs before pruning"`
}
