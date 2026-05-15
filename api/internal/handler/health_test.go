package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
)

func TestHealthHandler_Check(t *testing.T) {
	app := fiber.New()
	handler := NewHealthHandler()
	app.Get("/health", handler.Check)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	resp, _ := app.Test(req)

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	assert.True(t, result["success"].(bool))
	data := result["data"].(map[string]interface{})
	assert.Equal(t, "ok", data["status"])
	assert.Equal(t, "1.0.1", data["version"])
	assert.Equal(t, "sqlite", data["db"])
}
