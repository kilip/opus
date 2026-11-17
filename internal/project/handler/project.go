package handler

import (
	"encoding/json"
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/kilip/opus/internal/platform/httpx"
	"github.com/kilip/opus/internal/project/domain"
	"github.com/kilip/opus/internal/project/service"
)

// ponytail: keep domain imported for swag OpenAPI doc generation
var (
	_ = (*domain.Project)(nil)
	_ = (*domain.CreateProjectRequest)(nil)
	_ = (*domain.UpdateProjectRequest)(nil)
	_ = (*domain.ProjectNodeResponse)(nil)
	_ = (*domain.UpdateProjectNodePathRequest)(nil)
)

// Handler serves project endpoints.
type Handler struct {
	svc *service.Project
}

// New returns Handler backed by project service.
func New(svc *service.Project) *Handler {
	return &Handler{svc: svc}
}

// Register mounts project routes on r.
func (h *Handler) Register(r fiber.Router) {
	g := r.Group("/projects")
	g.Get("/", h.List)
	g.Post("/", h.Create)
	g.Get("/:id", h.Get)
	g.Patch("/:id", h.Update)
	g.Delete("/:id", h.Delete)
	g.Get("/:id/nodes", h.ListNodeMappings)
	g.Put("/:id/nodes/:nodeId", h.SetNodePath)
}

// List godoc
//
//	@Summary		List projects
//	@Description	Returns every project ordered by ID.
//	@ID				list-projects
//	@Tags			projects
//	@Produce		json
//	@Success		200	{object}	httpx.Response[[]domain.Project]
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Router			/projects [get]
func (h *Handler) List(c fiber.Ctx) error {
	projects, err := h.svc.List(c.Context())
	if err != nil {
		return mapErr(err)
	}
	total := len(projects)
	return httpx.List(c, projects, &httpx.Meta{Total: &total})
}

// Create godoc
//
//	@Summary		Create project
//	@Description	Creates a new project.
//	@ID				create-project
//	@Tags			projects
//	@Accept			json
//	@Produce		json
//	@Param			request	body		domain.CreateProjectRequest	true	"Create Project Request"
//	@Success		201		{object}	httpx.Response[domain.Project]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		409		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Router			/projects [post]
func (h *Handler) Create(c fiber.Ctx) error {
	var req domain.CreateProjectRequest
	if err := c.Bind().Body(&req); err != nil {
		return httpx.BadRequest(err)
	}
	p, err := h.svc.Create(c.Context(), req)
	if err != nil {
		return mapErr(err)
	}
	return httpx.Created(c, p)
}

// Get godoc
//
//	@Summary		Get project
//	@Description	Returns a single project by ID.
//	@ID				get-project
//	@Tags			projects
//	@Produce		json
//	@Param			id	path		string	true	"Project ID"
//	@Success		200	{object}	httpx.Response[domain.Project]
//	@Failure		404	{object}	httpx.ErrorResponse
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Router			/projects/{id} [get]
func (h *Handler) Get(c fiber.Ctx) error {
	p, err := h.svc.Get(c.Context(), c.Params("id"))
	if err != nil {
		return mapErr(err)
	}
	return httpx.OK(c, p)
}

// Update godoc
//
//	@Summary		Update project
//	@Description	Updates fields of an existing project.
//	@ID				update-project
//	@Tags			projects
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string						true	"Project ID"
//	@Param			request	body		domain.UpdateProjectRequest	true	"Update Project Request"
//	@Success		200		{object}	httpx.Response[domain.Project]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		404		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Router			/projects/{id} [patch]
func (h *Handler) Update(c fiber.Ctx) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(c.Body(), &raw); err != nil {
		return httpx.BadRequest(err)
	}
	if _, ok := raw["slug"]; ok {
		return httpx.BadRequest(domain.ErrPatchForbidden)
	}
	if _, ok := raw["repo_ref"]; ok {
		return httpx.BadRequest(domain.ErrPatchForbidden)
	}
	if _, ok := raw["repoRef"]; ok {
		return httpx.BadRequest(domain.ErrPatchForbidden)
	}

	var req domain.UpdateProjectRequest
	if err := c.Bind().Body(&req); err != nil {
		return httpx.BadRequest(err)
	}
	p, err := h.svc.Update(c.Context(), c.Params("id"), req)
	if err != nil {
		return mapErr(err)
	}
	return httpx.OK(c, p)
}

// Delete godoc
//
//	@Summary		Delete project
//	@Description	Deletes a project by ID.
//	@ID				delete-project
//	@Tags			projects
//	@Param			id	path		string	true	"Project ID"
//	@Success		204	"No Content"
//	@Failure		404	{object}	httpx.ErrorResponse
//	@Failure		409	{object}	httpx.ErrorResponse
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Router			/projects/{id} [delete]
func (h *Handler) Delete(c fiber.Ctx) error {
	if err := h.svc.Delete(c.Context(), c.Params("id")); err != nil {
		return mapErr(err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// ListNodeMappings godoc
//
//	@Summary		List project node mappings
//	@Description	Returns node path mappings for all known nodes for the project.
//	@ID				list-project-node-mappings
//	@Tags			projects
//	@Produce		json
//	@Param			id	path		string	true	"Project ID"
//	@Success		200	{object}	httpx.Response[[]domain.ProjectNodeResponse]
//	@Failure		400	{object}	httpx.ErrorResponse
//	@Failure		404	{object}	httpx.ErrorResponse
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Router			/projects/{id}/nodes [get]
func (h *Handler) ListNodeMappings(c fiber.Ctx) error {
	id := c.Params("id")
	if _, err := uuid.Parse(id); err != nil {
		return httpx.BadRequest(err)
	}
	mappings, err := h.svc.ListNodeMappings(c.Context(), id)
	if err != nil {
		return mapErr(err)
	}
	total := len(mappings)
	return httpx.List(c, mappings, &httpx.Meta{Total: &total})
}

// SetNodePath godoc
//
//	@Summary		Set project node path mapping
//	@Description	Upserts a path mapping for a node in the project.
//	@ID				set-project-node-path
//	@Tags			projects
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string								true	"Project ID"
//	@Param			nodeId	path		string								true	"Node ID"
//	@Param			request	body		domain.UpdateProjectNodePathRequest	true	"Node Path Request"
//	@Success		200		{object}	httpx.Response[domain.ProjectNodeResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		404		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Router			/projects/{id}/nodes/{nodeId} [put]
func (h *Handler) SetNodePath(c fiber.Ctx) error {
	var req domain.UpdateProjectNodePathRequest
	if len(c.Body()) > 0 {
		if err := c.Bind().Body(&req); err != nil {
			return httpx.BadRequest(err)
		}
	}
	resp, err := h.svc.SetNodePath(c.Context(), c.Params("id"), c.Params("nodeId"), req.Path)
	if err != nil {
		return mapErr(err)
	}
	return httpx.OK(c, resp)
}

// mapErr maps domain sentinel errors to httpx status errors.
func mapErr(err error) error {
	switch {
	case errors.Is(err, domain.ErrProjectNotFound),
		errors.Is(err, domain.ErrNodeNotFound):
		return httpx.NotFound(err)
	case errors.Is(err, domain.ErrInvalidSlug),
		errors.Is(err, domain.ErrInvalidRepo),
		errors.Is(err, domain.ErrPatchForbidden),
		errors.Is(err, domain.ErrInvalidPath):
		return httpx.BadRequest(err)
	case errors.Is(err, domain.ErrDuplicateSlug),
		errors.Is(err, domain.ErrProjectHasTasks):
		return httpx.Conflict(err)
	default:
		return err
	}
}
