// server/internal/delivery/gofiber/middleware/logger_test.go
package middleware_test

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/opus/server/internal/delivery/gofiber/middleware"
	"github.com/kilip/opus/server/internal/shared/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestLogger(t *testing.T) {
	app := fiber.New()
	log := &logger.NoopLogger{}
	app.Use(middleware.RequestLogger(log))

	app.Get("/test", func(c fiber.Ctx) error {
		return c.SendStatus(200)
	})

	resp, err := app.Test(httptest.NewRequest("GET", "/test", nil))
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}
