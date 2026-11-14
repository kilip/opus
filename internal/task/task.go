// Package task is the task feature. Other packages depend on this package only,
// never on its subpackages.
package task

import (
	"github.com/gofiber/fiber/v3"
	"github.com/kilip/opus/internal/task/handler"
)

// Register mounts the task HTTP routes on r.
func Register(r fiber.Router) {
	handler.New().Register(r)
}
