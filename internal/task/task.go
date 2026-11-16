// Package task is the task feature. Other packages depend on this package only,
// never on its subpackages.
package task

import (
	"github.com/gofiber/fiber/v3"
	"github.com/kilip/opus/ent"
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

// New returns a new Task feature facade backed by Ent.
func New(client *ent.Client, bus *eventbus.Bus) *Task {
	r := repo.New(client)
	return &Task{handler: handler.New(service.New(r, bus))}
}

// Register mounts the task HTTP routes on r.
func (t *Task) Register(r fiber.Router) {
	t.handler.Register(r)
}
