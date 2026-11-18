package run_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/kilip/opus/ent"
	"github.com/kilip/opus/ent/enttest"
	entnode "github.com/kilip/opus/ent/node"
	"github.com/kilip/opus/internal/notify"
	"github.com/kilip/opus/internal/platform/eventbus"
	"github.com/kilip/opus/internal/project"
	"github.com/kilip/opus/internal/run"
	"github.com/kilip/opus/internal/session"
	"github.com/kilip/opus/internal/task"
	"github.com/kilip/opus/internal/worker"
	_ "modernc.org/sqlite"
)

type e2eWorkerTransport struct {
	t         *testing.T
	git       *worker.GitManager
	remoteDir string
	fakeGHBin string
	tasks     *task.Task
	sessions  *session.Session
	runs      *run.Feature
	mu        sync.Mutex
	repoDir   string
}

func (w *e2eWorkerTransport) SendStartRun(ctx context.Context, _ uuid.UUID, cmd worker.StartRun) error {
	repoDir, err := w.git.EnsureRepo(ctx, cmd.ProjectSlug, w.remoteDir, cmd.MappedPath)
	if err != nil {
		return err
	}
	wtDir, err := w.git.PrepareWorktree(ctx, repoDir, cmd.ProjectSlug, cmd.TaskKey, cmd.Branch, cmd.DefaultBranch)
	if err != nil {
		return err
	}
	w.mu.Lock()
	w.repoDir = repoDir
	w.mu.Unlock()

	if !strings.HasPrefix(cmd.Prompt, "/vibe 129\n") {
		w.t.Errorf("unexpected prompt prefix: %q", cmd.Prompt)
	}
	if cmd.Policy == nil || !cmd.Policy.AllowMerge {
		w.t.Errorf("expected AllowMerge=true policy, got %+v", cmd.Policy)
	}

	// Simulate fake agy ACP + fake gh execution asynchronously.
	go func() {
		bg := context.Background()
		featureFile := filepath.Join(wtDir, "feature.txt")
		if err := os.WriteFile(featureFile, []byte("implemented #129\n"), 0o644); err != nil {
			w.t.Errorf("write feature file: %v", err)
			return
		}
		mustGit(w.t, wtDir, "add", "feature.txt")
		mustGit(w.t, wtDir, "commit", "-m", "feat(api): smoke test change")
		mustGit(w.t, wtDir, "push", "-u", "origin", cmd.Branch)

		// Invoke fake gh CLI (pr create, pr checks --watch, pr merge --ff-only).
		for _, args := range [][]string{
			{"pr", "create", "--head", cmd.Branch},
			{"pr", "checks", "--watch"},
			{"pr", "merge", "--ff-only", cmd.Branch},
		} {
			c := exec.CommandContext(bg, w.fakeGHBin, args...)
			c.Dir = wtDir
			c.Env = append(os.Environ(), "OPUS_REMOTE_DIR="+w.remoteDir)
			if out, err := c.CombinedOutput(); err != nil {
				w.t.Errorf("fake gh %v failed: %v\n%s", args, err, string(out))
				return
			}
		}

		// Complete active session and mark task done (which triggers cleanup + merged + run.finished).
		runRec, err := w.runs.Get(bg, uuid.MustParse(cmd.RunID))
		if err == nil && runRec.SessionID != nil {
			_, _, _ = w.sessions.Complete(bg, *runRec.SessionID)
		}
		done := task.StatusDone
		if _, err := w.tasks.Update(bg, cmd.TaskID, task.UpdateTaskRequest{Status: &done}); err != nil {
			w.t.Errorf("tasks.Update done: %v", err)
		}
	}()
	return nil
}

func (w *e2eWorkerTransport) SendCancelRun(_ context.Context, _ uuid.UUID, _ worker.CancelRun) error {
	return nil
}

func (w *e2eWorkerTransport) CleanupWorktree(ctx context.Context, _ uuid.UUID, projectSlug, taskKey, branch, _ string) error {
	w.mu.Lock()
	repoDir := w.repoDir
	w.mu.Unlock()
	return w.git.CleanupWorktree(ctx, repoDir, projectSlug, taskKey, branch)
}

func mustGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Opus Test",
		"GIT_AUTHOR_EMAIL=test@opus.local",
		"GIT_COMMITTER_NAME=Opus Test",
		"GIT_COMMITTER_EMAIL=test@opus.local",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v in %s failed: %v\n%s", args, dir, err, string(out))
	}
	return strings.TrimSpace(string(out))
}

