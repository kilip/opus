package httpx

import (
	"errors"
	"log/slog"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/requestid"
)

var (
	ErrBadRequest    = errors.New("BAD_REQUEST")
	ErrUnauthorized  = errors.New("UNAUTHORIZED")
	ErrForbidden     = errors.New("FORBIDDEN")
	ErrCSRFForbidden = errors.New("CSRF_FORBIDDEN")
	ErrNotFound      = errors.New("RESOURCE_NOT_FOUND")
	ErrConflict      = errors.New("CONFLICT")
	ErrInternal      = errors.New("INTERNAL_SERVER_ERROR")
)

func NewError(status int, err error) error {
	return fiber.NewError(status, err.Error())
}

func BadRequest(err error) error {
	return NewError(fiber.StatusBadRequest, err)
}

func Unauthorized() error {
	return NewError(fiber.StatusUnauthorized, ErrUnauthorized)
}

func CSRFForbidden() error {
	return NewError(fiber.StatusForbidden, ErrCSRFForbidden)
}

func NotFound(err error) error {
	return NewError(fiber.StatusNotFound, err)
}

func Conflict(err error) error {
	return NewError(fiber.StatusConflict, err)
}

func ErrorHandler(logger *slog.Logger) fiber.ErrorHandler {
	return func(c fiber.Ctx, err error) error {
		code := fiber.StatusInternalServerError
		errCode := ErrInternal.Error()

		var e *fiber.Error
		if errors.As(err, &e) {
			code = e.Code
			errCode = e.Message
		}

		level := slog.LevelError
		if code < fiber.StatusInternalServerError {
			level = slog.LevelWarn
		}
		logger.Log(c.Context(), level, "request error",
			"request_id", requestid.FromContext(c),
			"method", c.Method(),
			"path", c.Path(),
			"status", code,
			"err", err,
		)

		return c.Status(code).JSON(ErrorResponse{
			Data:  nil,
			Error: errCode,
		})
	}
}
