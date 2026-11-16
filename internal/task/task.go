// Package task is the task feature. Other packages depend on this package only,
// never on its subpackages.
package task

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/opus/internal/platform/eventbus"
	"github.com/kilip/opus/internal/task/domain"
	"github.com/kilip/opus/internal/task/handler"
	"github.com/kilip/opus/internal/task/repo"
	"github.com/kilip/opus/internal/task/service"
)

// Event type constants re-exported from domain.
const (
	EventReceived      = domain.EventReceived
	EventStatusChanged = domain.EventStatusChanged

	StatusBacklog    = domain.TaskStatusBacklog
	StatusTodo       = domain.TaskStatusTodo
	StatusInProgress = domain.TaskStatusInProgress
	StatusReview     = domain.TaskStatusReview
	StatusDone       = domain.TaskStatusDone
)

// Re-exported domain types.
type (
	TaskStatus               = domain.TaskStatus
	Status                   = domain.TaskStatus
	TaskReceivedPayload      = domain.TaskReceivedPayload
	TaskStatusChangedPayload = domain.TaskStatusChangedPayload
)

// Task is the task feature facade.
type Task struct {
	handler *handler.Handler
}

// ponytail: in-memory mutex store seeded with initial tasks; replace with Ent client when schema lands.
func New(bus *eventbus.Bus) *Task {
	r := repo.NewMemory(time.Now)
	return &Task{handler: handler.New(service.New(r, bus))}
}

// Register mounts the task HTTP routes on r.
func (t *Task) Register(r fiber.Router) {
	t.handler.Register(r)
}