func TestE2E_VibeRunSmoke(t *testing.T) {
	ctx := context.Background()

	// 1. Setup bare git remote seeded with main branch.
	remoteDir := filepath.Join(t.TempDir(), "remote.git")
	mustGit(t, ".", "init", "--bare", "--initial-branch=main", remoteDir)

	seedDir := t.TempDir()
	mustGit(t, ".", "clone", remoteDir, seedDir)
	mustGit(t, seedDir, "checkout", "-B", "main")
	if err := os.WriteFile(filepath.Join(seedDir, "README.md"), []byte("# opus\n"), 0o644); err != nil {
		t.Fatalf("write README: %v", err)
	}
	mustGit(t, seedDir, "add", "README.md")
	mustGit(t, seedDir, "commit", "-m", "chore: initial commit")
	mustGit(t, seedDir, "push", "-u", "origin", "main")

	// 2. Setup fake gh script in t.TempDir() that fast-forward merges to main on bare remote.
	fakeGHBin := filepath.Join(t.TempDir(), "gh")
	ghScript := `#!/usr/bin/env bash
set -euo pipefail
if [ "$1" = "pr" ] && [ "$2" = "merge" ]; then
  branch="$4"
  git fetch origin main
  git push "$OPUS_REMOTE_DIR" "origin/${branch}:refs/heads/main"
fi
`
	if err := os.WriteFile(fakeGHBin, []byte(ghScript), 0o755); err != nil {
		t.Fatalf("write fake gh: %v", err)
	}

	// 3. Setup Telegram httptest.Server.
	var (
		tgMu   sync.Mutex
		tgMsgs []string
	)
	tgSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body struct {
			Text string `json:"text"`
		}
		_ = json.Unmarshal(raw, &body)
		tgMu.Lock()
		tgMsgs = append(tgMsgs, body.Text)
		tgMu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer tgSrv.Close()

	// 4. Wire real DB, eventbus, and Opus features.
	db, err := sql.Open("sqlite", "file:"+t.Name()+"?mode=memory&cache=shared&_pragma=foreign_keys(ON)&_pragma=busy_timeout(10000)")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db.SetMaxOpenConns(1)
	drv := entsql.OpenDB(dialect.SQLite, db)
	client := enttest.NewClient(t, enttest.WithOptions(ent.Driver(drv)))
	t.Cleanup(func() { _ = client.Close() })

	bus := eventbus.New()
	t.Cleanup(func() { _ = bus.Close(context.Background()) })

	if err := notify.New(notify.Config{
		BotToken: "e2e-token",
		ChatID:   "e2e-chat",
		BaseURL:  tgSrv.URL,
	}).Register(bus); err != nil {
		t.Fatalf("notify.Register: %v", err)
	}

	projFeature := project.New(client)
	taskFeature := task.New(client, bus, task.WithProjectChecker(projFeature))
	sessionFeature := session.New(client, session.Config{Bus: bus})

	proj, err := client.Project.Create().
		SetSlug("opus-e2e").
		SetName("Opus E2E").
		SetRepoProvider("github").
		SetRepoRef(remoteDir).
		SetTrackerProvider("github").
		SetTrackerRef("kilip/opus").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	n, err := client.Node.Create().
		SetName("worker-e2e").
		SetRole(entnode.RoleWorker).
		SetStatus(entnode.StatusOnline).
		Save(ctx)
	if err != nil {
		t.Fatalf("create node: %v", err)
	}
	if _, err := client.ProjectNode.Create().
		SetProjectID(proj.ID).
		SetNodeID(n.ID).
		Save(ctx); err != nil {
		t.Fatalf("create project_node: %v", err)
	}

	appHome := t.TempDir()
	transport := &e2eWorkerTransport{
		t:         t,
		git:       worker.NewGitManager(appHome),
		remoteDir: remoteDir,
		fakeGHBin: fakeGHBin,
		tasks:     taskFeature,
		sessions:  sessionFeature,
	}

	runFeature := run.New(client, run.Config{
		Bus:          bus,
		AppHome:      appHome,
		Tasks:        taskFeature,
		Projects:     projFeature,
		Sessions:     sessionFeature,
		Worker:       transport,
		PollInterval: 25 * time.Millisecond,
	})
	transport.runs = runFeature

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	runDone := make(chan error, 1)
	go func() {
		runDone <- runFeature.Run(runCtx)
	}()

	// 5. Import eligible GitHub issue -> triggers full vibe run lifecycle.
	imported, err := taskFeature.Import(ctx, task.ImportTaskRequest{
		ProjectID:   proj.ID.String(),
		Source:      task.SourceGitHub,
		ExternalID:  "129",
		Title:       "e2e smoke run",
		Description: "## Scope\nSmoke test.\n\n## Acceptance criteria\n- AC1: Passes.\n\n## Definition of Done\n- Run `go test ./...`\n",
		Status:      task.StatusTodo,
		Labels:      []string{"vibe", "type:feat"},
	})
	if err != nil {
		t.Fatalf("taskFeature.Import: %v", err)
	}

	// 6. Wait for Run to reach merged + Telegram notification to arrive.
	deadline := time.Now().Add(10 * time.Second)
	var finalRun run.Run
	for time.Now().Before(deadline) {
		runs, err := runFeature.List(ctx, run.ListFilter{})
		tgMu.Lock()
		tgCount := len(tgMsgs)
		tgMu.Unlock()
		if err == nil && len(runs) == 1 && runs[0].State == run.StateMerged && tgCount >= 1 {
			finalRun = runs[0]
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	if finalRun.State != run.StateMerged {
		runs, _ := runFeature.List(ctx, run.ListFilter{})
		tgMu.Lock()
		msgs := append([]string(nil), tgMsgs...)
		tgMu.Unlock()
		t.Fatalf("expected Run state merged, got runs=%+v tgMsgs=%v", runs, msgs)
	}

	// Verify bare remote main received the feature commit.
	logOut := mustGit(t, remoteDir, "log", "-n", "1", "--oneline", "main")
	if !strings.Contains(logOut, "feat(api): smoke test change") {
		t.Fatalf("expected fast-forward merged commit on remote main, got %q", logOut)
	}

	// Verify worktree was cleaned up.
	wtDir := filepath.Join(appHome, "wt", "opus-e2e", imported.Key)
	if _, err := os.Stat(wtDir); !os.IsNotExist(err) {
		t.Fatalf("expected worktree %s to be removed, stat err=%v", wtDir, err)
	}

	// Verify Telegram notification content.
	tgMu.Lock()
	defer tgMu.Unlock()
	if len(tgMsgs) == 0 || !strings.Contains(tgMsgs[0], "merged") {
		t.Fatalf("expected Telegram notification for merged run, got %v", tgMsgs)
	}

	cancel()
	if err := <-runDone; err != nil {
		t.Fatalf("runFeature.Run returned error: %v", err)
	}
}
