package service_test

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kilip/opus/internal/platform/eventbus"
	"github.com/kilip/opus/internal/project"
	"github.com/kilip/opus/internal/run/domain"
	"github.com/kilip/opus/internal/run/service"
	"github.com/kilip/opus/internal/session"
	"github.com/kilip/opus/internal/task"
	"github.com/kilip/opus/internal/worker"
)

type fakeWorkerTransport struct {
	mu        sync.Mutex
	started   []worker.StartRun
	cancelled []worker.CancelRun
	cleanups  []string
}

func (f *fakeWorkerTransport) SendStartRun(_ context.Context, _ uuid.UUID, cmd worker.StartRun) error {
	f.mu.Lock()
	f.started = append(f.started, cmd)
	f.mu.Unlock()
	return nil
}

func (f *fakeWorkerTransport) SendCancelRun(_ context.Context, _ uuid.UUID, cmd worker.CancelRun) error {
	f.mu.Lock()
	f.cancelled = append(f.cancelled, cmd)
	f.mu.Unlock()
	return nil
}

func (f *fakeWorkerTransport) CleanupWorktree(_ context.Context, _ uuid.UUID, projectSlug, taskKey, branch, _ string) error {
	f.mu.Lock()
	f.cleanups = append(f.cleanups, projectSlug+"/"+taskKey+":"+branch)
	f.mu.Unlock()
	return nil
}

