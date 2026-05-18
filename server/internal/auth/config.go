package auth

// Config represents auth feature configuration.
type Config struct {
	JWTSecret       string      `mapstructure:"jwt_secret" json:"-" jsonschema:"-"`
	AccessTokenTTL  string      `mapstructure:"access_token_ttl" json:"access_token_ttl" jsonschema:"default=15m"`
	RefreshTokenTTL string      `mapstructure:"refresh_token_ttl" json:"refresh_token_ttl" jsonschema:"default=168h"`
	OAuth           OAuthConfig `mapstructure:"oauth" json:"oauth"`
}

// OAuthConfig groups supported OAuth configurations.
type OAuthConfig struct {
	Google ProviderCredentials `mapstructure:"google" json:"google"`
	GitHub ProviderCredentials `mapstructure:"github" json:"github"`
}

// ProviderCredentials contains OAuth IDs and secret configurations.
type ProviderCredentials struct {
	ClientID     string `mapstructure:"client_id" json:"client_id"`
	ClientSecret string `mapstructure:"client_secret" json:"-" jsonschema:"-"`
	RedirectURL  string `mapstructure:"redirect_url" json:"redirect_url"`
}
