package handler

import (
	"time"

	"github.com/gofiber/fiber/v3"
)

// Handler serves the task endpoints. For now it only returns fixture JSON.
type Handler struct {
	now func() time.Time
}

// New returns a Handler.
func New() *Handler {
	return &Handler{now: time.Now}
}

// Register mounts the task routes on r.
func (h *Handler) Register(r fiber.Router) {
	g := r.Group("/tasks")
	g.Get("/", h.List)
	g.Get("/:id", h.Get)
	g.Get("/:id/activity", h.Activity)
}

// List godoc
//
//	@Summary		List tasks
//	@Description	Returns every task, normalised from its source tracker.
//	@ID				list-tasks
//	@Tags			tasks
//	@Produce		json
//	@Success		200	{object}	TaskList
//	@Router			/tasks [get]
func (h *Handler) List(c fiber.Ctx) error {
	items := newFixtures(h.now()).tasks
	return c.JSON(TaskList{Items: items, Total: len(items)})
}

// Get godoc
//
//	@Summary		Get a task
//	@Description	Returns a single task by its Opus identifier.
//	@ID				get-task
//	@Tags			tasks
//	@Produce		json
//	@Param			id	path		string	true	"Task ID"
//	@Success		200	{object}	Task
//	@Failure		404	{object}	ErrorResponse
//	@Router			/tasks/{id} [get]
func (h *Handler) Get(c fiber.Ctx) error {
	task, ok := newFixtures(h.now()).task(c.Params("id"))
	if !ok {
		return notFound(c)
	}
	return c.JSON(task)
}

// Activity godoc
//
//	@Summary		List task activity
//	@Description	Returns the chatter timeline of a task, oldest first.
//	@ID				list-task-activity
//	@Tags			tasks
//	@Produce		json
//	@Param			id	path		string	true	"Task ID"
//	@Success		200	{object}	ActivityList
//	@Failure		404	{object}	ErrorResponse
//	@Router			/tasks/{id}/activity [get]
func (h *Handler) Activity(c fiber.Ctx) error {
	f := newFixtures(h.now())
	id := c.Params("id")
	if _, ok := f.task(id); !ok {
		return notFound(c)
	}
	items := f.activity[id]
	if items == nil {
		items = []Activity{}
	}
	return c.JSON(ActivityList{Items: items})
}

func notFound(c fiber.Ctx) error {
	return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{Message: "task not found"})
}
