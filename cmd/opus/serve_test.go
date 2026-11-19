package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
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
	"github.com/kilip/opus/internal/app"
	"github.com/kilip/opus/internal/node"
	"github.com/kilip/opus/internal/platform/eventbus"
	"github.com/kilip/opus/internal/platform/httpx"
	"github.com/kilip/opus/internal/run"
	"github.com/kilip/opus/internal/task"
	"github.com/kilip/opus/internal/worker"
	_ "modernc.org/sqlite"
)

func openTestDB(t *testing.T) *ent.Client {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	dsn := "file:" + dbPath + "?_pragma=foreign_keys(ON)&_pragma=busy_timeout(5000)"
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	client := ent.NewClient(ent.Driver(entsql.OpenDB(dialect.SQLite, sqlDB)))
	if err := client.Schema.Create(context.Background()); err != nil {
		t.Fatalf("Schema.Create: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func freePort(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}
	_, port, _ := net.SplitHostPort(ln.Addr().String())
	_ = ln.Close()
	return port
}

func testConfig(t *testing.T) app.Config {
	t.Helper()
	return app.Config{
		AppName:             "opus-test",
		AppHome:             t.TempDir(),
		Host:                "127.0.0.1",
		Port:                freePort(t),
		LogLevel:            slog.LevelError,
		ReadTimeout:         5 * time.Second,
		WriteTimeout:        5 * time.Second,
		IdleTimeout:         10 * time.Second,
		ShutdownTimeout:     2 * time.Second,
		BodyLimit:           4 * 1024 * 1024,
		SessionSecret:       "test-secret-key-at-least-32-bytes-long",
		SessionTTL:          24 * time.Hour,
		TrackerPollInterval: time.Hour,
		RunPollInterval:     25 * time.Millisecond,
		RunStageTimeout:     2 * time.Hour,
	}
}

func createAuthSessionCookie(t *testing.T, db *ent.Client, secret string) string {
	t.Helper()
	ctx := context.Background()
	u, err := db.User.Create().
		SetEmail("owner@example.com").
		Save(ctx)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	token := "test-session-token-" + uuid.NewString()
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(token))
	hash := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if _, err := db.AuthSession.Create().
		SetUser(u).
		SetTokenHash(hash).
		SetExpiresAt(time.Now().Add(24 * time.Hour)).
		Save(ctx); err != nil {
		t.Fatalf("create auth session: %v", err)
	}
	return token
}

type fakeGHRunner struct {
	mu    sync.Mutex
	calls [][]string
}

func (f *fakeGHRunner) run(_ context.Context, args ...string) ([]byte, error) {
	f.mu.Lock()
	f.calls = append(f.calls, append([]string(nil), args...))
	f.mu.Unlock()
	return []byte(`{}`), nil
}

func (f *fakeGHRunner) snapshot() [][]string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([][]string, len(f.calls))
	copy(out, f.calls)
	return out
}

func initTestRepo(t *testing.T) (remote string) {
	t.Helper()
	ctx := context.Background()
	tmp := t.TempDir()
	remote = filepath.Join(tmp, "remote.git")
	seed := filepath.Join(tmp, "seed")
	runGit := func(dir string, args ...string) {
		t.Helper()
		c := exec.CommandContext(ctx, "git", args...)
		c.Dir = dir
		c.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=test",
			"GIT_AUTHOR_EMAIL=test@example.com",
			"GIT_COMMITTER_NAME=test",
			"GIT_COMMITTER_EMAIL=test@example.com",
		)
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, string(out))
		}
	}
	runGit(tmp, "init", "--bare", remote)
	runGit(tmp, "clone", remote, seed)
	if err := os.WriteFile(filepath.Join(seed, "README.md"), []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(seed, "add", "README.md")
	runGit(seed, "commit", "-m", "init")
	runGit(seed, "branch", "-M", "main")
	runGit(seed, "push", "-u", "origin", "main")
	return remote
}

