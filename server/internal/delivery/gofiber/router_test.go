// server/internal/delivery/gofiber/router_test.go
package gofiber_test

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/kilip/opus/server/internal/delivery/gofiber"
	"github.com/kilip/opus/server/internal/shared/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRouter(t *testing.T) {
	cfg := gofiber.Config{Address: ":8080"}
	log := &logger.NoopLogger{}
	app := gofiber.New(cfg, log)

	tests := []struct {
		name           string
		path           string
		expectedStatus int
		validate       func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:           "Health Check",
			path:           "/health",
			expectedStatus: 200,
			validate: func(t *testing.T, w *httptest.ResponseRecorder) {
				var body gofiber.Envelope[map[string]string]
				err := json.NewDecoder(w.Body).Decode(&body)
				require.NoError(t, err)
				assert.Equal(t, "ok", body.Data["status"])
			},
		},
		{
			name:           "Global Error 404",
			path:           "/not-found-route",
			expectedStatus: 404,
			validate: func(t *testing.T, w *httptest.ResponseRecorder) {
				var body gofiber.Envelope[any]
				err := json.NewDecoder(w.Body).Decode(&body)
				require.NoError(t, err)
				assert.NotNil(t, body.Error)
				assert.Equal(t, "https://opus.local/errors/not-found", body.Error.Type)
				assert.Equal(t, "Resource Not Found", body.Error.Title)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := app.Test(httptest.NewRequest("GET", tt.path, nil))
			require.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			rec := httptest.NewRecorder()
			if resp.Body != nil {
				_, _ = rec.Body.ReadFrom(resp.Body)
			}
			tt.validate(t, rec)
		})
	}
}
