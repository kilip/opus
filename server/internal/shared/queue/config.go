package queue

// Driver identifies the backing engine for the job queue.
type Driver string

const (
	// DriverSQLite selects the SQLite-backed queue, which reuses the shared Ent database connection.
	DriverSQLite Driver = "sqlite"
	// DriverPostgres selects the PostgreSQL-backed queue.
	DriverPostgres Driver = "postgres"
	// DriverRedis selects the Redis-backed queue via Asynq.
	DriverRedis Driver = "redis"
)

// Config holds all queue configuration.
// Owned by the queue package; composed into the root config.Config
// by internal/config/model.go.
type Config struct {
	Driver         Driver `mapstructure:"driver" json:"driver" jsonschema:"enum=sqlite,enum=postgres,enum=redis,default=sqlite,description=Queue backend driver"`
	Concurrency    int    `mapstructure:"concurrency" json:"concurrency" jsonschema:"default=10,description=Number of concurrent job processing workers"`
	RetentionHours int    `mapstructure:"retention_hours" json:"retention_hours" jsonschema:"default=168,description=Hours to retain completed and failed jobs before pruning"`
	DSN            string `mapstructure:"dsn" json:"dsn,omitempty" jsonschema:"description=Required for postgres and redis drivers only. Inject via OPUS_QUEUE_DSN."`
}
