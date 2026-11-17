package handler

import (
	"encoding/json"
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/kilip/opus/internal/platform/httpx"
	"github.com/kilip/opus/internal/task/domain"
	"github.com/kilip/opus/internal/task/service"
)

// Handler serves task endpoints.
type Handler struct {
	svc *service.Task
}

// New returns Handler backed by task service.
func New(svc *service.Task) *Handler {
	return &Handler{svc: svc}
}

// Register mounts task routes on r.
func (h *Handler) Register(r fiber.Router) {
	g := r.Group("/tasks")
	g.Get("/", h.List)
	g.Post("/", h.Create)
	g.Get("/:id", h.Get)
	g.Patch("/:id", h.Update)
	g.Get("/:id/activity", h.ListActivity)
	g.Post("/:id/activity", h.CreateActivity)
}

// List godoc
//
//	@Summary		List tasks
//	@Description	Returns every task, normalised from its source tracker.
//	@ID				list-tasks
//	@Tags			tasks
//	@Produce		json
//	@Param			projectId	query		string	false	"Filter tasks by project ID"
//	@Success		200			{object}	httpx.Response[[]domain.Task]
//	@Failure		400			{object}	httpx.ErrorResponse
//	@Failure		500			{object}	httpx.ErrorResponse
//	@Router			/tasks [get]
func (h *Handler) List(c fiber.Ctx) error {
	projectID := c.Query("projectId")
	if projectID != "" {
		if _, err := uuid.Parse(projectID); err != nil {
			return httpx.BadRequest(domain.ErrInvalidProjectID)
		}
	}
	tasks, err := h.svc.List(c.Context(), domain.TaskFilter{ProjectID: projectID})
	if err != nil {
		return mapErr(err)
	}
	total := len(tasks)
	return httpx.List(c, tasks, &httpx.Meta{Total: &total})
}

// Create godoc
//
//	@Summary		Create task
//	@Description	Creates a new task with given details.
//	@ID				create-task
//	@Tags			tasks
//	@Accept			json
//	@Produce		json
//	@Param			request	body		domain.CreateTaskRequest	true	"Create Task Request"
//	@Success		201		{object}	httpx.Response[domain.Task]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		404		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Router			/tasks [post]
func (h *Handler) Create(c fiber.Ctx) error {
	var req domain.CreateTaskRequest
	if err := c.Bind().Body(&req); err != nil {
		return httpx.BadRequest(err)
	}
	t, err := h.svc.Create(c.Context(), req)
	if err != nil {
		return mapErr(err)
	}
	return httpx.Created(c, t)
}

// Get godoc
//
//	@Summary		Get a task
//	@Description	Returns a single task by its Opus identifier.
//	@ID				get-task
//	@Tags			tasks
//	@Produce		json
//	@Param			id	path		string	true	"Task ID"
//	@Success		200	{object}	httpx.Response[domain.Task]
//	@Failure		404	{object}	httpx.ErrorResponse
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Router			/tasks/{id} [get]
func (h *Handler) Get(c fiber.Ctx) error {
	t, err := h.svc.Get(c.Context(), c.Params("id"))
	if err != nil {
		return mapErr(err)
	}
	return httpx.OK(c, t)
}

// Update godoc
//
//	@Summary		Update task
//	@Description	Updates fields of an existing task.
//	@ID				update-task
//	@Tags			tasks
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string						true	"Task ID"
//	@Param			request	body		domain.UpdateTaskRequest	true	"Update Task Request"
//	@Success		200		{object}	httpx.Response[domain.Task]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		404		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Router			/tasks/{id} [patch]
func (h *Handler) Update(c fiber.Ctx) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(c.Body(), &raw); err != nil {
		return httpx.BadRequest(err)
	}
	if _, ok := raw["projectId"]; ok {
		return httpx.BadRequest(domain.ErrPatchForbidden)
	}
	if _, ok := raw["project_id"]; ok {
		return httpx.BadRequest(domain.ErrPatchForbidden)
	}

	var req domain.UpdateTaskRequest
	if err := c.Bind().Body(&req); err != nil {
		return httpx.BadRequest(err)
	}
	t, err := h.svc.Update(c.Context(), c.Params("id"), req)
	if err != nil {
		return mapErr(err)
	}
	return httpx.OK(c, t)
}

// ListActivity godoc
//
//	@Summary		List task activity
//	@Description	Returns chatter timeline of a task, oldest first.
//	@ID				list-task-activity
//	@Tags			tasks
//	@Produce		json
//	@Param			id	path		string	true	"Task ID"
//	@Success		200	{object}	httpx.Response[[]domain.Activity]
//	@Failure		404	{object}	httpx.ErrorResponse
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Router			/tasks/{id}/activity [get]
func (h *Handler) ListActivity(c fiber.Ctx) error {
	acts, err := h.svc.ListActivity(c.Context(), c.Params("id"))
	if err != nil {
		return mapErr(err)
	}
	total := len(acts)
	return httpx.List(c, acts, &httpx.Meta{Total: &total})
}

// CreateActivity godoc
//
//	@Summary		Create task activity
//	@Description	Adds an activity comment/event to task chatter timeline.
//	@ID				create-task-activity
//	@Tags			tasks
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string							true	"Task ID"
//	@Param			request	body		domain.CreateActivityRequest	true	"Create Activity Request"
//	@Success		201		{object}	httpx.Response[domain.Activity]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		404		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Router			/tasks/{id}/activity [post]
func (h *Handler) CreateActivity(c fiber.Ctx) error {
	var req domain.CreateActivityRequest
	if err := c.Bind().Body(&req); err != nil {
		return httpx.BadRequest(err)
	}
	a, err := h.svc.CreateActivity(c.Context(), c.Params("id"), req)
	if err != nil {
		return mapErr(err)
	}
	return httpx.Created(c, a)
}

// mapErr maps domain sentinel errors to httpx status errors.
func mapErr(err error) error {
	switch {
	case errors.Is(err, domain.ErrTaskNotFound), errors.Is(err, domain.ErrProjectNotFound):
		return httpx.NotFound(err)
	case errors.Is(err, domain.ErrTaskTitleEmpty),
		errors.Is(err, domain.ErrActivityBodyEmpty),
		errors.Is(err, domain.ErrProjectIDRequired),
		errors.Is(err, domain.ErrInvalidProjectID),
		errors.Is(err, domain.ErrPatchForbidden):
		return httpx.BadRequest(err)
	default:
		return err
	}
}
