package middleware

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
)

func TestErrorHandler(t *testing.T) {
	app := fiber.New(fiber.Config{
		ErrorHandler: ErrorHandler,
	})

	t.Run("GenericError", func(t *testing.T) {
		app.Get("/generic", func(c fiber.Ctx) error {
			return errors.New("something went wrong")
		})

		req := httptest.NewRequest("GET", "/generic", nil)
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
		assert.Equal(t, "ERR_500", errMap["code"])
		assert.Equal(t, "something went wrong", errMap["message"])
	})

	t.Run("FiberError_NotFound", func(t *testing.T) {
		app.Get("/notfound", func(c fiber.Ctx) error {
			return fiber.NewError(fiber.StatusNotFound, "custom not found")
		})

		req := httptest.NewRequest("GET", "/notfound", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		var res map[string]interface{}
		err = json.Unmarshal(body, &res)
		assert.NoError(t, err)

		assert.False(t, res["success"].(bool))
		
		errMap := res["error"].(map[string]interface{})
		assert.Equal(t, "ERR_404", errMap["code"])
		assert.Equal(t, "custom not found", errMap["message"])
	})

	t.Run("FiberError_Forbidden", func(t *testing.T) {
		app.Get("/forbidden", func(c fiber.Ctx) error {
			return fiber.NewError(fiber.StatusForbidden, "access denied")
		})

		req := httptest.NewRequest("GET", "/forbidden", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		var res map[string]interface{}
		err = json.Unmarshal(body, &res)
		assert.NoError(t, err)

		assert.False(t, res["success"].(bool))
		
		errMap := res["error"].(map[string]interface{})
		assert.Equal(t, "ERR_403", errMap["code"])
		assert.Equal(t, "access denied", errMap["message"])
	})
}
