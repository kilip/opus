package config

import (
	"log/slog"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func resetLogger() {
	logger = nil
	loggerOnce = sync.Once{}
}

func TestGetLogger_Singleton(t *testing.T) {
	resetLogger()
	defer resetLogger()

	l1 := GetLogger()
	l2 := GetLogger()

	assert.NotNil(t, l1)
	assert.Same(t, l1, l2)
}

func TestGetLogger_Development(t *testing.T) {
	resetConfig()
	resetLogger()
	defer resetConfig()
	defer resetLogger()

	t.Setenv("OPUS_SERVER_ENV", "development")

	l := GetLogger()
	assert.NotNil(t, l)

	// In development, it should use a TextHandler
	// We check this by seeing if l.Handler() is not a JSONHandler
	// (Since we can't easily access the internal fields of the standard handlers)
	// Actually, we can check the string representation or just ensure it's a valid handler.
	h := l.Handler()
	assert.NotNil(t, h)
	
	// We can try to cast to see if it's what we expect
	// Note: slog handlers are often wrapped, but the direct ones are exported.
	_, isJSON := h.(*slog.JSONHandler)
	assert.False(t, isJSON, "Should not be a JSONHandler in development")
}

func TestGetLogger_Production(t *testing.T) {
	resetConfig()
	resetLogger()
	defer resetConfig()
	defer resetLogger()

	t.Setenv("OPUS_SERVER_ENV", "production")

	l := GetLogger()
	assert.NotNil(t, l)

	h := l.Handler()
	assert.NotNil(t, h)
	
	_, isJSON := h.(*slog.JSONHandler)
	assert.True(t, isJSON, "Should be a JSONHandler in production")
}
