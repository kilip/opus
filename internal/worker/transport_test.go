package worker_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/kilip/opus/internal/node"
	"github.com/kilip/opus/internal/worker"
)

type recordingConn struct {
	mu   sync.Mutex
	sent []node.Envelope
}

func (c *recordingConn) WriteJSON(v any) error {
	env, ok := v.(node.Envelope)
	if !ok {
		raw, err := json.Marshal(v)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(raw, &env); err != nil {
			return err
		}
	}
	c.mu.Lock()
	c.sent = append(c.sent, env)
	c.mu.Unlock()
	return nil
}

func (c *recordingConn) Close() error { return nil }

func (c *recordingConn) envelopes() []node.Envelope {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]node.Envelope, len(c.sent))
	copy(out, c.sent)
	return out
}

func TestHubTransport_SendAndDisconnected(t *testing.T) {
	ctx := context.Background()
	hub := node.New(nil, node.Config{}).Hub()
	defer func() { _ = hub.Close() }()

	nodeID := uuid.New()
	tr := &worker.HubTransport{Hub: hub}

	if err := tr.SendStartRun(ctx, nodeID, worker.StartRun{RunID: "run-1"}); !errors.Is(err, node.ErrNodeNotConnected) {
		t.Fatalf("SendStartRun disconnected = %v, want ErrNodeNotConnected", err)
	}
	if err := tr.SendCancelRun(ctx, nodeID, worker.CancelRun{RunID: "run-1"}); !errors.Is(err, node.ErrNodeNotConnected) {
		t.Fatalf("SendCancelRun disconnected = %v, want ErrNodeNotConnected", err)
	}
	if err := tr.CleanupWorktree(ctx, nodeID, "opus", "T-1", "vibe/T-1", ""); !errors.Is(err, node.ErrNodeNotConnected) {
		t.Fatalf("CleanupWorktree disconnected = %v, want ErrNodeNotConnected", err)
	}

	conn := &recordingConn{}
	unreg := hub.RegisterLocal(nodeID.String(), conn)
	defer unreg()

	if err := tr.SendStartRun(ctx, nodeID, worker.StartRun{
		RunID:     "run-1",
		SessionID: "sess-1",
		Prompt:    "/vibe 1",
		Branch:    "vibe/T-1",
	}); err != nil {
		t.Fatalf("SendStartRun: %v", err)
	}
	if err := tr.SendCancelRun(ctx, nodeID, worker.CancelRun{RunID: "run-1", Reason: "stop"}); err != nil {
		t.Fatalf("SendCancelRun: %v", err)
	}
	if err := tr.CleanupWorktree(ctx, nodeID, "opus", "T-1", "vibe/T-1", ""); err != nil {
		t.Fatalf("CleanupWorktree remote: %v", err)
	}

	envs := conn.envelopes()
	if len(envs) != 3 {
		t.Fatalf("expected 3 envelopes, got %d", len(envs))
	}

	var startCmd worker.StartRun
	if err := json.Unmarshal(envs[0].Payload, &startCmd); err != nil {
		t.Fatalf("unmarshal start_run: %v", err)
	}
	if startCmd.Type != worker.CommandStartRun || startCmd.RunID != "run-1" || startCmd.SessionID != "sess-1" {
		t.Fatalf("unexpected start_run payload: %+v", startCmd)
	}

	var cancelCmd worker.CancelRun
	if err := json.Unmarshal(envs[1].Payload, &cancelCmd); err != nil {
		t.Fatalf("unmarshal cancel_run: %v", err)
	}
	if cancelCmd.Type != worker.CommandCancelRun || cancelCmd.RunID != "run-1" || cancelCmd.Reason != "stop" {
		t.Fatalf("unexpected cancel_run payload: %+v", cancelCmd)
	}

	var cleanupCmd worker.CleanupWorktree
	if err := json.Unmarshal(envs[2].Payload, &cleanupCmd); err != nil {
		t.Fatalf("unmarshal cleanup_worktree: %v", err)
	}
	if cleanupCmd.Type != worker.CommandCleanupWorktree || cleanupCmd.ProjectSlug != "opus" || cleanupCmd.TaskKey != "T-1" || cleanupCmd.Branch != "vibe/T-1" {
		t.Fatalf("unexpected cleanup_worktree payload: %+v", cleanupCmd)
	}
}

func TestHubTransport_MasterCleanupUsesLocalGit(t *testing.T) {
	ctx := context.Background()
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	remote := filepath.Join(tmp, "remote.git")

	runGitCmd := func(dir string, args ...string) {
		t.Helper()
		cmd := exec.CommandContext(ctx, "git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=test",
			"GIT_AUTHOR_EMAIL=test@example.com",
			"GIT_COMMITTER_NAME=test",
			"GIT_COMMITTER_EMAIL=test@example.com",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, string(out))
		}
	}

	runGitCmd(tmp, "init", "--bare", remote)
	seed := filepath.Join(tmp, "seed")
	runGitCmd(tmp, "clone", remote, seed)
	if err := os.WriteFile(filepath.Join(seed, "README.md"), []byte("init\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitCmd(seed, "add", "README.md")
	runGitCmd(seed, "commit", "-m", "init")
	runGitCmd(seed, "branch", "-M", "main")
	runGitCmd(seed, "push", "-u", "origin", "main")

	gitMgr := worker.NewGitManager(home)
	repoDir, err := gitMgr.EnsureRepo(ctx, "opus", remote, "")
	if err != nil {
		t.Fatalf("EnsureRepo: %v", err)
	}
	wtDir, err := gitMgr.PrepareWorktree(ctx, repoDir, "opus", "T-1", "vibe/T-1", "main")
	if err != nil {
		t.Fatalf("PrepareWorktree: %v", err)
	}
	if _, err := os.Stat(wtDir); err != nil {
		t.Fatalf("expected worktree dir to exist: %v", err)
	}

	masterID := uuid.New()
	tr := &worker.HubTransport{
		Git:      gitMgr,
		MasterID: masterID.String(),
	}
	if err := tr.CleanupWorktree(ctx, masterID, "opus", "T-1", "vibe/T-1", ""); err != nil {
		t.Fatalf("CleanupWorktree master: %v", err)
	}
	if _, err := os.Stat(wtDir); !os.IsNotExist(err) {
		t.Fatalf("expected worktree dir %s to be removed, stat err = %v", wtDir, err)
	}
}
