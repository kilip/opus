package handler

import (
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/kilip/opus/internal/platform/httpx"
	"github.com/kilip/opus/internal/run/domain"
	"github.com/kilip/opus/internal/run/service"
)

// Handler serves run REST endpoints.
type Handler struct {
	svc      *service.Service
	stageCfg service.StageConfig
}

// New creates a new run Handler.
func New(svc *service.Service, stageCfg service.StageConfig) *Handler {
	return &Handler{
		svc:      svc,
		stageCfg: stageCfg,
	}
}

// Register mounts protected run routes on r.
func (h *Handler) Register(r fiber.Router) {
	g := r.Group("/runs")
	g.Get("/", h.List)
	g.Get("/:id", h.Get)
	g.Post("/:id/cancel", h.Cancel)
}

// List godoc
//
//	@Summary		List runs
//	@Description	Returns runs filtered by optional taskId, projectId, and state.
//	@ID				list-runs
//	@Tags			runs
//	@Produce		json
//	@Param			taskId		query		string	false	"Filter by task ID"
//	@Param			projectId	query		string	false	"Filter by project ID"
//	@Param			state		query		string	false	"Filter by run state"
//	@Success		200			{object}	httpx.Response[[]domain.RunResponse]
//	@Failure		400			{object}	httpx.ErrorResponse
//	@Failure		500			{object}	httpx.ErrorResponse
//	@Router			/runs [get]
func (h *Handler) List(c fiber.Ctx) error {
	var filter domain.ListFilter
	if raw := c.Query("taskId"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			return httpx.BadRequest(err)
		}
		filter.TaskID = &id
	}
	if raw := c.Query("projectId"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			return httpx.BadRequest(err)
		}
		filter.ProjectID = &id
	}
	if raw := c.Query("state"); raw != "" {
		st := domain.RunState(raw)
		if !domain.IsValidState(st) {
			return httpx.BadRequest(domain.ErrInvalidState)
		}
		filter.State = &st
	}

	runs, err := h.svc.List(c.Context(), filter)
	if err != nil {
		return mapErr(err)
	}

	resp := make([]domain.RunResponse, len(runs))
	for i, r := range runs {
		resp[i] = domain.ToRunResponse(r)
	}
	total := len(resp)
	return httpx.List(c, resp, &httpx.Meta{Total: &total})
}

// Get godoc
//
//	@Summary		Get a run
//	@Description	Returns a single run by its identifier.
//	@ID				get-run
//	@Tags			runs
//	@Produce		json
//	@Param			id	path		string	true	"Run ID"
//	@Success		200	{object}	httpx.Response[domain.RunResponse]
//	@Failure		400	{object}	httpx.ErrorResponse
//	@Failure		404	{object}	httpx.ErrorResponse
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Router			/runs/{id} [get]
func (h *Handler) Get(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httpx.BadRequest(domain.ErrInvalidRunID)
	}
	run, err := h.svc.Get(c.Context(), id)
	if err != nil {
		return mapErr(err)
	}
	return httpx.OK(c, domain.ToRunResponse(run))
}

// Cancel godoc
//
//	@Summary		Cancel a run
//	@Description	Cancels an active run, stops its session, cleans up its worktree, and resets the task to todo.
//	@ID				cancel-run
//	@Tags			runs
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"Run ID"
//	@Param			request	body		domain.CancelRunRequest	false	"Optional cancellation reason"
//	@Success		200		{object}	httpx.Response[domain.RunResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		404		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Router			/runs/{id}/cancel [post]
func (h *Handler) Cancel(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httpx.BadRequest(domain.ErrInvalidRunID)
	}

	var req domain.CancelRunRequest
	if len(c.Body()) > 0 {
		if err := c.Bind().JSON(&req); err != nil {
			return httpx.BadRequest(err)
		}
	}

	run, err := h.svc.CancelRun(c.Context(), h.stageCfg, id, req.Reason)
	if err != nil {
		return mapErr(err)
	}
	return httpx.OK(c, domain.ToRunResponse(run))
}

func mapErr(err error) error {
	switch {
	case errors.Is(err, domain.ErrRunNotFound):
		return httpx.NotFound(err)
	case errors.Is(err, domain.ErrInvalidRunID),
		errors.Is(err, domain.ErrInvalidState),
		errors.Is(err, domain.ErrInvalidMode),
		errors.Is(err, domain.ErrInvalidTransition),
		errors.Is(err, domain.ErrTaskIDRequired),
		errors.Is(err, domain.ErrProjectIDRequired):
		return httpx.BadRequest(err)
	default:
		return err
	}
}
