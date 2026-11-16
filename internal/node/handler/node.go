package handler

import (
	"errors"
	"strings"
	"time"

	"github.com/gofiber/contrib/v3/websocket"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/kilip/opus/internal/node/domain"
	"github.com/kilip/opus/internal/node/service"
	"github.com/kilip/opus/internal/platform/httpx"
)

// ponytail: in-memory IP rate limiter at 10 req/min; upgrade to Redis/distributed store when cluster scales
const (
	defaultJoinMaxIPRequests = 10
	defaultJoinWindow        = 1 * time.Minute
)

// RateLimitConfig configures join request rate limits.
type RateLimitConfig struct {
	MaxIPRequests int
	Window        time.Duration
	Disabled      bool
}

// clientIP extracts the client IP address from proxy headers or connection.
// ponytail: direct proxy header check; configure Fiber TrustProxy when behind strict edge proxy
func clientIP(c fiber.Ctx) string {
	if ip := c.Get(fiber.HeaderXForwardedFor); ip != "" {
		if comma := strings.Index(ip, ","); comma != -1 {
			return strings.TrimSpace(ip[:comma])
		}
		return strings.TrimSpace(ip)
	}
	if ip := c.Get("X-Real-IP"); ip != "" {
		return strings.TrimSpace(ip)
	}
	return c.IP()
}

func newJoinLimiter(cfg RateLimitConfig) fiber.Handler {
	if cfg.Disabled {
		return func(c fiber.Ctx) error { return c.Next() }
	}
	maxRequests := cfg.MaxIPRequests
	if maxRequests <= 0 {
		maxRequests = defaultJoinMaxIPRequests
	}
	window := cfg.Window
	if window <= 0 {
		window = defaultJoinWindow
	}

	return limiter.New(limiter.Config{
		Max:        maxRequests,
		Expiration: window,
		KeyGenerator: func(c fiber.Ctx) string {
			return clientIP(c)
		},
		LimitReached: func(c fiber.Ctx) error {
			return httpx.NewError(fiber.StatusTooManyRequests, domain.ErrRateLimited)
		},
	})
}

// Handler serves node endpoints.
type Handler struct {
	svc       *service.Node
	hub       *service.Hub
	limiter   fiber.Handler
	wsHandler fiber.Handler
}

// ponytail: keep domain imported for swag OpenAPI doc generation
var (
	_ = (*domain.NodeResponse)(nil)
	_ = (*domain.JoinTokenResponse)(nil)
	_ = (*domain.JoinNodeRequest)(nil)
	_ = (*domain.JoinNodeResponse)(nil)
)

// New returns Handler backed by node service and hub.
func New(svc *service.Node, hub *service.Hub, rl ...RateLimitConfig) *Handler {
	var cfg RateLimitConfig
	if len(rl) > 0 {
		cfg = rl[0]
	}
	if hub == nil {
		hub = service.NewHub()
	}
	h := &Handler{
		svc:     svc,
		hub:     hub,
		limiter: newJoinLimiter(cfg),
	}
	h.wsHandler = websocket.New(h.handleWS, websocket.Config{
		Origins:          []string{"*"},
		AllowEmptyOrigin: true,
	})
	return h
}

// Register mounts node routes on r.
func (h *Handler) Register(r fiber.Router) {
	g := r.Group("/nodes")
	g.Get("/", h.List)
	g.Post("/join-tokens", h.CreateJoinToken)
	g.Delete("/:id", h.Delete)
}

