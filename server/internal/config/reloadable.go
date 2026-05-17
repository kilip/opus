// server/internal/config/reloadable.go
package config

// Reloadable defines an interface for services that need to react to config changes.
type Reloadable interface {
	Reload(cfg *Config)
}
