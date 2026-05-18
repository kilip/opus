// Package gofiber_test contains the unit tests for the GoFiber delivery layer.
package gofiber_test

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/opus/server/internal/delivery/gofiber/response"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestResponseHelpers tests the standard envelope and API response formats.
func TestResponseHelpers(t *testing.T) {
	app := fiber.New()

	app.Get("/ok", func(c fiber.Ctx) error {
		return response.OK(c, fiber.Map{"foo": "bar"})
	})
	app.Post("/created", func(c fiber.Ctx) error {
		return response.Created(c, fiber.Map{"id": 123})
	})
	app.Delete("/no-content", func(c fiber.Ctx) error {
		return response.NoContent(c)
	})

	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
		validateBody   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:           "OK Response",
			method:         "GET",
			path:           "/ok",
			expectedStatus: 200,
			validateBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var body response.Envelope[map[string]string]
				err := json.NewDecoder(w.Body).Decode(&body)
				require.NoError(t, err)
				assert.Equal(t, "bar", body.Data["foo"])
			},
		},
		{
			name:           "Created Response",
			method:         "POST",
			path:           "/created",
			expectedStatus: 201,
			validateBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var body response.Envelope[map[string]int]
				err := json.NewDecoder(w.Body).Decode(&body)
				require.NoError(t, err)
				assert.Equal(t, 123, body.Data["id"])
			},
		},
		{
			name:           "No Content Response",
			method:         "DELETE",
			path:           "/no-content",
			expectedStatus: 204,
			validateBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, 0, w.Body.Len())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			rec := httptest.NewRecorder()
			if resp.Body != nil {
				_, _ = rec.Body.ReadFrom(resp.Body)
			}
			tt.validateBody(t, rec)
		})
	}
}
