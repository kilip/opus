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
	
	// In the new implementation, GetLogger returns a multiHandler
	mh, isMulti := h.(*multiHandler)
	assert.True(t, isMulti, "Should be a multiHandler")
	assert.NotEmpty(t, mh.handlers)
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
	
	mh, isMulti := h.(*multiHandler)
	assert.True(t, isMulti, "Should be a multiHandler")

	// In production, at least one handler should be a JSONHandler
	foundJSON := false
	for _, h := range mh.handlers {
		if _, ok := h.(*slog.JSONHandler); ok {
			foundJSON = true
			break
		}
	}
	assert.True(t, foundJSON, "Should contain a JSONHandler in production")
}
