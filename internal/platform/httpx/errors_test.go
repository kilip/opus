package httpx_test

import (
	"context"
	"errors"
	"log/slog"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/opus/internal/platform/httpx"
)

type captureHandler struct {
	levels []slog.Level
}

func (h *captureHandler) Enabled(context.Context, slog.Level) bool { return true }
func (h *captureHandler) Handle(_ context.Context, r slog.Record) error {
	h.levels = append(h.levels, r.Level)
	return nil
}
func (h *captureHandler) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h *captureHandler) WithGroup(string) slog.Handler      { return h }

func TestErrorHandlerLogLevel(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want slog.Level
	}{
		{"4xx logs warn", httpx.Unauthorized(), slog.LevelWarn},
		{"5xx logs error", errors.New("boom"), slog.LevelError},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := &captureHandler{}
			app := fiber.New(fiber.Config{
				ErrorHandler: httpx.ErrorHandler(slog.New(h)),
			})
			app.Get("/test", func(fiber.Ctx) error { return tc.err })

			if _, err := app.Test(httptest.NewRequest("GET", "/test", nil)); err != nil {
				t.Fatalf("app.Test failed: %v", err)
			}
			if len(h.levels) != 1 || h.levels[0] != tc.want {
				t.Fatalf("got levels %v, want [%v]", h.levels, tc.want)
			}
		})
	}
}

func TestUnauthorized(t *testing.T) {
	err := httpx.Unauthorized()
	var fiberErr *fiber.Error
	if !errors.As(err, &fiberErr) {
		t.Fatalf("expected *fiber.Error, got %T", err)
	}
	if fiberErr.Code != fiber.StatusUnauthorized {
		t.Errorf("got status %d, want %d", fiberErr.Code, fiber.StatusUnauthorized)
	}
	if fiberErr.Message != httpx.ErrUnauthorized.Error() {
		t.Errorf("got message %q, want %q", fiberErr.Message, httpx.ErrUnauthorized.Error())
	}
}

func TestCSRFForbidden(t *testing.T) {
	err := httpx.CSRFForbidden()
	var fiberErr *fiber.Error
	if !errors.As(err, &fiberErr) {
		t.Fatalf("expected *fiber.Error, got %T", err)
	}
	if fiberErr.Code != fiber.StatusForbidden {
		t.Errorf("got status %d, want %d", fiberErr.Code, fiber.StatusForbidden)
	}
	if fiberErr.Message != httpx.ErrCSRFForbidden.Error() {
		t.Errorf("got message %q, want %q", fiberErr.Message, httpx.ErrCSRFForbidden.Error())
	}
}
