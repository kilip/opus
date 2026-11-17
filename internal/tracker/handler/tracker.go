package handler

import (
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/opus/internal/platform/httpx"
	"github.com/kilip/opus/internal/tracker/domain"
	"github.com/kilip/opus/internal/tracker/service"
)

// ponytail: keep domain imported for swag OpenAPI doc generation
var (
	_ = (*domain.TrackerStatusResponse)(nil)
	_ = (*domain.TrackerSettingsResponse)(nil)
	_ = (*domain.UpdateTrackerSettingsRequest)(nil)
)

// Handler serves tracker endpoints.
type Handler struct {
	svc *service.Tracker
}

// New returns Handler backed by tracker service.
func New(svc *service.Tracker) *Handler {
	return &Handler{svc: svc}
}

// Register mounts tracker routes on r.
func (h *Handler) Register(r fiber.Router) {
	r.Get("/projects/:id/tracker", h.GetProjectStatus)
	g := r.Group("/tracker")
	g.Get("/settings", h.GetSettings)
	g.Put("/settings", h.UpdateSettings)
}

// GetProjectStatus godoc
//
//	@Summary		Get project tracker status
//	@Description	Returns the last sync timestamp and last error for a project's tracker.
//	@ID				get-project-tracker-status
//	@Tags			tracker
//	@Produce		json
//	@Param			id	path		string	true	"Project ID"
//	@Success		200	{object}	httpx.Response[domain.TrackerStatusResponse]
//	@Failure		400	{object}	httpx.ErrorResponse
//	@Failure		404	{object}	httpx.ErrorResponse
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Router			/projects/{id}/tracker [get]
func (h *Handler) GetProjectStatus(c fiber.Ctx) error {
	resp, err := h.svc.GetProjectStatus(c.Context(), c.Params("id"))
	if err != nil {
		return mapErr(err)
	}
	return httpx.OK(c, resp)
}

// GetSettings godoc
//
//	@Summary		Get tracker settings
//	@Description	Returns global tracker status-to-label mappings with defaults filled.
//	@ID				get-tracker-settings
//	@Tags			tracker
//	@Produce		json
//	@Success		200	{object}	httpx.Response[domain.TrackerSettingsResponse]
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Router			/tracker/settings [get]
func (h *Handler) GetSettings(c fiber.Ctx) error {
	resp, err := h.svc.GetSettings(c.Context())
	if err != nil {
		return mapErr(err)
	}
	return httpx.OK(c, resp)
}

// UpdateSettings godoc
//
//	@Summary		Update tracker settings
//	@Description	Replaces global tracker status-to-label overrides.
//	@ID				update-tracker-settings
//	@Tags			tracker
//	@Accept			json
//	@Produce		json
//	@Param			request	body		domain.UpdateTrackerSettingsRequest	true	"Update Tracker Settings Request"
//	@Success		200		{object}	httpx.Response[domain.TrackerSettingsResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Router			/tracker/settings [put]
func (h *Handler) UpdateSettings(c fiber.Ctx) error {
	var req domain.UpdateTrackerSettingsRequest
	if err := c.Bind().Body(&req); err != nil {
		return httpx.BadRequest(err)
	}
	resp, err := h.svc.UpdateSettings(c.Context(), req)
	if err != nil {
		return mapErr(err)
	}
	return httpx.OK(c, resp)
}

func mapErr(err error) error {
	switch {
	case errors.Is(err, domain.ErrInvalidProjectID),
		errors.Is(err, domain.ErrInvalidLabelConfig):
		return httpx.BadRequest(err)
	case errors.Is(err, domain.ErrProjectNotFound):
		return httpx.NotFound(err)
	default:
		return err
	}
}
