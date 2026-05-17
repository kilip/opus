// server/internal/shared/logger/config.go
package logger

// Config holds all logging infrastructure parameters.
type Config struct {
	// Level controls the minimum severity of emitted log entries.
	// Valid values: "debug", "info", "warn", "error". Default: "info".
	Level string `mapstructure:"level" json:"level" jsonschema:"enum=debug,enum=info,enum=warn,enum=error,default=info"`

	// Format controls the output encoding for console logs.
	// Valid values: "json", "text". Default: "json".
	Format string `mapstructure:"format" json:"format" jsonschema:"enum=json,enum=text,default=json"`

	// FileEnabled determines if log messages are also written to a physical file.
	FileEnabled bool `mapstructure:"file_enabled" json:"file_enabled" jsonschema:"default=false"`

	// Filename is the target path for the active log file (e.g. "logs/opus.log").
	Filename string `mapstructure:"filename" json:"filename" jsonschema:"default=logs/opus.log"`

	// MaxSize defines the maximum size of a log file in Megabytes before it rotates.
	MaxSize int `mapstructure:"max_size" json:"max_size" jsonschema:"default=100"`

	// MaxBackups specifies the maximum number of retained old log files.
	MaxBackups int `mapstructure:"max_backups" json:"max_backups" jsonschema:"default=7"`

	// MaxAge is the maximum number of days to retain old log files.
	MaxAge int `mapstructure:"max_age" json:"max_age" jsonschema:"default=14"`

	// Compress determines if rotated log files should be gzipped.
	Compress bool `mapstructure:"compress" json:"compress" jsonschema:"default=true"`
}

// DefaultConfig returns standard operational default parameters.
func DefaultConfig() Config {
	return Config{
		Level:       "info",
		Format:      "json",
		FileEnabled: false,
		Filename:    "logs/opus.log",
		MaxSize:     100,
		MaxBackups:  7,
		MaxAge:      14,
		Compress:    true,
	}
}
