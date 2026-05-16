package config

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func resetDatabase() {
	dbClient = nil
	dbOnce = sync.Once{}
}

func TestGetDatabase_Singleton(t *testing.T) {
	// We need to ensure config is set for SQLite in-memory to avoid real DB connections
	resetConfig()
	resetDatabase()
	defer resetConfig()
	defer resetDatabase()

	t.Setenv("OPUS_DATABASE_DRIVER", "sqlite")
	t.Setenv("OPUS_DATABASE_DSN", ":memory:")

	db1 := GetDatabase()
	db2 := GetDatabase()

	assert.NotNil(t, db1)
	assert.Same(t, db1, db2)
}

func TestGetDatabase_SQLite(t *testing.T) {
	resetConfig()
	resetDatabase()
	defer resetConfig()
	defer resetDatabase()

	// Use in-memory sqlite for testing
	t.Setenv("OPUS_DATABASE_DRIVER", "sqlite")
	t.Setenv("OPUS_DATABASE_DSN", ":memory:")

	db := GetDatabase()
	assert.NotNil(t, db)
	
	// Ensure we can close it (optional but good practice)
	// Actually, GetDatabase returns a pointer to a singleton that might be used by other tests 
	// if we don't reset correctly.
}