// RegisterPublic mounts unauthenticated node routes on r.
func (h *Handler) RegisterPublic(r fiber.Router) {
	g := r.Group("/nodes")
	g.Post("/join", h.limiter, h.Join)
	g.Get("/connect", h.Connect)
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

// Delete godoc
//
//	@Summary		Delete node
//	@Description	Deletes a worker node and terminates any active control channel connection.
//	@ID				delete-node
//	@Tags			nodes
//	@Produce		json
//	@Param			id	path		string	true	"Node ID"
//	@Success		204	"No Content"
//	@Failure		401	{object}	httpx.ErrorResponse
//	@Failure		404	{object}	httpx.ErrorResponse
//	@Failure		409	{object}	httpx.ErrorResponse
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Router			/nodes/{id} [delete]
func (h *Handler) Delete(c fiber.Ctx) error {
	id := c.Params("id")
	if err := h.svc.Delete(c.Context(), id); err != nil {
		return mapErr(err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// Join godoc
//
//	@Summary		Join node
//	@Description	Exchanges a valid join token and node name for node credentials.
//	@ID				join-node
//	@Tags			nodes
//	@Accept			json
//	@Produce		json
//	@Param			request	body		domain.JoinNodeRequest	true	"Join node request"
//	@Success		201		{object}	httpx.Response[domain.JoinNodeResponse]
//	@Failure		400		{object}	httpx.ErrorResponse	"Invalid or empty node name"
//	@Failure		401		{object}	httpx.ErrorResponse	"Expired, used, or unknown join token"
//	@Failure		409		{object}	httpx.ErrorResponse	"Node name already in use"
//	@Failure		429		{object}	httpx.ErrorResponse	"Rate limit exceeded"
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Router			/nodes/join [post]
func (h *Handler) Join(c fiber.Ctx) error {
	var req domain.JoinNodeRequest
	if err := c.Bind().Body(&req); err != nil {
		return httpx.BadRequest(err)
	}
	res, err := h.svc.Join(c.Context(), req)
	if err != nil {
		return mapErr(err)
	}
	return httpx.Created(c, res)
}

// Connect godoc
//
//	@Summary		Connect worker node control channel
//	@Description	Upgrades HTTP connection to WebSocket control channel for worker nodes.
//	@ID				connect-node
//	@Tags			nodes
//	@Success		101	"Switching Protocols"
//	@Failure		401	{object}	httpx.ErrorResponse	"Missing or invalid credentials"
//	@Failure		403	{object}	httpx.ErrorResponse	"Master node connection forbidden"
//	@Failure		426	{object}	httpx.ErrorResponse	"Upgrade required"
//	@Router			/nodes/connect [get]
func (h *Handler) Connect(c fiber.Ctx) error {
	authHeader := c.Get(fiber.HeaderAuthorization)
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
		return httpx.NewError(fiber.StatusUnauthorized, domain.ErrUnauthorized)
	}
	cred := strings.TrimSpace(parts[1])

	node, err := h.svc.AuthenticateWorker(c.Context(), cred)
	if err != nil {
		return mapErr(err)
	}

	c.Locals("nodeID", node.ID)
	return h.wsHandler(c)
}

func (h *Handler) handleWS(c *websocket.Conn) {
	nodeID, ok := c.Locals("nodeID").(string)
	if !ok || nodeID == "" {
		_ = c.Close()
		return
	}

	cleanup, err := h.hub.Register(nodeID, c)
	if err != nil {
		_ = c.Close()
		return
	}
	defer cleanup()

	for {
		_, msg, err := c.ReadMessage()
		if err != nil {
			break
		}
		if err := h.hub.HandleMessage(nodeID, msg); err != nil {
			// Unknown envelope types or unmarshal errors rejected without closing
			continue
		}
	}
}

// mapErr maps domain sentinel errors to httpx status errors.
func mapErr(err error) error {
	switch {
	case errors.Is(err, domain.ErrTokenNotFound),
		errors.Is(err, domain.ErrTokenExpired),
		errors.Is(err, domain.ErrTokenAlreadyUsed),
		errors.Is(err, domain.ErrUnauthorized):
		return httpx.NewError(fiber.StatusUnauthorized, err)
	case errors.Is(err, domain.ErrMasterConnectForbidden):
		return httpx.NewError(fiber.StatusForbidden, err)
	case errors.Is(err, domain.ErrInvalidNodeName):
		return httpx.BadRequest(err)
	case errors.Is(err, domain.ErrNodeNameConflict),
		errors.Is(err, domain.ErrMasterDeleteForbidden):
		return httpx.Conflict(err)
	case errors.Is(err, domain.ErrRateLimited):
		return httpx.NewError(fiber.StatusTooManyRequests, err)
	case errors.Is(err, domain.ErrNodeNotConnected),
		errors.Is(err, domain.ErrNodeNotFound):
		return httpx.NewError(fiber.StatusNotFound, err)
	default:
		return err
	}
}
