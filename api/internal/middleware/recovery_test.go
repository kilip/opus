package middleware

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/opus/api/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestRecovery(t *testing.T) {
	// Setup custom logger to capture output
	handler := newRecordHandler()
	logger := slog.New(handler)
	
	// Save original and restore after test
	config.SetLogger(logger)
	defer config.ResetLogger()

	app := fiber.New()
	app.Use(Recovery())

	t.Run("Panic", func(t *testing.T) {
		handler.records = nil // Reset records
		
		app.Get("/panic", func(c fiber.Ctx) error {
			panic("something went wrong")
		})

		req := httptest.NewRequest("GET", "/panic", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		var res map[string]interface{}
		err = json.Unmarshal(body, &res)
		assert.NoError(t, err)

		assert.False(t, res["success"].(bool))
		assert.Nil(t, res["data"])
		
		errMap := res["error"].(map[string]interface{})
		assert.Equal(t, "INTERNAL_SERVER_ERROR", errMap["code"])
		assert.Equal(t, "An unexpected error occurred", errMap["message"])

		// Check logs
		assert.Len(t, handler.records, 1)
		log := handler.records[0]
		assert.Equal(t, "panic recovered", log["msg"])
		assert.Equal(t, "something went wrong", log["panic"])
		assert.Equal(t, "GET", log["method"])
		assert.Equal(t, "/panic", log["path"])
	})

	t.Run("NoPanic", func(t *testing.T) {
		handler.records = nil // Reset records
		
		app.Get("/ok", func(c fiber.Ctx) error {
			return c.Status(http.StatusOK).JSON(fiber.Map{
				"success": true,
				"data":    "all good",
			})
		})

		req := httptest.NewRequest("GET", "/ok", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		var res map[string]interface{}
		err = json.Unmarshal(body, &res)
		assert.NoError(t, err)

		assert.True(t, res["success"].(bool))
		assert.Equal(t, "all good", res["data"])
		
		// Should be no panic logs
		assert.Len(t, handler.records, 0)
	})
}
