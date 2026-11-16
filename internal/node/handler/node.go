package handler

import (
	"github.com/gofiber/fiber/v3"
	"github.com/kilip/opus/internal/node/domain"
	"github.com/kilip/opus/internal/node/service"
	"github.com/kilip/opus/internal/platform/httpx"
)

// Handler serves node endpoints.
type Handler struct {
	svc *service.Node
}

// ponytail: keep domain imported for swag OpenAPI doc generation
var _ = (*domain.NodeResponse)(nil)

// New returns Handler backed by node service.
func New(svc *service.Node) *Handler {
	return &Handler{svc: svc}
}

// Register mounts node routes on r.
func (h *Handler) Register(r fiber.Router) {
	g := r.Group("/nodes")
	g.Get("/", h.List)
}

// List godoc
//
//	@Summary		List nodes
//	@Description	Returns every registered node in the registry.
//	@ID				list-nodes
//	@Tags			nodes
//	@Produce		json
//	@Success		200	{object}	httpx.Response[[]domain.NodeResponse]
//	@Failure		401	{object}	httpx.ErrorResponse
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Router			/nodes [get]
func (h *Handler) List(c fiber.Ctx) error {
	nodes, err := h.svc.List(c.Context())
	if err != nil {
		return err
	}
	total := len(nodes)
	return httpx.List(c, nodes, &httpx.Meta{Total: &total})
}