func waitUntil(t *testing.T, timeout time.Duration, desc string, fn func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(15 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", desc)
}

func TestConfig_RunEnvVars(t *testing.T) {
	t.Setenv("RUN_POLL_INTERVAL", "12s")
	t.Setenv("RUN_STAGE_TIMEOUT", "45m")
	cfg, err := app.GetConfig(context.Background())
	if err != nil {
		t.Fatalf("GetConfig: %v", err)
	}
	if cfg.RunPollInterval != 12*time.Second {
		t.Fatalf("RunPollInterval = %v, want 12s", cfg.RunPollInterval)
	}
	if cfg.RunStageTimeout != 45*time.Minute {
		t.Fatalf("RunStageTimeout = %v, want 45m", cfg.RunStageTimeout)
	}
}

func TestServe_RestartRecovery(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db := openTestDB(t)
	cfg := testConfig(t)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	srv, err := buildServer(ctx, cfg, logger, db,
		withCapabilityProbe(func(context.Context) worker.Capabilities {
			return worker.Capabilities{Agy: worker.ToolStatusReady, Gh: worker.ToolStatusReady}
		}),
	)
	if err != nil {
		t.Fatalf("buildServer: %v", err)
	}

	proj, err := db.Project.Create().
		SetSlug("opus").
		SetName("Opus").
		SetRepoProvider("github").
		SetRepoRef("kilip/opus").
		SetTrackerProvider("github").
		SetTrackerRef("kilip/opus").
		SetDefaultBranch("main").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	tsk, err := srv.tasks.Import(ctx, task.ImportTaskRequest{
		ProjectID:  proj.ID.String(),
		Title:      "Stale run task",
		Status:     task.StatusInProgress,
		Source:     task.SourceGitHub,
		ExternalID: "100",
	})
	if err != nil {
		t.Fatalf("tasks.Import: %v", err)
	}

	seeded, err := srv.runs.Create(ctx, run.CreateInput{
		TaskID:    uuid.MustParse(tsk.ID),
		ProjectID: proj.ID,
		Mode:      run.ModeVibe,
	})
	if err != nil {
		t.Fatalf("runs.Create: %v", err)
	}
	if _, err := srv.runs.Transition(ctx, seeded.ID, run.StatePreparing, run.TransitionInput{}); err != nil {
		t.Fatalf("Transition preparing: %v", err)
	}
	if _, err := srv.runs.Transition(ctx, seeded.ID, run.StateRunning, run.TransitionInput{}); err != nil {
		t.Fatalf("Transition running: %v", err)
	}

	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.opus.Serve(ctx) }()

	waitUntil(t, 5*time.Second, "stale running run to fail on restart", func() bool {
		got, err := srv.runs.Get(context.Background(), seeded.ID)
		return err == nil && got.State == run.StateFailed
	})

	got, err := srv.runs.Get(ctx, seeded.ID)
	if err != nil {
		t.Fatalf("runs.Get: %v", err)
	}
	if got.FailureReason == nil || !strings.Contains(*got.FailureReason, "interrupted by server restart") {
		t.Fatalf("FailureReason = %v, want interrupted by server restart", got.FailureReason)
	}

	cancel()
	if err := <-serveErr; err != nil {
		t.Fatalf("Serve: %v", err)
	}
}

