package worker_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/kilip/opus/ent"
	"github.com/kilip/opus/internal/node"
	"github.com/kilip/opus/internal/platform/eventbus"
	"github.com/kilip/opus/internal/session"
	"github.com/kilip/opus/internal/worker"
	_ "modernc.org/sqlite"
)

func writeFakeAgy(t *testing.T, mode, stateFile string) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "agy")
	script := `#!/usr/bin/env python3
import json, os, sys, threading

mode = sys.argv[1] if False else os.environ.get("FAKE_AGY_MODE", "ok")
state_file = os.environ.get("FAKE_AGY_STATE", "")

if len(sys.argv) >= 2 and sys.argv[1] == "models":
    sys.exit(1 if mode == "unauth" else 0)

def write_msg(obj):
    sys.stdout.write(json.dumps(obj) + "\n")
    sys.stdout.flush()

pending_prompt_id = None
cancel_event = threading.Event()

for line in sys.stdin:
    line = line.strip()
    if not line:
        continue
    msg = json.loads(line)
    mid = msg.get("id")
    method = msg.get("method", "")
    params = msg.get("params") or {}

    if method == "initialize":
        write_msg({"jsonrpc": "2.0", "id": mid, "result": {"protocolVersion": 1}})
    elif method == "session/new":
        if state_file:
            with open(state_file, "a") as f:
                f.write(json.dumps({"event": "session_new", "cwd": os.getcwd(), "paramCwd": params.get("cwd", "")}) + "\n")
        write_msg({"jsonrpc": "2.0", "id": mid, "result": {"sessionId": "acp-sess-1"}})
    elif method == "session/prompt":
        prompt_text = ""
        for p in params.get("prompt", []):
            if p.get("type") == "text":
                prompt_text = p.get("text", "")
        if state_file:
            with open(state_file, "a") as f:
                f.write(json.dumps({"event": "prompt", "text": prompt_text}) + "\n")
        if mode == "fail":
            write_msg({"jsonrpc": "2.0", "id": mid, "error": {"code": -32000, "message": "boom Bearer super-secret-cred"}})
        elif mode == "cancel":
            pending_prompt_id = mid
            def wait_cancel(pid):
                cancel_event.wait(timeout=10)
                write_msg({"jsonrpc": "2.0", "id": pid, "result": {"stopReason": "cancelled"}})
            threading.Thread(target=wait_cancel, args=(mid,), daemon=True).start()
        else:
            write_msg({"jsonrpc": "2.0", "method": "session/update", "params": {"sessionId": "acp-sess-1", "update": {"sessionUpdate": "agent_message_chunk", "title": "step 1"}}})
            write_msg({"jsonrpc": "2.0", "method": "session/update", "params": {"sessionId": "acp-sess-1", "update": {"sessionUpdate": "agent_message_chunk", "title": "step 2"}}})
            write_msg({"jsonrpc": "2.0", "id": mid, "result": {"stopReason": "end_turn"}})
    elif method == "session/cancel":
        cancel_event.set()
        if mid is not None:
            write_msg({"jsonrpc": "2.0", "id": mid, "result": {}})
`
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake agy: %v", err)
	}
	t.Setenv("FAKE_AGY_MODE", mode)
	t.Setenv("FAKE_AGY_STATE", stateFile)
	return bin
}

