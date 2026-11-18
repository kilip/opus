package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/kilip/opus/internal/node/domain"
	"github.com/kilip/opus/internal/node/repo"
	"github.com/kilip/opus/internal/platform/eventbus"
)

// ponytail: token TTL fixed at 15 minutes; make configurable when nodes need longer join windows
const joinTokenTTL = 15 * time.Minute

// Config holds service parameters for node management.
type Config struct {
	BaseURL string
	Now     func() time.Time
	Bus     *eventbus.Bus
	Hub     *Hub
}

// Node owns node registry business rules.
type Node struct {
	repo    *repo.Ent
	baseURL string
	now     func() time.Time
	bus     *eventbus.Bus
	hub     *Hub
}

// New returns Node service backed by Ent repo and configuration.
func New(r *repo.Ent, cfg Config) *Node {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:7880"
	}
	now := cfg.Now
	if now == nil {
		now = time.Now
	}
	return &Node{
		repo:    r,
		baseURL: strings.TrimRight(baseURL, "/"),
		now:     now,
		bus:     cfg.Bus,
		hub:     cfg.Hub,
	}
}

// Bootstrap registers master node and resets workers to offline.
func (s *Node) Bootstrap(ctx context.Context) error {
	hostname, err := os.Hostname()
	if err != nil || hostname == "" {
		hostname = "master"
	}
	if err := s.repo.EnsureMaster(ctx, hostname); err != nil {
		return err
	}
	return s.repo.ResetWorkersOffline(ctx)
}

func toNodeResponse(n domain.Node) domain.NodeResponse {
	return domain.NodeResponse{
		ID:         n.ID,
		Name:       n.Name,
		Role:       n.Role,
		Status:     n.Status,
		LastSeenAt: n.LastSeenAt,
		Hostname:   n.Hostname,
		IPAddress:  n.IPAddress,
		MACAddress: n.MACAddress,
	}
}

// List returns every registered node as NodeResponse DTOs.
func (s *Node) List(ctx context.Context) ([]domain.NodeResponse, error) {
	nodes, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.NodeResponse, len(nodes))
	for i, n := range nodes {
		out[i] = toNodeResponse(n)
	}
	return out, nil
}

// Get returns a single node by id as a NodeResponse DTO, or domain.ErrNodeNotFound.
func (s *Node) Get(ctx context.Context, id string) (domain.NodeResponse, error) {
	n, err := s.repo.Get(ctx, id)
	if err != nil {
		return domain.NodeResponse{}, err
	}
	return toNodeResponse(*n), nil
}

// Update validates and updates provided editable fields on a node.
func (s *Node) Update(ctx context.Context, id string, req domain.UpdateNodeRequest) (domain.NodeResponse, error) {
	if req.Name != nil {
		if err := domain.ValidateNodeName(*req.Name); err != nil {
			return domain.NodeResponse{}, err
		}
	}
	if req.IPAddress != nil && *req.IPAddress != "" {
		if err := domain.ValidateIPAddress(*req.IPAddress); err != nil {
			return domain.NodeResponse{}, err
		}
	}
	if req.MACAddress != nil && *req.MACAddress != "" {
		normalized, err := domain.NormalizeMACAddress(*req.MACAddress)
		if err != nil {
			return domain.NodeResponse{}, err
		}
		req.MACAddress = &normalized
	}

	n, err := s.repo.UpdateNode(ctx, id, req)
	if err != nil {
		return domain.NodeResponse{}, err
	}
	return toNodeResponse(*n), nil
}

// GenerateJoinToken creates a 32-byte cryptographically secure join token,
// stores its SHA-256 hash in the database, and returns the join URL and expiration.
func (s *Node) GenerateJoinToken(ctx context.Context) (domain.JoinTokenResponse, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return domain.JoinTokenResponse{}, err
	}
	rawToken := base64.RawURLEncoding.EncodeToString(b)
	tokenHash := hashToken(rawToken)

	now := s.now()
	expiresAt := now.Add(joinTokenTTL)

	if err := s.repo.CreateJoinToken(ctx, tokenHash, expiresAt); err != nil {
		return domain.JoinTokenResponse{}, err
	}

	return domain.JoinTokenResponse{
		URL:       fmt.Sprintf("%s/join/%s", s.baseURL, rawToken),
		ExpiresAt: expiresAt,
	}, nil
}

// ConsumeJoinToken validates and consumes a join token by its raw token string.
func (s *Node) ConsumeJoinToken(ctx context.Context, rawToken string) error {
	if rawToken == "" {
		return domain.ErrTokenNotFound
	}
	tokenHash := hashToken(rawToken)
	return s.repo.ConsumeJoinToken(ctx, tokenHash, s.now())
}

// Join validates the join request, consumes the join token, and creates a worker node.
// Returns the node ID and a single-use credential.
func (s *Node) Join(ctx context.Context, req domain.JoinNodeRequest) (domain.JoinNodeResponse, error) {
	if err := domain.ValidateNodeName(req.Name); err != nil {
		return domain.JoinNodeResponse{}, err
	}
	if req.Token == "" {
		return domain.JoinNodeResponse{}, domain.ErrTokenNotFound
	}

	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return domain.JoinNodeResponse{}, err
	}
	credential := base64.RawURLEncoding.EncodeToString(b)
	credHash := hashToken(credential)
	tokenHash := hashToken(req.Token)

	h, ip, mac := sanitizeNetInfo(req.Hostname, req.IPAddress, req.MACAddress)
	n, err := s.repo.JoinNode(ctx, tokenHash, req.Name, credHash, s.now(), h, ip, mac)
	if err != nil {
		return domain.JoinNodeResponse{}, err
	}

	return domain.JoinNodeResponse{
		NodeID:     n.ID,
		Credential: credential,
	}, nil
}