func TestServe_RunLifecycleCancelAndTrackerBus(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db := openTestDB(t)
	cfg := testConfig(t)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	remote := initTestRepo(t)
	fakeGH := &fakeGHRunner{}

	var (
		mu        sync.Mutex
		started   []worker.StartRun
		cancelled []string
	)
	srv, err := buildServer(ctx, cfg, logger, db,
		withTrackerRunner(fakeGH.run),
		withCapabilityProbe(func(context.Context) worker.Capabilities {
			return worker.Capabilities{Agy: worker.ToolStatusReady, Gh: worker.ToolStatusReady}
		}),
		withOnStartRun(func(runCtx context.Context, cmd worker.StartRun) error {
			mu.Lock()
			started = append(started, cmd)
			mu.Unlock()
			<-runCtx.Done()
			mu.Lock()
			cancelled = append(cancelled, cmd.RunID)
			mu.Unlock()
			return nil
		}),
	)
	if err != nil {
		t.Fatalf("buildServer: %v", err)
	}

	cookie := createAuthSessionCookie(t, db, cfg.SessionSecret)

	// AC1: GET /api/runs shares the exact runFeature wired into Opus.Serve.
	req := httptest.NewRequest(http.MethodGet, "/api/runs", nil)
	req.Header.Set("Cookie", "opus_session="+cookie)
	resp, err := srv.fiber.Test(req)
	if err != nil {
		t.Fatalf("GET /api/runs: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/runs status = %d, want 200", resp.StatusCode)
	}

	proj, err := db.Project.Create().
		SetSlug("opus").
		SetName("Opus").
		SetRepoProvider("github").
		SetRepoRef(remote).
		SetTrackerProvider("github").
		SetTrackerRef("kilip/opus").
		SetDefaultBranch("main").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	if _, err := db.ProjectNode.Create().
		SetProjectID(proj.ID).
		SetNodeID(uuid.MustParse(srv.masterID)).
		Save(ctx); err != nil {
		t.Fatalf("create project_node: %v", err)
	}

	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.opus.Serve(ctx) }()

	waitUntil(t, 5*time.Second, "master node agy ready", func() bool {
		return srv.nodes.IsAgyReady(srv.masterID)
	})

	tsk, err := srv.tasks.Import(ctx, task.ImportTaskRequest{
		ProjectID:   proj.ID.String(),
		Title:       "Implement wiring",
		Description: "## Scope\n- wire\n## Acceptance Criteria\n- wired\n## Definition of Done\n- `go test ./...`\n",
		Status:      task.StatusTodo,
		Labels:      []string{"vibe"},
		Source:      task.SourceGitHub,
		ExternalID:  "kilip/opus#42",
	})
	if err != nil {
		t.Fatalf("tasks.Import: %v", err)
	}

	// AC4: Eligible task -> run reaches running and OnStartRun receives StartRun.
	taskUUID := uuid.MustParse(tsk.ID)
	waitUntil(t, 5*time.Second, "first run to reach running and start", func() bool {
		mu.Lock()
		n := len(started)
		mu.Unlock()
		if n < 1 {
			return false
		}
		runs, err := srv.runs.List(context.Background(), run.ListFilter{TaskID: &taskUUID})
		return err == nil && len(runs) == 1 && runs[0].State == run.StateRunning
	})

	mu.Lock()
	firstCmd := started[0]
	mu.Unlock()
	if firstCmd.RunID == "" || firstCmd.SessionID == "" || firstCmd.Prompt == "" || firstCmd.Branch == "" {
		t.Fatalf("incomplete StartRun payload: %+v", firstCmd)
	}

	// Prepare a worktree dir to verify AC5 CleanupWorktree removes it when task moves to done.
	gitMgr := worker.NewGitManager(cfg.AppHome)
	repoDir, err := gitMgr.EnsureRepo(ctx, proj.Slug, remote, "")
	if err != nil {
		t.Fatalf("EnsureRepo: %v", err)
	}
	wtDir, err := gitMgr.PrepareWorktree(ctx, repoDir, proj.Slug, tsk.Key, firstCmd.Branch, "main")
	if err != nil {
		t.Fatalf("PrepareWorktree: %v", err)
	}

	// Complete session first so second task can run cleanly later, then mark task done.
	if _, _, err := srv.sessions.Complete(ctx, uuid.MustParse(firstCmd.SessionID)); err != nil {
		t.Fatalf("sessions.Complete: %v", err)
	}
	doneStatus := task.StatusDone
	if _, err := srv.tasks.Update(ctx, tsk.ID, task.UpdateTaskRequest{Status: &doneStatus}); err != nil {
		t.Fatalf("tasks.Update done: %v", err)
	}

	// AC5: Task done -> run merged + CleanupWorktree removes worktree.
	waitUntil(t, 5*time.Second, "run merged and worktree cleaned", func() bool {
		got, err := srv.runs.Get(context.Background(), uuid.MustParse(firstCmd.RunID))
		if err != nil || got.State != run.StateMerged {
			return false
		}
		_, statErr := os.Stat(wtDir)
		return os.IsNotExist(statErr)
	})

	// AC7: Tracker bus subscriber reacted to task.status_changed and invoked fakeGH issue command.
	waitUntil(t, 5*time.Second, "tracker outbound gh issue call", func() bool {
		for _, args := range fakeGH.snapshot() {
			if len(args) >= 2 && args[0] == "issue" && args[1] == "view" {
				return true
			}
		}
		return false
	})

	// AC6: Second task -> running -> POST /api/runs/:id/cancel -> SendCancelRun on same transport.
	if _, err := srv.tasks.Import(ctx, task.ImportTaskRequest{
		ProjectID:   proj.ID.String(),
		Title:       "Cancellable task",
		Description: "## Scope\n- cancel\n## Acceptance Criteria\n- cancelled\n## Definition of Done\n- `go test ./...`\n",
		Status:      task.StatusTodo,
		Labels:      []string{"vibe"},
		Source:      task.SourceGitHub,
		ExternalID:  "43",
	}); err != nil {
		t.Fatalf("tasks.Import 2: %v", err)
	}
	waitUntil(t, 5*time.Second, "second run to start", func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(started) >= 2
	})
	mu.Lock()
	secondCmd := started[1]
	mu.Unlock()

	cancelReq := httptest.NewRequest(http.MethodPost, "/api/runs/"+secondCmd.RunID+"/cancel", bytes.NewBufferString(`{"reason":"test cancel"}`))
	cancelReq.Header.Set("Content-Type", "application/json")
	cancelReq.Header.Set("Cookie", "opus_session="+cookie)
	cancelResp, err := srv.fiber.Test(cancelReq)
	if err != nil {
		t.Fatalf("POST /api/runs/:id/cancel: %v", err)
	}
	var cancelBody httpx.Response[run.RunResponse]
	_ = json.NewDecoder(cancelResp.Body).Decode(&cancelBody)
	_ = cancelResp.Body.Close()
	if cancelResp.StatusCode != http.StatusOK || cancelBody.Data.State != run.StateCancelled {
		t.Fatalf("cancel response status=%d state=%s, want 200 cancelled", cancelResp.StatusCode, cancelBody.Data.State)
	}

	waitUntil(t, 5*time.Second, "worker OnStartRun context cancelled", func() bool {
		mu.Lock()
		defer mu.Unlock()
		for _, id := range cancelled {
			if id == secondCmd.RunID {
				return true
			}
		}
		return false
	})

	cancel()
	if err := <-serveErr; err != nil {
		t.Fatalf("Serve: %v", err)
	}
}