func setupRunnerRemoteRepo(t *testing.T) string {
	t.Helper()
	ctx := context.Background()
	tmp := t.TempDir()
	remote := filepath.Join(tmp, "remote.git")
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
	if err := os.WriteFile(filepath.Join(seed, "README.md"), []byte("init\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(seed, "add", "README.md")
	runGit(seed, "commit", "-m", "init")
	runGit(seed, "branch", "-M", "main")
	runGit(seed, "push", "-u", "origin", "main")
	return remote
}

func openRunnerSessionDB(t *testing.T) *ent.Client {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "runner.db")
	sqlDB, err := sql.Open("sqlite", "file:"+dbPath+"?_pragma=foreign_keys(ON)&_pragma=busy_timeout(5000)")
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

func drainHello(t *testing.T, tr *worker.InProcessTransport) {
	t.Helper()
	select {
	case env := <-tr.Events:
		if env.Type != node.EnvelopeTypeHello {
			t.Fatalf("expected hello, got %s", env.Type)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for hello")
	}
}

func TestRunner_StartRunSuccessSessionEventsAndCleanup(t *testing.T) {
	appHome := t.TempDir()
	t.Setenv("APP_HOME", appHome)
	stateFile := filepath.Join(t.TempDir(), "state.jsonl")
	fakeAgy := writeFakeAgy(t, "ok", stateFile)
	remote := setupRunnerRemoteRepo(t)

	db := openRunnerSessionDB(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Seed project, task, node, and session so HandleWorkerEvent persists updates and completion.
	proj, err := db.Project.Create().
		SetSlug("opus").
		SetName("Opus").
		SetRepoProvider("github").
		SetRepoRef(remote).
		SetTrackerProvider("github").
		SetTrackerRef("kilip/opus").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	tsk, err := db.Task.Create().
		SetProjectID(proj.ID).
		SetKey("OP-1").
		SetTitle("Test runner").
		Save(ctx)
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	nd, err := db.Node.Create().
		SetName("master").
		SetRole("master").
		SetStatus("online").
		Save(ctx)
	if err != nil {
		t.Fatalf("create node: %v", err)
	}
	sessSvc := session.New(db, session.Config{Bus: eventbus.New()})
	sessRec, err := sessSvc.Enqueue(ctx, session.CreateInput{
		TaskID:   tsk.ID,
		NodeID:   nd.ID,
		Provider: "agy",
	})
	if err != nil {
		t.Fatalf("Enqueue session: %v", err)
	}

	tr := worker.NewInProcess()
	defer func() { _ = tr.Close() }()

	// OnStartRun intentionally nil to verify worker.New binds default w.runStart (AC9).
	w := worker.New(worker.Config{
		NodeID:            nd.ID.String(),
		HomeDir:           appHome,
		AgyBin:            fakeAgy,
		InProcess:         tr,
		HeartbeatInterval: time.Hour,
		Logger:            slog.New(slog.NewTextHandler(io.Discard, nil)),
	})

	runDone := make(chan error, 1)
	go func() { runDone <- w.Run(ctx) }()
	drainHello(t, tr)

	runID := uuid.NewString()
	cmd := worker.StartRun{
		RunID:         runID,
		SessionID:     sessRec.ID.String(),
		TaskID:        tsk.ID.String(),
		TaskKey:       "OP-1",
		ProjectID:     proj.ID.String(),
		ProjectSlug:   "opus",
		RepoURL:       remote,
		DefaultBranch: "main",
		Branch:        "vibe/OP-1",
		Prompt:        "/vibe 1\nimplement feature",
	}
	startEnv, err := worker.NewStartRunEnvelope("cmd-start-1", cmd)
	if err != nil {
		t.Fatalf("NewStartRunEnvelope: %v", err)
	}
	if err := tr.SendCommand(ctx, startEnv); err != nil {
		t.Fatalf("SendCommand start_run: %v", err)
	}

	var updateCount int
	completed := false
	deadline := time.After(10 * time.Second)
	for !completed {
		select {
		case env := <-tr.Events:
			if env.Type != node.EnvelopeTypeEvent {
				continue
			}
			var evt worker.Event
			if err := json.Unmarshal(env.Payload, &evt); err != nil {
				t.Fatalf("unmarshal event: %v", err)
			}
			if evt.SessionID != sessRec.ID.String() {
				t.Fatalf("event %q missing/wrong sessionId: %q", evt.Type, evt.SessionID)
			}
			sessSvc.HandleWorkerEvent(ctx, nd.ID.String(), env)
			switch evt.Type {
			case worker.EventSessionUpdate:
				updateCount++
			case worker.EventSessionCompleted:
				completed = true
			case worker.EventRunError:
				t.Fatalf("unexpected run_error: %s", evt.Error)
			}
		case <-deadline:
			t.Fatal("timed out waiting for session_completed")
		}
	}
	if updateCount != 2 {
		t.Fatalf("updateCount = %d, want 2", updateCount)
	}

	// Verify worktree exists on cmd.Branch and fake agy received cwd + prompt.
	wtDir := filepath.Join(appHome, "wt", "opus", "OP-1")
	if _, err := os.Stat(wtDir); err != nil {
		t.Fatalf("expected worktree %s to exist: %v", wtDir, err)
	}
	stateRaw, err := os.ReadFile(stateFile)
	if err != nil {
		t.Fatalf("read stateFile: %v", err)
	}
	if !strings.Contains(string(stateRaw), wtDir) || !strings.Contains(string(stateRaw), "/vibe 1") {
		t.Fatalf("unexpected fake agy state:\n%s", string(stateRaw))
	}

	// Verify session service persisted updates and completed status.
	events, err := sessSvc.ListEventsAfter(ctx, sessRec.ID, 0)
	if err != nil {
		t.Fatalf("ListEventsAfter: %v", err)
	}
	var sessionUpdates []string
	for _, ev := range events {
		if ev.Kind == worker.EventSessionUpdate {
			sessionUpdates = append(sessionUpdates, string(ev.Payload))
		}
	}
	if len(sessionUpdates) != 2 || !strings.Contains(sessionUpdates[0], "step 1") || !strings.Contains(sessionUpdates[1], "step 2") {
		t.Fatalf("unexpected persisted session/update events: %+v", sessionUpdates)
	}
	gotSess, err := sessSvc.Get(ctx, sessRec.ID)
	if err != nil {
		t.Fatalf("sessSvc.Get: %v", err)
	}
	if gotSess.Status != session.StatusCompleted {
		t.Fatalf("session status = %s, want completed", gotSess.Status)
	}

	// AC10: cleanup_worktree removes worktree dir and local branch.
	cleanupEnv, err := worker.NewCleanupWorktreeEnvelope("cmd-cleanup-1", worker.CleanupWorktree{
		ProjectSlug: "opus",
		TaskKey:     "OP-1",
		Branch:      "vibe/OP-1",
	})
	if err != nil {
		t.Fatalf("NewCleanupWorktreeEnvelope: %v", err)
	}
	if err := tr.SendCommand(ctx, cleanupEnv); err != nil {
		t.Fatalf("SendCommand cleanup_worktree: %v", err)
	}
	cleanDeadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(cleanDeadline) {
		if _, err := os.Stat(wtDir); os.IsNotExist(err) {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if _, err := os.Stat(wtDir); !os.IsNotExist(err) {
		t.Fatalf("expected worktree %s to be removed", wtDir)
	}

	cancel()
	if err := <-runDone; err != nil {
		t.Fatalf("w.Run: %v", err)
	}
}

func TestRunner_FailureScrubsCredential(t *testing.T) {
	appHome := t.TempDir()
	t.Setenv("APP_HOME", appHome)
	fakeAgy := writeFakeAgy(t, "fail", "")
	remote := setupRunnerRemoteRepo(t)

	tr := worker.NewInProcess()
	defer func() { _ = tr.Close() }()

	w := worker.New(worker.Config{
		NodeID:            "node-1",
		Credential:        "super-secret-cred",
		HomeDir:           appHome,
		AgyBin:            fakeAgy,
		InProcess:         tr,
		HeartbeatInterval: time.Hour,
		Logger:            slog.New(slog.NewTextHandler(io.Discard, nil)),
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runDone := make(chan error, 1)
	go func() { runDone <- w.Run(ctx) }()
	drainHello(t, tr)

	startEnv, _ := worker.NewStartRunEnvelope("cmd-fail", worker.StartRun{
		RunID:         "run-fail",
		SessionID:     "sess-fail",
		TaskKey:       "OP-2",
		ProjectSlug:   "opus",
		RepoURL:       remote,
		DefaultBranch: "main",
		Branch:        "vibe/OP-2",
		Prompt:        "fail please",
	})
	_ = tr.SendCommand(ctx, startEnv)

	select {
	case env := <-tr.Events:
		var evt worker.Event
		if err := json.Unmarshal(env.Payload, &evt); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if evt.Type != worker.EventRunError || evt.SessionID != "sess-fail" {
			t.Fatalf("got %+v, want run_error with sessionId=sess-fail", evt)
		}
		if strings.Contains(evt.Error, "super-secret-cred") {
			t.Fatalf("credential leaked in error: %q", evt.Error)
		}
	case <-time.After(8 * time.Second):
		t.Fatal("timed out waiting for run_error")
	}

	cancel()
	if err := <-runDone; err != nil {
		t.Fatalf("w.Run: %v", err)
	}
}

func TestRunner_CancelAndDuplicate(t *testing.T) {
	appHome := t.TempDir()
	t.Setenv("APP_HOME", appHome)
	stateFile := filepath.Join(t.TempDir(), "state.jsonl")
	fakeAgy := writeFakeAgy(t, "cancel", stateFile)
	remote := setupRunnerRemoteRepo(t)

	tr := worker.NewInProcess()
	defer func() { _ = tr.Close() }()

	w := worker.New(worker.Config{
		NodeID:            "node-1",
		HomeDir:           appHome,
		AgyBin:            fakeAgy,
		InProcess:         tr,
		HeartbeatInterval: time.Hour,
		Logger:            slog.New(slog.NewTextHandler(io.Discard, nil)),
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runDone := make(chan error, 1)
	go func() { runDone <- w.Run(ctx) }()
	drainHello(t, tr)

	startEnv, _ := worker.NewStartRunEnvelope("cmd-cancel-1", worker.StartRun{
		RunID:         "run-cancel",
		SessionID:     "sess-cancel",
		TaskKey:       "OP-3",
		ProjectSlug:   "opus",
		RepoURL:       remote,
		DefaultBranch: "main",
		Branch:        "vibe/OP-3",
		Prompt:        "long prompt",
	})
	_ = tr.SendCommand(ctx, startEnv)

	// Wait until agy has started session/prompt.
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		if raw, err := os.ReadFile(stateFile); err == nil && strings.Contains(string(raw), `"event": "prompt"`) {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	// AC7: Duplicate RunID emits run_error with sessionId.
	dupEnv, _ := worker.NewStartRunEnvelope("cmd-dup", worker.StartRun{
		RunID:     "run-cancel",
		SessionID: "sess-cancel",
	})
	_ = tr.SendCommand(ctx, dupEnv)
	select {
	case env := <-tr.Events:
		var evt worker.Event
		_ = json.Unmarshal(env.Payload, &evt)
		if evt.Type != worker.EventRunError || evt.SessionID != "sess-cancel" || !strings.Contains(evt.Error, "run already active") {
			t.Fatalf("unexpected duplicate event: %+v", evt)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for duplicate run_error")
	}

	// AC6: cancel_run stops agy and emits run_cancelled with sessionId and reason.
	cancelEnv, _ := worker.NewCancelRunEnvelope("cmd-cancel", worker.CancelRun{
		RunID:  "run-cancel",
		Reason: "user abort",
	})
	_ = tr.SendCommand(ctx, cancelEnv)
	select {
	case env := <-tr.Events:
		var evt worker.Event
		_ = json.Unmarshal(env.Payload, &evt)
		if evt.Type != worker.EventRunCancelled || evt.SessionID != "sess-cancel" || evt.Reason != "user abort" {
			t.Fatalf("unexpected cancel event: %+v", evt)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for run_cancelled")
	}

	cancel()
	if err := <-runDone; err != nil {
		t.Fatalf("w.Run: %v", err)
	}
}

func TestRunner_PreflightMissingAndUnauthenticated(t *testing.T) {
	for _, tc := range []struct {
		name    string
		mode    string
		bin     string
		wantErr string
	}{
		{name: "missing", mode: "ok", bin: "/nonexistent/agy-missing-bin", wantErr: "agy is not installed"},
		{name: "unauthenticated", mode: "unauth", wantErr: "agy is not authenticated"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			appHome := t.TempDir()
			t.Setenv("APP_HOME", appHome)
			bin := tc.bin
			if bin == "" {
				bin = writeFakeAgy(t, tc.mode, "")
			}

			tr := worker.NewInProcess()
			defer func() { _ = tr.Close() }()

			w := worker.New(worker.Config{
				NodeID:            "node-1",
				HomeDir:           appHome,
				AgyBin:            bin,
				InProcess:         tr,
				HeartbeatInterval: time.Hour,
				Logger:            slog.New(slog.NewTextHandler(io.Discard, nil)),
			})

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			runDone := make(chan error, 1)
			go func() { runDone <- w.Run(ctx) }()
			drainHello(t, tr)

			start := time.Now()
			startEnv, _ := worker.NewStartRunEnvelope("cmd-preflight", worker.StartRun{
				RunID:       "run-pre",
				SessionID:   "sess-pre",
				TaskKey:     "OP-4",
				ProjectSlug: "opus",
				Branch:      "vibe/OP-4",
			})
			_ = tr.SendCommand(ctx, startEnv)

			select {
			case env := <-tr.Events:
				var evt worker.Event
				_ = json.Unmarshal(env.Payload, &evt)
				if evt.Type != worker.EventRunError || evt.SessionID != "sess-pre" || !strings.Contains(evt.Error, tc.wantErr) {
					t.Fatalf("got %+v, want error containing %q", evt, tc.wantErr)
				}
				if time.Since(start) >= 5*time.Second {
					t.Fatalf("preflight took too long: %v", time.Since(start))
				}
			case <-time.After(5 * time.Second):
				t.Fatal("timed out waiting for preflight run_error")
			}

			// Verify no worktree was created.
			wtDir := filepath.Join(appHome, "wt", "opus", "OP-4")
			if _, err := os.Stat(wtDir); !os.IsNotExist(err) {
				t.Fatalf("expected no worktree created on preflight failure, stat=%v", err)
			}

			cancel()
			if err := <-runDone; err != nil {
				t.Fatalf("w.Run: %v", err)
			}
		})
	}
}
