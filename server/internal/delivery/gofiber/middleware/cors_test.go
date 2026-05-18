package middleware_test

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/opus/server/internal/delivery/gofiber/middleware"
	"github.com/stretchr/testify/assert"
)

func TestCORS_ValidOrigin(t *testing.T) {
	app := fiber.New()
	cfg := middleware.CORSConfig{
		AllowedOrigins:   []string{"http://localhost:3000"},
		AllowCredentials: true,
	}

	app.Use(middleware.CORS(cfg))

	app.Get("/test", func(c fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "http://localhost:3000", resp.Header.Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "true", resp.Header.Get("Access-Control-Allow-Credentials"))
}

func TestCORS_InvalidOrigin(t *testing.T) {
	app := fiber.New()
	cfg := middleware.CORSConfig{
		AllowedOrigins: []string{"http://localhost:3000"},
	}

	app.Use(middleware.CORS(cfg))

	app.Get("/test", func(c fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://malicious.com")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Empty(t, resp.Header.Get("Access-Control-Allow-Origin"))
}

func TestCORS_WildcardPanicWithCredentials(t *testing.T) {
	cfg := middleware.CORSConfig{
		AllowedOrigins:   []string{"*"},
		AllowCredentials: true,
	}

	assert.PanicsWithValue(t, "cors: AllowedOrigins must not contain \"*\" when AllowCredentials is true", func() {
		middleware.CORS(cfg)
	})
}

func TestCORS_Preflight(t *testing.T) {
	app := fiber.New()
	cfg := middleware.CORSConfig{
		AllowedOrigins: []string{"http://localhost:3000"},
		AllowedMethods: []string{"POST", "OPTIONS"},
		MaxAge:         3600,
	}

	app.Use(middleware.CORS(cfg))

	req := httptest.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "POST")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 204, resp.StatusCode)
	assert.Equal(t, "http://localhost:3000", resp.Header.Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "POST, OPTIONS", resp.Header.Get("Access-Control-Allow-Methods"))
}
