// Package task is the task feature. Other packages depend on this package only,
// never on its subpackages.
package task

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/opus/internal/task/handler"
	"github.com/kilip/opus/internal/task/repo"
)

// Task is the task feature facade.
type Task struct {
	handler *handler.Handler
}

// ponytail: in-memory mutex store seeded with initial tasks; replace with Ent client when schema lands.
func New() *Task {
	r := repo.NewMemory(time.Now)
	return &Task{handler: handler.New(r)}
}

// Register mounts the task HTTP routes on r.
func (t *Task) Register(r fiber.Router) {
	t.handler.Register(r)
}
