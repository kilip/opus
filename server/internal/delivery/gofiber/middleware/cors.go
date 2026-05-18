// Package middleware implements global and route-specific HTTP middlewares for the GoFiber server.
package middleware

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
)

// CORSConfig holds CORS middleware configuration.
type CORSConfig struct {
	// AllowedOrigins is the list of origins permitted to make cross-origin requests.
	// Each entry must be a fully qualified origin: scheme + host + optional port.
	// Example: ["http://localhost:5173", "https://opus.example.com"]
	AllowedOrigins []string `mapstructure:"allowed_origins" json:"allowed_origins" jsonschema:"description=List of permitted origins for cross-origin requests"`

	// AllowedMethods lists the HTTP methods permitted in cross-origin requests.
	AllowedMethods []string `mapstructure:"allowed_methods" json:"allowed_methods" jsonschema:"description=Permitted HTTP methods for cross-origin requests"`

	// AllowedHeaders lists the request headers permitted in cross-origin requests.
	AllowedHeaders []string `mapstructure:"allowed_headers" json:"allowed_headers" jsonschema:"description=Permitted request headers for cross-origin requests"`

	// ExposeHeaders lists the response headers the browser is allowed to access.
	ExposeHeaders []string `mapstructure:"expose_headers" json:"expose_headers" jsonschema:"description=Response headers exposed to the browser"`

	// AllowCredentials enables Access-Control-Allow-Credentials.
	// Must be true for httpOnly cookie-based authentication.
	AllowCredentials bool `mapstructure:"allow_credentials" json:"allow_credentials" jsonschema:"default=true,description=Enable Access-Control-Allow-Credentials for cookie-based auth"`

	// MaxAge sets the Access-Control-Max-Age header in seconds.
	// Controls how long preflight results are cached by the browser.
	MaxAge int `mapstructure:"max_age" json:"max_age" jsonschema:"default=3600,description=Preflight cache duration in seconds"`
}

// CORS returns a configured Fiber CORS middleware from the provided CORSConfig.
// Panics at startup if AllowCredentials is true and AllowedOrigins contains "*".
func CORS(cfg CORSConfig) fiber.Handler {
	for _, origin := range cfg.AllowedOrigins {
		if origin == "*" && cfg.AllowCredentials {
			panic("cors: AllowedOrigins must not contain \"*\" when AllowCredentials is true")
		}
	}

	return cors.New(cors.Config{
		AllowOrigins:     cfg.AllowedOrigins,
		AllowMethods:     defaultMethods(cfg.AllowedMethods),
		AllowHeaders:     defaultHeaders(cfg.AllowedHeaders),
		ExposeHeaders:    cfg.ExposeHeaders,
		AllowCredentials: cfg.AllowCredentials,
		MaxAge:           cfg.MaxAge,
	})
}

func defaultMethods(methods []string) []string {
	if len(methods) == 0 {
		return []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	}
	return methods
}

func defaultHeaders(headers []string) []string {
	if len(headers) == 0 {
		return []string{"Origin", "Content-Type", "Accept", "Authorization"}
	}
	return headers
}
