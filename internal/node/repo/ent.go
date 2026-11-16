package repo

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/kilip/opus/ent"
	"github.com/kilip/opus/ent/jointoken"
	entnode "github.com/kilip/opus/ent/node"
	"github.com/kilip/opus/internal/node/domain"
)

func toDomain(n *ent.Node) *domain.Node {
	if n == nil {
		return nil
	}
	return &domain.Node{
		ID:             n.ID.String(),
		Name:           n.Name,
		Role:           domain.Role(n.Role),
		Status:         domain.Status(n.Status),
		LastSeenAt:     n.LastSeenAt,
		CredentialHash: n.CredentialHash,
	}
}

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
		out[i] = *toDomain(n)
	}
	return out, nil
}

// Delete deletes a worker node by id.
// Returns ErrNodeNotFound if the node does not exist or id is malformed,
// and ErrMasterDeleteForbidden if the node has role master.
func (r *Ent) Delete(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return domain.ErrNodeNotFound
	}

	n, err := r.client.Node.Get(ctx, uid)
	if err != nil {
		if ent.IsNotFound(err) {
			return domain.ErrNodeNotFound
		}
		return err
	}

	if n.Role == entnode.RoleMaster {
		return domain.ErrMasterDeleteForbidden
	}

	if err := r.client.Node.DeleteOneID(uid).Exec(ctx); err != nil {
		if ent.IsNotFound(err) {
			return domain.ErrNodeNotFound
		}
		return err
	}
	return nil
}

// RecordHeartbeat updates the last_seen_at timestamp and sets status to online.
// Returns the updated node and whether the status changed from non-online to online.
// Returns ErrNodeNotFound if the node does not exist, and ErrMasterConnectForbidden if role is master.
func (r *Ent) RecordHeartbeat(ctx context.Context, id string, seenAt time.Time) (*domain.Node, bool, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, false, domain.ErrNodeNotFound
	}

	tx, err := r.client.Tx(ctx)
	if err != nil {
		return nil, false, err
	}
	defer func() { _ = tx.Rollback() }()

	n, err := tx.Node.Get(ctx, uid)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, false, domain.ErrNodeNotFound
		}
		return nil, false, err
	}

	if n.Role == entnode.RoleMaster {
		return nil, false, domain.ErrMasterConnectForbidden
	}

	statusChanged := n.Status != entnode.StatusOnline

	updated, err := tx.Node.UpdateOne(n).
		SetStatus(entnode.StatusOnline).
		SetLastSeenAt(seenAt).
		Save(ctx)
	if err != nil {
		return nil, false, err
	}

	if err := tx.Commit(); err != nil {
		return nil, false, err
	}

	return toDomain(updated), statusChanged, nil
}

// MarkOffline sets the node status to offline.
// Returns the updated node and whether the status changed from non-offline to offline.
// Returns ErrNodeNotFound if the node does not exist, and ErrMasterConnectForbidden if role is master.
func (r *Ent) MarkOffline(ctx context.Context, id string) (*domain.Node, bool, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, false, domain.ErrNodeNotFound
	}

	tx, err := r.client.Tx(ctx)
	if err != nil {
		return nil, false, err
	}
	defer func() { _ = tx.Rollback() }()

	n, err := tx.Node.Get(ctx, uid)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, false, domain.ErrNodeNotFound
		}
		return nil, false, err
	}

	if n.Role == entnode.RoleMaster {
		return nil, false, domain.ErrMasterConnectForbidden
	}

	statusChanged := n.Status != entnode.StatusOffline
	if !statusChanged {
		return toDomain(n), false, nil
	}

	updated, err := tx.Node.UpdateOne(n).
		SetStatus(entnode.StatusOffline).
		Save(ctx)
	if err != nil {
		return nil, false, err
	}

	if err := tx.Commit(); err != nil {
		return nil, false, err
	}

	return toDomain(updated), statusChanged, nil
}

// FindStaleWorkers returns online worker nodes whose last_seen_at is older than cutoff or nil.
// Master nodes are excluded.
func (r *Ent) FindStaleWorkers(ctx context.Context, cutoff time.Time) ([]domain.Node, error) {
	nodes, err := r.client.Node.Query().
		Where(
			entnode.RoleEQ(entnode.RoleWorker),
			entnode.StatusEQ(entnode.StatusOnline),
			entnode.Or(
				entnode.LastSeenAtLT(cutoff),
				entnode.LastSeenAtIsNil(),
			),
		).
		Order(entnode.ByID()).
		All(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]domain.Node, len(nodes))
	for i, n := range nodes {
		out[i] = *toDomain(n)
	}
	return out, nil
}

// FindByCredentialHash finds a node by its credential hash.
// Returns nil, nil if no node with the given hash is found.
func (r *Ent) FindByCredentialHash(ctx context.Context, hash string) (*domain.Node, error) {
	n, err := r.client.Node.Query().
		Where(entnode.CredentialHashEQ(hash)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return toDomain(n), nil
}

// GetMaster returns the registered master node, or nil, nil if not found.
func (r *Ent) GetMaster(ctx context.Context) (*domain.Node, error) {
	n, err := r.client.Node.Query().
		Where(entnode.RoleEQ(entnode.RoleMaster)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return toDomain(n), nil
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

// JoinNode atomically validates and consumes a join token, and inserts a worker node.
// Returns ErrTokenNotFound if not found, ErrTokenAlreadyUsed if already consumed,
// ErrTokenExpired if past its expiration time, and ErrNodeNameConflict if node name is taken.
func (r *Ent) JoinNode(ctx context.Context, tokenHash, name, credHash string, now time.Time) (*domain.Node, error) {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	tok, err := tx.JoinToken.Query().
		Where(jointoken.TokenHashEQ(tokenHash)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, domain.ErrTokenNotFound
		}
		return nil, err
	}

	if tok.UsedAt != nil {
		return nil, domain.ErrTokenAlreadyUsed
	}

	if now.After(tok.ExpiresAt) {
		return nil, domain.ErrTokenExpired
	}

	if err := tx.JoinToken.UpdateOne(tok).
		SetUsedAt(now).
		Exec(ctx); err != nil {
		return nil, err
	}

	n, err := tx.Node.Create().
		SetName(name).
		SetRole(entnode.RoleWorker).
		SetStatus(entnode.StatusOffline).
		SetCredentialHash(credHash).
		Save(ctx)
	if err != nil {
		if ent.IsConstraintError(err) {
			return nil, domain.ErrNodeNameConflict
		}
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return toDomain(n), nil
}
