package middleware

import (
	"bytes"
	"context"
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

type recordHandler struct {
	records []map[string]interface{}
	buf     *bytes.Buffer
}

func newRecordHandler() *recordHandler {
	return &recordHandler{
		buf: new(bytes.Buffer),
	}
}

func (h *recordHandler) Enabled(_ context.Context, _ slog.Level) bool { return true }
func (h *recordHandler) Handle(_ context.Context, r slog.Record) error {
	// We use a temporary JSON handler to format the record into our buffer
	var buf bytes.Buffer
	jh := slog.NewJSONHandler(&buf, nil)
	if err := jh.Handle(context.Background(), r); err != nil {
		return err
	}
	
	var m map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		return err
	}
	h.records = append(h.records, m)
	return nil
}
func (h *recordHandler) WithAttrs(attrs []slog.Attr) slog.Handler { return h }
func (h *recordHandler) WithGroup(name string) slog.Handler      { return h }

func TestLogger(t *testing.T) {
	// Setup custom logger to capture output
	handler := newRecordHandler()
	logger := slog.New(handler)
	
	// Save original and restore after test
	config.SetLogger(logger)
	defer config.ResetLogger()

	app := fiber.New()
	app.Use(Logger())

	app.Get("/test", func(c fiber.Ctx) error {
		return c.SendString("OK")
	})

	t.Run("LogFormat", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		assert.Equal(t, "OK", string(body))

		// Check logs
		assert.Len(t, handler.records, 1)
		log := handler.records[0]

		assert.Equal(t, "request processed", log["msg"])
		assert.Equal(t, "GET", log["method"])
		assert.Equal(t, "/test", log["path"])
		assert.Equal(t, float64(200), log["status"]) // JSON numbers are float64
		assert.Contains(t, log, "latency")
		assert.Contains(t, log, "ip")
	})

	t.Run("CustomStatus", func(t *testing.T) {
		handler.records = nil // Reset records
		
		app.Get("/error", func(c fiber.Ctx) error {
			return c.Status(http.StatusBadRequest).SendString("Bad Request")
		})

		req := httptest.NewRequest("GET", "/error", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		assert.Len(t, handler.records, 1)
		log := handler.records[0]
		assert.Equal(t, float64(400), log["status"])
		assert.Equal(t, "/error", log["path"])
	})
}