func TestServe_StageTimeoutFromConfig(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db := openTestDB(t)
	cfg := testConfig(t)
	cfg.RunPollInterval = 20 * time.Millisecond
	cfg.RunStageTimeout = 50 * time.Millisecond
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	srv, err := buildServer(ctx, cfg, logger, db,
		withCapabilityProbe(func(context.Context) worker.Capabilities {
			return worker.Capabilities{Agy: worker.ToolStatusReady, Gh: worker.ToolStatusReady}
		}),
	)
	if err != nil {
		t.Fatalf("buildServer: %v", err)
	}

	proj, err := db.Project.Create().
		SetSlug("opus").
		SetName("Opus").
		SetRepoProvider("github").
		SetRepoRef("kilip/opus").
		SetTrackerProvider("github").
		SetTrackerRef("kilip/opus").
		SetDefaultBranch("main").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	tsk, err := srv.tasks.Import(ctx, task.ImportTaskRequest{
		ProjectID:  proj.ID.String(),
		Title:      "Timeout task",
		Status:     task.StatusTodo,
		Source:     task.SourceGitHub,
		ExternalID: "200",
	})
	if err != nil {
		t.Fatalf("tasks.Import: %v", err)
	}
	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.opus.Serve(ctx) }()
	// Let startup RecoverActiveRuns finish before creating the preparing run.
	time.Sleep(80 * time.Millisecond)

	seeded, err := srv.runs.Create(ctx, run.CreateInput{
		TaskID:    uuid.MustParse(tsk.ID),
		ProjectID: proj.ID,
		Mode:      run.ModeVibe,
	})
	if err != nil {
		t.Fatalf("runs.Create: %v", err)
	}
	if _, err := srv.runs.Transition(ctx, seeded.ID, run.StatePreparing, run.TransitionInput{}); err != nil {
		t.Fatalf("Transition preparing: %v", err)
	}

	waitUntil(t, 5*time.Second, "preparing run to fail on stage timeout", func() bool {
		got, err := srv.runs.Get(context.Background(), seeded.ID)
		return err == nil && got.State == run.StateFailed
	})

	got, err := srv.runs.Get(ctx, seeded.ID)
	if err != nil {
		t.Fatalf("runs.Get: %v", err)
	}
	if got.FailureReason == nil || !strings.Contains(*got.FailureReason, "stage timeout") {
		t.Fatalf("FailureReason = %v, want stage timeout", got.FailureReason)
	}

	cancel()
	if err := <-serveErr; err != nil {
		t.Fatalf("Serve: %v", err)
	}
}