func TestStages_ClaimPrepareRunAndCleanup(t *testing.T) {
	ctx := context.Background()
	svc, client, bus, tasks, sessions, projID, nodeID := setupReconcilerEnv(t)
	projects := project.New(client)
	fw := &fakeWorkerTransport{}

	var (
		mu     sync.Mutex
		events []eventbus.Event
	)
	_ = bus.Subscribe("test-stage-events", func(_ context.Context, e eventbus.Event) error {
		mu.Lock()
		events = append(events, e)
		mu.Unlock()
		return nil
	}, domain.EventStarted, domain.EventFinished)

	rec := importGitHubTask(t, tasks, projID, "122", task.StatusTodo, []string{"vibe", "type:feat"}, validDescription)

	run, err := svc.Create(ctx, domain.CreateInput{
		TaskID:    uuid.MustParse(rec.ID),
		ProjectID: projID,
		NodeID:    &nodeID,
		Mode:      domain.ModeVibe,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	cfg := service.StageConfig{
		AppHome:  "/tmp/opus-home",
		Tasks:    tasks,
		Projects: projects,
		Sessions: sessions,
		Worker:   fw,
	}

	// 1. Claim: preparing + task in-progress + run.started.
	run, err = svc.Claim(ctx, cfg, run.ID)
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}
	if run.State != domain.StatePreparing || run.StageAttempts != 1 {
		t.Fatalf("after Claim: %+v", run)
	}
	gotTask, err := tasks.Get(ctx, rec.ID)
	if err != nil {
		t.Fatalf("tasks.Get: %v", err)
	}
	if gotTask.Status != task.StatusInProgress {
		t.Fatalf("task status = %s, want in-progress", gotTask.Status)
	}

	// 2. Prepare & Start Session: running + session enqueued + StartRun dispatched.
	run, err = svc.PrepareAndStartSession(ctx, cfg, run.ID)
	if err != nil {
		t.Fatalf("PrepareAndStartSession: %v", err)
	}
	if run.State != domain.StateRunning || run.SessionID == nil {
		t.Fatalf("after PrepareAndStartSession: %+v", run)
	}
	if !strings.HasPrefix(run.Branch, "feat/issue-122") {
		t.Fatalf("unexpected derived branch: %q", run.Branch)
	}

	fw.mu.Lock()
	if len(fw.started) != 1 {
		fw.mu.Unlock()
		t.Fatalf("expected 1 StartRun command, got %d", len(fw.started))
	}
	cmd := fw.started[0]
	fw.mu.Unlock()

	wantWT := filepath.Join("/tmp/opus-home", "wt", gotTask.Key[:0]+"", rec.Key)
	_ = wantWT
	if cmd.Policy == nil || !strings.Contains(cmd.Policy.WorktreeRoot, filepath.Join("wt")) {
		t.Fatalf("expected worktreeRoot policy, got %+v", cmd.Policy)
	}
	if !strings.HasPrefix(cmd.Prompt, "/vibe 122\n") {
		t.Fatalf("expected /vibe 122 prompt, got %q", cmd.Prompt)
	}

	// 3. Complete & Cleanup via task.status_changed -> done.
	doneStatus := task.StatusDone
	if _, err := tasks.Update(ctx, rec.ID, task.UpdateTaskRequest{Status: &doneStatus}); err != nil {
		t.Fatalf("tasks.Update done: %v", err)
	}
	donePayload, _ := json.Marshal(task.TaskStatusChangedPayload{
		ID:        rec.ID,
		ProjectID: projID.String(),
		From:      task.StatusInProgress,
		To:        task.StatusDone,
	})
	if err := svc.HandleTaskStatusChanged(ctx, cfg, eventbus.Event{
		Type:    task.EventStatusChanged,
		Payload: donePayload,
	}); err != nil {
		t.Fatalf("HandleTaskStatusChanged: %v", err)
	}

	finalRun, err := svc.Get(ctx, run.ID)
	if err != nil {
		t.Fatalf("Get final run: %v", err)
	}
	if finalRun.State != domain.StateMerged || finalRun.EndedAt == nil {
		t.Fatalf("expected merged run with EndedAt, got %+v", finalRun)
	}

	fw.mu.Lock()
	cleanups := len(fw.cleanups)
	fw.mu.Unlock()
	if cleanups != 1 {
		t.Fatalf("expected 1 worktree cleanup, got %d", cleanups)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		n := len(events)
		mu.Unlock()
		if n >= 2 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(events) != 2 || events[0].Type != domain.EventStarted || events[1].Type != domain.EventFinished {
		t.Fatalf("expected [run.started, run.finished], got %+v", events)
	}
}

type failingStartWorker struct {
	fakeWorkerTransport
	err error
}

func (f *failingStartWorker) SendStartRun(context.Context, uuid.UUID, worker.StartRun) error {
	return f.err
}

func TestPrepareAndStartSession_DispatchFailure(t *testing.T) {
	ctx := context.Background()
	svc, client, _, tasks, sessions, projID, nodeID := setupReconcilerEnv(t)
	fw := &failingStartWorker{err: fmt.Errorf("node unreachable")}
	cfg := service.StageConfig{
		AppHome:  t.TempDir(),
		Tasks:    tasks,
		Projects: project.New(client),
		Sessions: sessions,
		Worker:   fw,
	}

	rec := importGitHubTask(t, tasks, projID, "502", task.StatusTodo, []string{"vibe"}, validDescription)
	run, err := svc.Create(ctx, domain.CreateInput{
		TaskID:    uuid.MustParse(rec.ID),
		ProjectID: projID,
		NodeID:    &nodeID,
		Mode:      domain.ModeVibe,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := svc.Claim(ctx, cfg, run.ID); err != nil {
		t.Fatalf("Claim: %v", err)
	}
	if _, err := svc.PrepareAndStartSession(ctx, cfg, run.ID); err == nil {
		t.Fatal("expected PrepareAndStartSession to fail on SendStartRun error")
	}

	got, err := svc.Get(ctx, run.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.State != domain.StateFailed || got.FailureReason == nil || !strings.Contains(*got.FailureReason, "failed to dispatch to worker") {
		t.Fatalf("expected failed run with dispatch reason, got %+v", got)
	}
	if got.SessionID != nil {
		sess, err := sessions.Get(ctx, *got.SessionID)
		if err != nil || sess.Status != session.StatusFailed {
			t.Fatalf("expected session failed, got %+v (err=%v)", sess, err)
		}
	}
	fw.mu.Lock()
	cleanups := len(fw.cleanups)
	fw.mu.Unlock()
	if cleanups != 1 {
		t.Fatalf("expected best-effort worktree cleanup on dispatch failure, got %d", cleanups)
	}
}
