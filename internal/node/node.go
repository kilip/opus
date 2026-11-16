// Package node is the node registry feature. Other packages depend on this package only,
// never on its subpackages.
package node

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/opus/ent"
	"github.com/kilip/opus/internal/node/domain"
	"github.com/kilip/opus/internal/node/handler"
	"github.com/kilip/opus/internal/node/repo"
	"github.com/kilip/opus/internal/node/service"
)

// Event type constants re-exported from domain.
const (
	EventOnline  = domain.EventOnline
	EventOffline = domain.EventOffline

	RoleMaster = domain.RoleMaster
	RoleWorker = domain.RoleWorker

	StatusOnline  = domain.StatusOnline
	StatusOffline = domain.StatusOffline
)

// Sentinel errors re-exported from domain.
var (
	ErrTokenNotFound    = domain.ErrTokenNotFound
	ErrTokenExpired     = domain.ErrTokenExpired
	ErrTokenAlreadyUsed = domain.ErrTokenAlreadyUsed
	ErrNodeNameConflict = domain.ErrNodeNameConflict
	ErrInvalidNodeName  = domain.ErrInvalidNodeName
	ErrRateLimited      = domain.ErrRateLimited
)

// Re-exported domain types.
type (
	Role              = domain.Role
	Status            = domain.Status
	NodeEventPayload  = domain.NodeEventPayload
	NodeResponse      = domain.NodeResponse
	JoinTokenResponse = domain.JoinTokenResponse
	JoinNodeRequest   = domain.JoinNodeRequest
	JoinNodeResponse  = domain.JoinNodeResponse
	RateLimitConfig   = handler.RateLimitConfig
)

// Config holds configuration parameters for the node feature.
type Config struct {
	BaseURL         string
	Now             func() time.Time
	RateLimitConfig handler.RateLimitConfig
}

// Node is the node feature facade.
type Node struct {
	svc     *service.Node
	handler *handler.Handler
}

// New returns a new Node feature facade backed by Ent.
func New(client *ent.Client, cfg Config) *Node {
	r := repo.New(client)
	svc := service.New(r, service.Config{
		BaseURL: cfg.BaseURL,
		Now:     cfg.Now,
	})
	return &Node{
		svc:     svc,
		handler: handler.New(svc, cfg.RateLimitConfig),
	}
}

// Bootstrap registers the master node and resets workers to offline.
func (n *Node) Bootstrap(ctx context.Context) error {
	return n.svc.Bootstrap(ctx)
}

// GenerateJoinToken creates a new join token.
func (n *Node) GenerateJoinToken(ctx context.Context) (domain.JoinTokenResponse, error) {
	return n.svc.GenerateJoinToken(ctx)
}

// ConsumeJoinToken validates and consumes a join token.
func (n *Node) ConsumeJoinToken(ctx context.Context, token string) error {
	return n.svc.ConsumeJoinToken(ctx, token)
}

// Join exchanges a join token and node name for credentials.
func (n *Node) Join(ctx context.Context, req domain.JoinNodeRequest) (domain.JoinNodeResponse, error) {
	return n.svc.Join(ctx, req)
}

// Register mounts the protected node HTTP routes on r.
func (n *Node) Register(r fiber.Router) {
	n.handler.Register(r)
}

// RegisterPublic mounts unauthenticated node HTTP routes on r.
func (n *Node) RegisterPublic(r fiber.Router) {
	n.handler.RegisterPublic(r)
}
