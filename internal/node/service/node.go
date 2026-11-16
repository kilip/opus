package service

import (
	"context"
	"os"

	"github.com/kilip/opus/internal/node/domain"
	"github.com/kilip/opus/internal/node/repo"
)

// Node owns node registry business rules.
type Node struct {
	repo *repo.Ent
}

// New returns Node service backed by Ent repo.
func New(r *repo.Ent) *Node {
	return &Node{repo: r}
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

// List returns every registered node as NodeResponse DTOs.
func (s *Node) List(ctx context.Context) ([]domain.NodeResponse, error) {
	nodes, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.NodeResponse, len(nodes))
	for i, n := range nodes {
		out[i] = domain.NodeResponse{
			ID:         n.ID,
			Name:       n.Name,
			Role:       n.Role,
			Status:     n.Status,
			LastSeenAt: n.LastSeenAt,
		}
	}
	return out, nil
}
