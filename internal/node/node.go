// Package node is the node registry feature. Other packages depend on this package only,
// never on its subpackages.
package node

import (
	"context"

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

// Re-exported domain types.
type (
	Role             = domain.Role
	Status           = domain.Status
	NodeEventPayload = domain.NodeEventPayload
	NodeResponse     = domain.NodeResponse
)

// Node is the node feature facade.
type Node struct {
	svc     *service.Node
	handler *handler.Handler
}

// New returns a new Node feature facade backed by Ent.
func New(client *ent.Client) *Node {
	r := repo.New(client)
	svc := service.New(r)
	return &Node{
		svc:     svc,
		handler: handler.New(svc),
	}
}

// Bootstrap registers the master node and resets workers to offline.
func (n *Node) Bootstrap(ctx context.Context) error {
	return n.svc.Bootstrap(ctx)
}

// Register mounts the node HTTP routes on r.
func (n *Node) Register(r fiber.Router) {
	n.handler.Register(r)
}
