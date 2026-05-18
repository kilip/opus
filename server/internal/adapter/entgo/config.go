package entgo

// Config holds the configuration details for the database connection.
type Config struct {
	// Driver defines the database engine/driver to use.
	Driver string `mapstructure:"driver" json:"driver" jsonschema:"enum=sqlite3,enum=postgres,default=sqlite3,description=Database driver"`

	// DSN is the data source name containing connection options.
	DSN string `mapstructure:"dsn" json:"dsn" jsonschema:"description=Data source name. Inject via OPUS_DATABASE_DSN for production secrets"`
}
