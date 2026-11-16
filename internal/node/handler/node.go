package handler

import (
	"errors"

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
var (
	_ = (*domain.NodeResponse)(nil)
	_ = (*domain.JoinTokenResponse)(nil)
)

// New returns Handler backed by node service.
func New(svc *service.Node) *Handler {
	return &Handler{svc: svc}
}

// Register mounts node routes on r.
func (h *Handler) Register(r fiber.Router) {
	g := r.Group("/nodes")
	g.Get("/", h.List)
	g.Post("/join-tokens", h.CreateJoinToken)
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
		return mapErr(err)
	}
	total := len(nodes)
	return httpx.List(c, nodes, &httpx.Meta{Total: &total})
}

// CreateJoinToken godoc
//
//	@Summary		Create node join token
//	@Description	Generates a single-use join token URL valid for 15 minutes.
//	@ID				create-node-join-token
//	@Tags			nodes
//	@Produce		json
//	@Success		201	{object}	httpx.Response[domain.JoinTokenResponse]
//	@Failure		401	{object}	httpx.ErrorResponse
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Router			/nodes/join-tokens [post]
func (h *Handler) CreateJoinToken(c fiber.Ctx) error {
	res, err := h.svc.GenerateJoinToken(c.Context())
	if err != nil {
		return mapErr(err)
	}
	return httpx.Created(c, res)
}

// mapErr maps domain sentinel errors to httpx status errors.
func mapErr(err error) error {
	switch {
	case errors.Is(err, domain.ErrTokenNotFound):
		return httpx.NotFound(err)
	case errors.Is(err, domain.ErrTokenExpired), errors.Is(err, domain.ErrTokenAlreadyUsed):
		return httpx.BadRequest(err)
	default:
		return err
	}
}
