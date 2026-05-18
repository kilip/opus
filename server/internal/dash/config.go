package dash

// Config holds configuration for the Dash static server.
type Config struct {
	// Address is the TCP address the Dash static server listens on.
	// Default: ":8081".
	Address string `mapstructure:"address" json:"address" jsonschema:"default=:8081,description=TCP address the Dash static file server listens on"`
}
