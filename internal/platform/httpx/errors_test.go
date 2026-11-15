package httpx_test

import (
	"errors"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/opus/internal/platform/httpx"
)

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