// AuthenticateWorker authenticates a worker node by its raw credential.
// Returns ErrUnauthorized for missing, empty, or unrecognised credentials.
// Returns ErrMasterConnectForbidden if the node has role master.
func (s *Node) AuthenticateWorker(ctx context.Context, cred string) (*domain.Node, error) {
	if cred == "" {
		return nil, domain.ErrUnauthorized
	}
	hash := hashToken(cred)
	n, err := s.repo.FindByCredentialHash(ctx, hash)
	if err != nil {
		return nil, err
	}
	if n == nil {
		return nil, domain.ErrUnauthorized
	}
	if n.Role == domain.RoleMaster {
		return nil, domain.ErrMasterConnectForbidden
	}
	return n, nil
}

// Delete deletes a worker node and terminates any active control channel connection.
func (s *Node) Delete(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	if s.hub != nil {
		if err := s.hub.Disconnect(id); err != nil && !errors.Is(err, domain.ErrNodeNotConnected) {
			return err
		}
	}
	return nil
}

// GetMaster returns the registered master node.
func (s *Node) GetMaster(ctx context.Context) (*domain.Node, error) {
	return s.repo.GetMaster(ctx)
}

// RecordHeartbeat records a worker heartbeat/hello, sets status to online,
// and publishes node.online if the node transitioned from non-online to online.
func (s *Node) RecordHeartbeat(ctx context.Context, id string) error {
	n, changed, err := s.repo.RecordHeartbeat(ctx, id, s.now())
	if err != nil {
		return err
	}
	if changed && s.bus != nil {
		evt, err := eventbus.NewEvent(domain.EventOnline, n.ID, domain.NodeEventPayload{
			ID:   n.ID,
			Name: n.Name,
		})
		if err != nil {
			slog.Error("failed to create node.online event", "node_id", n.ID, "error", err)
		} else {
			// ponytail: publish errors are logged, not returned; becomes transactional when the outbox lands
			if err := s.bus.Publish(ctx, evt); err != nil {
				slog.Error("failed to publish node.online event", "node_id", n.ID, "error", err)
			}
		}
	}
	return nil
}

// MarkOffline sets the worker status to offline,
// and publishes node.offline if the node transitioned from non-offline to offline.
func (s *Node) MarkOffline(ctx context.Context, id string) error {
	n, changed, err := s.repo.MarkOffline(ctx, id)
	if err != nil {
		return err
	}
	if changed && s.bus != nil {
		evt, err := eventbus.NewEvent(domain.EventOffline, n.ID, domain.NodeEventPayload{
			ID:   n.ID,
			Name: n.Name,
		})
		if err != nil {
			slog.Error("failed to create node.offline event", "node_id", n.ID, "error", err)
		} else {
			// ponytail: publish errors are logged, not returned; becomes transactional when the outbox lands
			if err := s.bus.Publish(ctx, evt); err != nil {
				slog.Error("failed to publish node.offline event", "node_id", n.ID, "error", err)
			}
		}
	}
	return nil
}

// DetectStale finds worker nodes with last_seen_at older than cutoff and marks them offline.
func (s *Node) DetectStale(ctx context.Context, cutoff time.Time) error {
	stale, err := s.repo.FindStaleWorkers(ctx, cutoff)
	if err != nil {
		return err
	}
	for _, n := range stale {
		if err := s.MarkOffline(ctx, n.ID); err != nil {
			slog.Error("failed to mark stale node offline", "node_id", n.ID, "error", err)
		}
	}
	return nil
}

// HandleHello parses a worker hello envelope and updates the node's hostname, IP, and MAC when valid.
func (s *Node) HandleHello(ctx context.Context, nodeID string, env domain.Envelope) error {
	if len(env.Payload) == 0 {
		return nil
	}
	var payload domain.HelloPayload
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		return nil
	}
	h, ip, mac := sanitizeNetInfo(payload.Hostname, payload.IPAddress, payload.MACAddress)
	return s.repo.UpdateNodeNetInfo(ctx, nodeID, h, ip, mac)
}

func sanitizeNetInfo(hostname, ipAddress, macAddress string) (string, string, string) {
	hostname = strings.TrimSpace(hostname)
	ipAddress = strings.TrimSpace(ipAddress)
	macAddress = strings.TrimSpace(macAddress)
	if ipAddress != "" && domain.ValidateIPAddress(ipAddress) != nil {
		ipAddress = ""
	}
	if macAddress != "" {
		if norm, err := domain.NormalizeMACAddress(macAddress); err == nil {
			macAddress = norm
		} else {
			macAddress = ""
		}
	}
	return hostname, ipAddress, macAddress
}

// hashToken returns the hex-encoded SHA-256 hash of a raw token.
func hashToken(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}
