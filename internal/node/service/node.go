package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/kilip/opus/internal/node/domain"
	"github.com/kilip/opus/internal/node/repo"
)

// ponytail: token TTL fixed at 15 minutes; make configurable when nodes need longer join windows
const joinTokenTTL = 15 * time.Minute

// Config holds service parameters for node management.
type Config struct {
	BaseURL string
	Now     func() time.Time
}

// Node owns node registry business rules.
type Node struct {
	repo    *repo.Ent
	baseURL string
	now     func() time.Time
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

// hashToken returns the hex-encoded SHA-256 hash of a raw token.
func hashToken(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}
