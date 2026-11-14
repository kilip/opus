package handler

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/opus/internal/platform/httpx"
	"github.com/kilip/opus/internal/task/domain"
	"github.com/kilip/opus/internal/task/repo"
)

// Handler serves task endpoints.
type Handler struct {
	repo *repo.Memory
}

// New returns Handler backed by Memory repo.
func New(r *repo.Memory) *Handler {
	return &Handler{repo: r}
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
//	@Success		200	{object}	httpx.Response[[]Task]
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Router			/tasks [get]
func (h *Handler) List(c fiber.Ctx) error {
	tasks := h.repo.List()
	items := make([]Task, len(tasks))
	for i, t := range tasks {
		items[i] = toTaskDTO(t)
	}
	total := len(items)
	return httpx.List(c, items, &httpx.Meta{Total: &total})
}

// Create godoc
//
//	@Summary		Create task
//	@Description	Creates a new task with given details.
//	@ID				create-task
//	@Tags			tasks
//	@Accept			json
//	@Produce		json
//	@Param			request	body		CreateTaskRequest	true	"Create Task Request"
//	@Success		201		{object}	httpx.Response[Task]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Router			/tasks [post]
func (h *Handler) Create(c fiber.Ctx) error {
	var req CreateTaskRequest
	if err := c.Bind().Body(&req); err != nil {
		return httpx.BadRequest(err)
	}

	if strings.TrimSpace(req.Title) == "" {
		return httpx.BadRequest(domain.ErrTaskTitleEmpty)
	}

	var status *domain.TaskStatus
	if req.Status != nil {
		s := domain.TaskStatus(*req.Status)
		status = &s
	}

	var priority *domain.TaskPriority
	if req.Priority != nil {
		p := domain.TaskPriority(*req.Priority)
		priority = &p
	}

	var kind *domain.TaskType
	if req.Type != nil {
		t := domain.TaskType(*req.Type)
		kind = &t
	}

	var assignee *domain.Assignee
	if req.Assignee != nil {
		assignee = &domain.Assignee{
			ID:   req.Assignee.ID,
			Name: req.Assignee.Name,
		}
	}

	created := h.repo.Create(repo.CreateInput{
		Title:       req.Title,
		Description: req.Description,
		Status:      status,
		Priority:    priority,
		Type:        kind,
		Assignee:    assignee,
		DueDate:     req.DueDate,
		Labels:      req.Labels,
	})

	return httpx.Created(c, toTaskDTO(created))
}

// Get godoc
//
//	@Summary		Get a task
//	@Description	Returns a single task by its Opus identifier.
//	@ID				get-task
//	@Tags			tasks
//	@Produce		json
//	@Param			id	path		string	true	"Task ID"
//	@Success		200	{object}	httpx.Response[Task]
//	@Failure		404	{object}	httpx.ErrorResponse
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Router			/tasks/{id} [get]
func (h *Handler) Get(c fiber.Ctx) error {
	t, err := h.repo.Get(c.Params("id"))
	if err != nil {
		if errors.Is(err, domain.ErrTaskNotFound) {
			return httpx.NotFound(domain.ErrTaskNotFound)
		}
		return err
	}
	return httpx.OK(c, toTaskDTO(t))
}

// Update godoc
//
//	@Summary		Update task
//	@Description	Updates fields of an existing task.
//	@ID				update-task
//	@Tags			tasks
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string				true	"Task ID"
//	@Param			request	body		UpdateTaskRequest	true	"Update Task Request"
//	@Success		200		{object}	httpx.Response[Task]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		404		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Router			/tasks/{id} [patch]
func (h *Handler) Update(c fiber.Ctx) error {
	var req UpdateTaskRequest
	if err := c.Bind().Body(&req); err != nil {
		return httpx.BadRequest(err)
	}

	if req.Title != nil && strings.TrimSpace(*req.Title) == "" {
		return httpx.BadRequest(domain.ErrTaskTitleEmpty)
	}

	var status *domain.TaskStatus
	if req.Status != nil {
		s := domain.TaskStatus(*req.Status)
		status = &s
	}

	var priority *domain.TaskPriority
	if req.Priority != nil {
		p := domain.TaskPriority(*req.Priority)
		priority = &p
	}

	var kind *domain.TaskType
	if req.Type != nil {
		t := domain.TaskType(*req.Type)
		kind = &t
	}

	var assignee *domain.Assignee
	if req.Assignee != nil {
		assignee = &domain.Assignee{
			ID:   req.Assignee.ID,
			Name: req.Assignee.Name,
		}
	}

	updated, err := h.repo.Update(c.Params("id"), repo.UpdateInput{
		Title:         req.Title,
		Description:   req.Description,
		Status:        status,
		Priority:      priority,
		Type:          kind,
		Assignee:      assignee,
		ClearAssignee: req.ClearAssignee,
		DueDate:       req.DueDate,
		Labels:        req.Labels,
	})
	if err != nil {
		if errors.Is(err, domain.ErrTaskNotFound) {
			return httpx.NotFound(domain.ErrTaskNotFound)
		}
		return err
	}

	return httpx.OK(c, toTaskDTO(updated))
}

// ListActivity godoc
//
//	@Summary		List task activity
//	@Description	Returns chatter timeline of a task, oldest first.
//	@ID				list-task-activity
//	@Tags			tasks
//	@Produce		json
//	@Param			id	path		string	true	"Task ID"
//	@Success		200	{object}	httpx.Response[[]Activity]
//	@Failure		404	{object}	httpx.ErrorResponse
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Router			/tasks/{id}/activity [get]
func (h *Handler) ListActivity(c fiber.Ctx) error {
	acts, err := h.repo.ListActivity(c.Params("id"))
	if err != nil {
		if errors.Is(err, domain.ErrTaskNotFound) {
			return httpx.NotFound(domain.ErrTaskNotFound)
		}
		return err
	}

	items := make([]Activity, len(acts))
	for i, a := range acts {
		items[i] = toActivityDTO(a)
	}
	total := len(items)
	return httpx.List(c, items, &httpx.Meta{Total: &total})
}

// CreateActivity godoc
//
//	@Summary		Create task activity
//	@Description	Adds an activity comment/event to task chatter timeline.
//	@ID				create-task-activity
//	@Tags			tasks
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"Task ID"
//	@Param			request	body		CreateActivityRequest	true	"Create Activity Request"
//	@Success		201		{object}	httpx.Response[Activity]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		404		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Router			/tasks/{id}/activity [post]
func (h *Handler) CreateActivity(c fiber.Ctx) error {
	var req CreateActivityRequest
	if err := c.Bind().Body(&req); err != nil {
		return httpx.BadRequest(err)
	}

	if strings.TrimSpace(req.Body) == "" {
		return httpx.BadRequest(domain.ErrActivityBodyEmpty)
	}

	var kind domain.ActivityKind
	if req.Kind != nil {
		kind = domain.ActivityKind(*req.Kind)
	}

	act, err := h.repo.AddActivity(c.Params("id"), kind, req.Author, req.Body)
	if err != nil {
		if errors.Is(err, domain.ErrTaskNotFound) {
			return httpx.NotFound(domain.ErrTaskNotFound)
		}
		return err
	}

	return httpx.Created(c, toActivityDTO(act))
}

func toTaskDTO(t domain.Task) Task {
	dto := Task{
		ID:           t.ID,
		Key:          t.Key,
		Source:       TaskSource(t.Source),
		ExternalID:   t.ExternalID,
		ExternalURL:  t.ExternalURL,
		Title:        t.Title,
		Description:  t.Description,
		Status:       TaskStatus(t.Status),
		SourceStatus: t.SourceStatus,
		Priority:     TaskPriority(t.Priority),
		Type:         TaskType(t.Type),
		DueDate:      t.DueDate,
		Labels:       t.Labels,
		CreatedAt:    t.CreatedAt,
		UpdatedAt:    t.UpdatedAt,
	}
	if t.Assignee != nil {
		dto.Assignee = &Assignee{
			ID:   t.Assignee.ID,
			Name: t.Assignee.Name,
		}
	}
	return dto
}

func toActivityDTO(a domain.Activity) Activity {
	return Activity{
		ID:        a.ID,
		Kind:      ActivityKind(a.Kind),
		Author:    a.Author,
		Body:      a.Body,
		CreatedAt: a.CreatedAt,
	}
}
