package config

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func resetConfig() {
	cfg = nil
	cfgOnce = sync.Once{}
	viper.Reset()
}

func TestGetOpusDir(t *testing.T) {
	// 1. Test OPUS_HOME override
	expected := "/tmp/opus_test"
	t.Setenv("OPUS_HOME", expected)
	assert.Equal(t, expected, GetOpusDir())

	// 2. Test default (empty env = production)
	t.Setenv("OPUS_HOME", "")
	t.Setenv("OPUS_SERVER_ENV", "")
	dir := GetOpusDir()
	assert.Contains(t, dir, ".opus")

	// 3. Test explicit development mode (force local .opus)
	t.Setenv("OPUS_SERVER_ENV", "development")
	// Use relative path comparison
	assert.Contains(t, GetOpusDir(), ".opus")

	// 4. Test explicit production mode (manual creation of local .opus)
	t.Setenv("OPUS_SERVER_ENV", "production")
	err := os.Mkdir(".opus", 0755)
	if err == nil {
		defer func() {
			_ = os.RemoveAll(".opus")
		}()
		assert.Contains(t, GetOpusDir(), ".opus")
	}

	// 5. Test production mode default (~/.opus)
	if _, err := os.Stat(".opus"); err == nil {
		_ = os.RemoveAll(".opus")
	}
	dir = GetOpusDir()
	home, _ := os.UserHomeDir()
	assert.Equal(t, filepath.Join(home, ".opus"), dir)
}

func TestGetConfig_Defaults(t *testing.T) {
	resetConfig()
	defer resetConfig()

	c := GetConfig()

	assert.NotNil(t, c)
	assert.Equal(t, "0.0.0.0", c.Server.Host)
	assert.Equal(t, 8080, c.Server.Port)
	assert.Equal(t, "production", c.Server.Env)
	assert.Equal(t, "sqlite", c.Database.Driver)
}

func TestGetConfig_EnvOverrides(t *testing.T) {
	resetConfig()
	defer resetConfig()

	t.Setenv("OPUS_SERVER_PORT", "9090")
	t.Setenv("OPUS_DATABASE_DRIVER", "postgres")

	c := GetConfig()

	assert.Equal(t, 9090, c.Server.Port)
	assert.Equal(t, "postgres", c.Database.Driver)
}

func TestGetConfig_Singleton(t *testing.T) {
	resetConfig()
	defer resetConfig()

	c1 := GetConfig()
	c2 := GetConfig()

	assert.Same(t, c1, c2)
}
