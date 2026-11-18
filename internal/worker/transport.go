package worker

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/kilip/opus/internal/node"
)

// HubTransport dispatches run commands and worktree cleanup via node.Hub
// (or directly via GitManager for the master node).
type HubTransport struct {
	Hub      *node.Hub
	Git      *GitManager
	MasterID string
}

// SendStartRun encodes cmd as a start_run envelope and sends it to nodeID.
func (t *HubTransport) SendStartRun(_ context.Context, nodeID uuid.UUID, cmd StartRun) error {
	if t.Hub == nil || !t.Hub.Connected(nodeID.String()) {
		return fmt.Errorf("%w: %s", node.ErrNodeNotConnected, nodeID)
	}
	env, err := NewStartRunEnvelope("cmd-start-"+cmd.RunID, cmd)
	if err != nil {
		return err
	}
	return t.Hub.Send(nodeID.String(), env)
}

// SendCancelRun encodes cmd as a cancel_run envelope and sends it to nodeID.
func (t *HubTransport) SendCancelRun(_ context.Context, nodeID uuid.UUID, cmd CancelRun) error {
	if t.Hub == nil || !t.Hub.Connected(nodeID.String()) {
		return fmt.Errorf("%w: %s", node.ErrNodeNotConnected, nodeID)
	}
	env, err := NewCancelRunEnvelope("cmd-cancel-"+cmd.RunID, cmd)
	if err != nil {
		return err
	}
	return t.Hub.Send(nodeID.String(), env)
}

// CleanupWorktree removes the worktree locally on the master node, or sends a
// cleanup_worktree command envelope to a connected remote worker node.
func (t *HubTransport) CleanupWorktree(
	ctx context.Context,
	nodeID uuid.UUID,
	projectSlug, taskKey, branch, mappedPath string,
) error {
	if t.MasterID != "" && nodeID.String() == t.MasterID && t.Git != nil {
		repoDir, err := t.Git.EnsureRepo(ctx, projectSlug, "", mappedPath)
		if err != nil {
			return err
		}
		return t.Git.CleanupWorktree(ctx, repoDir, projectSlug, taskKey, branch)
	}
	if t.Hub == nil || !t.Hub.Connected(nodeID.String()) {
		return fmt.Errorf("%w: %s", node.ErrNodeNotConnected, nodeID)
	}
	env, err := NewCleanupWorktreeEnvelope("cmd-cleanup-"+projectSlug+"-"+taskKey, CleanupWorktree{
		ProjectSlug: projectSlug,
		TaskKey:     taskKey,
		Branch:      branch,
		MappedPath:  mappedPath,
	})
	if err != nil {
		return err
	}
	return t.Hub.Send(nodeID.String(), env)
}
