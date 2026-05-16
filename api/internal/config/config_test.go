package config

import (
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