func TestServe_WorkerRunErrorAndDisconnectFailRun(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db := openTestDB(t)
	cfg := testConfig(t)
	cfg.RunPollInterval = 20 * time.Millisecond
	cfg.RunStageTimeout = time.Hour
	cfg.RunNodeGrace = 50 * time.Millisecond
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	remote := initTestRepo(t)

	srv, err := buildServer(ctx, cfg, logger, db,
		withCapabilityProbe(func(context.Context) worker.Capabilities {
			return worker.Capabilities{Agy: worker.ToolStatusReady, Gh: worker.ToolStatusReady}
		}),
		withOnStartRun(func(_ context.Context, cmd worker.StartRun) error {
			return fmt.Errorf("simulated worker error for %s", cmd.RunID)
		}),
	)
	if err != nil {
		t.Fatalf("buildServer: %v", err)
	}

	proj, err := db.Project.Create().
		SetSlug("opus-err").
		SetName("Opus Error").
		SetRepoProvider("github").
		SetRepoRef(remote).
		SetTrackerProvider("github").
		SetTrackerRef("kilip/opus").
		SetDefaultBranch("main").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	if _, err := db.ProjectNode.Create().
		SetProjectID(proj.ID).
		SetNodeID(uuid.MustParse(srv.masterID)).
		SetPath(t.TempDir()).
		Save(ctx); err != nil {
		t.Fatalf("create project node mapping: %v", err)
	}

	tsk, err := srv.tasks.Import(ctx, task.ImportTaskRequest{
		ProjectID:   proj.ID.String(),
		Title:       "Worker error task",
		Description: "## Scope\nVerify run_error propagates\n\n## Acceptance Criteria\n- Fails run\n\n## Definition of Done\n- `go test ./...`\n",
		Status:      task.StatusTodo,
		Labels:      []string{"vibe"},
		Source:      task.SourceGitHub,
		ExternalID:  "300",
	})
	if err != nil {
		t.Fatalf("tasks.Import: %v", err)
	}
	taskUUID := uuid.MustParse(tsk.ID)

	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.opus.Serve(ctx) }()

	waitUntil(t, 5*time.Second, "master node agy ready", func() bool {
		return srv.nodes.IsAgyReady(srv.masterID)
	})

	waitUntil(t, 5*time.Second, "run to fail on worker run_error", func() bool {
		runs, err := srv.runs.List(context.Background(), run.ListFilter{TaskID: &taskUUID})
		return err == nil && len(runs) == 1 && runs[0].State == run.StateFailed
	})

	// Also verify node.offline + disconnect grace fails an active preparing/running run.
	tsk2, err := srv.tasks.Import(ctx, task.ImportTaskRequest{
		ProjectID:  proj.ID.String(),
		Title:      "Disconnect grace task",
		Status:     task.StatusTodo,
		Source:     task.SourceGitHub,
		ExternalID: "301",
	})
	if err != nil {
		t.Fatalf("tasks.Import 2: %v", err)
	}
	offlineNode, err := db.Node.Create().
		SetName("worker-offline-test").
		SetRole("worker").
		SetStatus("offline").
		Save(ctx)
	if err != nil {
		t.Fatalf("create node: %v", err)
	}
	seeded, err := srv.runs.Create(ctx, run.CreateInput{
		TaskID:    uuid.MustParse(tsk2.ID),
		ProjectID: proj.ID,
		NodeID:    &offlineNode.ID,
		Mode:      run.ModeVibe,
	})
	if err != nil {
		t.Fatalf("runs.Create: %v", err)
	}
	if _, err := srv.runs.Transition(ctx, seeded.ID, run.StatePreparing, run.TransitionInput{}); err != nil {
		t.Fatalf("Transition preparing: %v", err)
	}
	payload, _ := json.Marshal(node.NodeEventPayload{ID: offlineNode.ID.String(), Name: offlineNode.Name})
	if err := srv.bus.Publish(ctx, eventbus.Event{Type: node.EventOffline, Payload: payload}); err != nil {
		t.Fatalf("bus.Publish node.offline: %v", err)
	}

	waitUntil(t, 5*time.Second, "preparing run to fail after RUN_NODE_GRACE", func() bool {
		got, err := srv.runs.Get(context.Background(), seeded.ID)
		return err == nil && got.State == run.StateFailed
	})

	cancel()
	if err := <-serveErr; err != nil {
		t.Fatalf("Serve: %v", err)
	}
}
