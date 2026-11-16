package repo

import (
	"context"
	"time"

	"github.com/kilip/opus/ent"
	"github.com/kilip/opus/ent/jointoken"
	entnode "github.com/kilip/opus/ent/node"
	"github.com/kilip/opus/internal/node/domain"
)

// Ent is an Ent-backed node repository.
type Ent struct {
	client *ent.Client
}

// New creates a new Ent node repository.
func New(client *ent.Client) *Ent {
	return &Ent{client: client}
}

// EnsureMaster ensures that exactly one node with role=master exists.
// Its name is updated to hostname, status set to online, credential_hash set to NULL,
// and any existing ID is preserved across restarts.
func (r *Ent) EnsureMaster(ctx context.Context, hostname string) error {
	masters, err := r.client.Node.Query().
		Where(entnode.RoleEQ(entnode.RoleMaster)).
		Order(entnode.ByID()).
		All(ctx)
	if err != nil {
		return err
	}

	if len(masters) > 0 {
		return r.client.Node.UpdateOne(masters[0]).
			SetName(hostname).
			SetStatus(entnode.StatusOnline).
			ClearCredentialHash().
			Exec(ctx)
	}

	byName, err := r.client.Node.Query().
		Where(entnode.NameEQ(hostname)).
		First(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return err
	}
	if byName != nil {
		return r.client.Node.UpdateOne(byName).
			SetRole(entnode.RoleMaster).
			SetStatus(entnode.StatusOnline).
			ClearCredentialHash().
			Exec(ctx)
	}

	return r.client.Node.Create().
		SetName(hostname).
		SetRole(entnode.RoleMaster).
		SetStatus(entnode.StatusOnline).
		Exec(ctx)
}

// ResetWorkersOffline resets status of all worker nodes to offline on startup.
func (r *Ent) ResetWorkersOffline(ctx context.Context) error {
	return r.client.Node.Update().
		Where(entnode.RoleEQ(entnode.RoleWorker)).
		SetStatus(entnode.StatusOffline).
		Exec(ctx)
}

// List returns all nodes ordered by id ascending.
func (r *Ent) List(ctx context.Context) ([]domain.Node, error) {
	nodes, err := r.client.Node.Query().
		Order(entnode.ByID()).
		All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Node, len(nodes))
	for i, n := range nodes {
		out[i] = domain.Node{
			ID:             n.ID.String(),
			Name:           n.Name,
			Role:           domain.Role(n.Role),
			Status:         domain.Status(n.Status),
			LastSeenAt:     n.LastSeenAt,
			CredentialHash: n.CredentialHash,
		}
	}
	return out, nil
}

// CreateJoinToken stores a join token hash and expiration.
func (r *Ent) CreateJoinToken(ctx context.Context, tokenHash string, expiresAt time.Time) error {
	return r.client.JoinToken.Create().
		SetTokenHash(tokenHash).
		SetExpiresAt(expiresAt).
		Exec(ctx)
}

// ConsumeJoinToken atomically validates and consumes a join token by its hash.
// Returns ErrTokenNotFound if not found, ErrTokenAlreadyUsed if already consumed,
// ErrTokenExpired if past its expiration time.
func (r *Ent) ConsumeJoinToken(ctx context.Context, tokenHash string, now time.Time) error {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	tok, err := tx.JoinToken.Query().
		Where(jointoken.TokenHashEQ(tokenHash)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return domain.ErrTokenNotFound
		}
		return err
	}

	if tok.UsedAt != nil {
		return domain.ErrTokenAlreadyUsed
	}

	if now.After(tok.ExpiresAt) {
		return domain.ErrTokenExpired
	}

	if err := tx.JoinToken.UpdateOne(tok).
		SetUsedAt(now).
		Exec(ctx); err != nil {
		return err
	}

	return tx.Commit()
}
