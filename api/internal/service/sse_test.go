package service_test

import (
	"testing"

	"github.com/kilip/opus/api/internal/service"
	"github.com/stretchr/testify/assert"
)

func TestSSEHub_Interface(t *testing.T) {
	var hub service.SSEHub
	assert.Nil(t, hub)
}
