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
	"github.com/kilip/opus/internal/platform/eventbus"
)

// Event type constants re-exported from domain.
const (
	EventOnline  = domain.EventOnline
	EventOffline = domain.EventOffline

	RoleMaster = domain.RoleMaster
	RoleWorker = domain.RoleWorker

	StatusOnline  = domain.StatusOnline
	StatusOffline = domain.StatusOffline

	EnvelopeTypeHello     = domain.EnvelopeTypeHello
	EnvelopeTypeHeartbeat = domain.EnvelopeTypeHeartbeat
	EnvelopeTypeCommand   = domain.EnvelopeTypeCommand
	EnvelopeTypeEvent     = domain.EnvelopeTypeEvent

	ToolStatusReady           = domain.ToolStatusReady
	ToolStatusUnauthenticated = domain.ToolStatusUnauthenticated
	ToolStatusMissing         = domain.ToolStatusMissing

	DefaultDetectorTimeout  = service.DefaultDetectorTimeout
	DefaultDetectorInterval = service.DefaultDetectorInterval
)

// Sentinel errors re-exported from domain.
var (
	ErrTokenNotFound          = domain.ErrTokenNotFound
	ErrTokenExpired           = domain.ErrTokenExpired
	ErrTokenAlreadyUsed       = domain.ErrTokenAlreadyUsed
	ErrNodeNameConflict       = domain.ErrNodeNameConflict
	ErrInvalidNodeName        = domain.ErrInvalidNodeName
	ErrInvalidIPAddress       = domain.ErrInvalidIPAddress
	ErrInvalidMACAddress      = domain.ErrInvalidMACAddress
	ErrRateLimited            = domain.ErrRateLimited
	ErrNodeNotConnected       = domain.ErrNodeNotConnected
	ErrNodeNotFound           = domain.ErrNodeNotFound
	ErrMasterConnectForbidden = domain.ErrMasterConnectForbidden
	ErrMasterDeleteForbidden  = domain.ErrMasterDeleteForbidden
	ErrUnauthorized           = domain.ErrUnauthorized
	ErrUnknownEnvelopeType    = domain.ErrUnknownEnvelopeType
)

// Re-exported domain types.
type (
	Role              = domain.Role
	Status            = domain.Status
	NodeRecord        = domain.Node
	NodeEventPayload  = domain.NodeEventPayload
	NodeResponse      = domain.NodeResponse
	UpdateNodeRequest = domain.UpdateNodeRequest
	JoinTokenResponse = domain.JoinTokenResponse
	JoinNodeRequest   = domain.JoinNodeRequest
	JoinNodeResponse  = domain.JoinNodeResponse
	Envelope          = domain.Envelope
	ToolStatus        = domain.ToolStatus
	Capabilities      = domain.Capabilities
	HelloPayload      = domain.HelloPayload
	RateLimitConfig   = handler.RateLimitConfig
	Hub               = service.Hub
	Conn              = service.Conn
	Detector          = service.Detector
	DetectorConfig    = service.DetectorConfig
)

// Config holds configuration parameters for the node feature.
type Config struct {
	BaseURL          string
	Now              func() time.Time
	RateLimitConfig  handler.RateLimitConfig
	Bus              *eventbus.Bus
	DetectorInterval time.Duration
	DetectorTimeout  time.Duration
	DetectorOnStop   func()
}

// Node is the node feature facade.
type Node struct {
	svc      *service.Node
	hub      *service.Hub
	detector *service.Detector
	handler  *handler.Handler
}

// New returns a new Node feature facade backed by Ent.
func New(client *ent.Client, cfg Config) *Node {
	r := repo.New(client)
	hub := service.NewHub()
	svc := service.New(r, service.Config{
		BaseURL: cfg.BaseURL,
		Now:     cfg.Now,
		Bus:     cfg.Bus,
		Hub:     hub,
	})
	hub.OnHeartbeat(func(nodeID string, env domain.Envelope) {
		_ = svc.RecordHeartbeat(context.Background(), nodeID)
		if env.Type == domain.EnvelopeTypeHello {
			_ = svc.HandleHello(context.Background(), nodeID, env)
		}
	})
	hub.OnDisconnect(func(nodeID string) {
		_ = svc.MarkOffline(context.Background(), nodeID)
	})
	detector := service.NewDetector(svc, service.DetectorConfig{
		Now:      cfg.Now,
		Interval: cfg.DetectorInterval,
		Timeout:  cfg.DetectorTimeout,
		OnStop:   cfg.DetectorOnStop,
	})
	return &Node{
		svc:      svc,
		hub:      hub,
		detector: detector,
		handler:  handler.New(svc, hub, cfg.RateLimitConfig),
	}
}

// Hub returns the node feature live connection hub.
func (n *Node) Hub() *service.Hub {
	return n.hub
}

// Running returns true if the background detection loop is running.
func (n *Node) Running() bool {
	if n.detector != nil {
		return n.detector.Running()
	}
	return false
}

// Start starts background services including the stale node detector.
func (n *Node) Start(ctx context.Context) error {
	if n.detector != nil {
		return n.detector.Start(ctx)
	}
	return nil
}

// Stop stops background services including the stale node detector and closes hub connections.
func (n *Node) Stop() {
	if n.detector != nil {
		n.detector.Stop()
	}
	if n.hub != nil {
		_ = n.hub.Close()
	}
}

// CheckStale runs a single stale detection check.
func (n *Node) CheckStale(ctx context.Context) error {
	if n.detector != nil {
		return n.detector.CheckStale(ctx)
	}
	return nil
}

// Bootstrap registers the master node and resets workers to offline.
func (n *Node) Bootstrap(ctx context.Context) error {
	if err := n.svc.Bootstrap(ctx); err != nil {
		return err
	}
	if master, err := n.svc.GetMaster(ctx); err == nil && master != nil {
		n.hub.SetMasterID(master.ID)
	}
	return nil
}

// MasterID returns the registered master node ID, or empty string if not bootstrapped.
func (n *Node) MasterID(ctx context.Context) (string, error) {
	master, err := n.svc.GetMaster(ctx)
	if err != nil || master == nil {
		return "", err
	}
	return master.ID, nil
}

// List returns every registered node as NodeResponse DTOs.
func (n *Node) List(ctx context.Context) ([]domain.NodeResponse, error) {
	return n.svc.List(ctx)
}

// Get returns a single node by id as a NodeResponse DTO, or ErrNodeNotFound.
func (n *Node) Get(ctx context.Context, id string) (domain.NodeResponse, error) {
	return n.svc.Get(ctx, id)
}

// Update updates editable fields on a node by id.
func (n *Node) Update(ctx context.Context, id string, req domain.UpdateNodeRequest) (domain.NodeResponse, error) {
	return n.svc.Update(ctx, id, req)
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

// Delete removes a worker node by id.
func (n *Node) Delete(ctx context.Context, id string) error {
	return n.svc.Delete(ctx, id)
}

// Register mounts the protected node HTTP routes on r.
func (n *Node) Register(r fiber.Router) {
	n.handler.Register(r)
}

// RegisterPublic mounts unauthenticated node HTTP routes on r.
func (n *Node) RegisterPublic(r fiber.Router) {
	n.handler.RegisterPublic(r)
}

// AuthenticateWorker authenticates a worker node by its raw Bearer credential.
func (n *Node) AuthenticateWorker(ctx context.Context, cred string) (*NodeRecord, error) {
	return n.svc.AuthenticateWorker(ctx, cred)
}
