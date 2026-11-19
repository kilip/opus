package service_test

import (
	"context"
	"encoding/json"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kilip/opus/internal/platform/eventbus"
	"github.com/kilip/opus/internal/project"
	"github.com/kilip/opus/internal/run/domain"
	"github.com/kilip/opus/internal/run/repo"
	"github.com/kilip/opus/internal/run/service"
	"github.com/kilip/opus/internal/session"
	"github.com/kilip/opus/internal/task"
)

func TestResilience_FailRunPreservesWorktreeAndLabelsTask(t *testing.T) {
	ctx := context.Background()
	svc, client, bus, tasks, sessions, projID, nodeID := setupReconcilerEnv(t)
	projects := project.New(client)
	fw := &fakeWorkerTransport{}

	var (
		mu       sync.Mutex
		finished []domain.RunFinishedPayload
	)
	_ = bus.Subscribe("test-fail-events", func(_ context.Context, e eventbus.Event) error {
		var p domain.RunFinishedPayload
		if err := json.Unmarshal(e.Payload, &p); err == nil {
			mu.Lock()
			finished = append(finished, p)
			mu.Unlock()
		}
		return nil
	}, domain.EventFinished)

	rec := importGitHubTask(t, tasks, projID, "501", task.StatusInProgress, []string{"vibe"}, validDescription)
	run, err := svc.Create(ctx, domain.CreateInput{
		TaskID:    uuid.MustParse(rec.ID),
		ProjectID: projID,
		NodeID:    &nodeID,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := svc.Transition(ctx, run.ID, domain.StatePreparing, domain.TransitionInput{}); err != nil {
		t.Fatalf("Transition preparing: %v", err)
	}

	cfg := service.StageConfig{
		Tasks:    tasks,
		Projects: projects,
		Sessions: sessions,
		Worker:   fw,
	}
	failed, err := svc.FailRun(ctx, cfg, run.ID, "compile error")
	if err != nil {
		t.Fatalf("FailRun: %v", err)
	}
	if failed.State != domain.StateFailed || failed.FailureReason == nil || *failed.FailureReason != "compile error" {
		t.Fatalf("unexpected failed run: %+v", failed)
	}

	fw.mu.Lock()
	cleanups := len(fw.cleanups)
	fw.mu.Unlock()
	if cleanups != 0 {
		t.Fatalf("FailRun should preserve worktree, got %d cleanups", cleanups)
	}

	gotTask, err := tasks.Get(ctx, rec.ID)
	if err != nil {
		t.Fatalf("tasks.Get: %v", err)
	}
	if gotTask.Status != task.StatusTodo {
		t.Fatalf("task status = %s, want todo", gotTask.Status)
	}
	if !slices.Contains(gotTask.Labels, "vibe:failed") {
		t.Fatalf("expected vibe:failed label, got %v", gotTask.Labels)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		n := len(finished)
		mu.Unlock()
		if n > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(finished) != 1 || finished[0].Outcome != domain.StateFailed || finished[0].Reason != "compile error" {
		t.Fatalf("unexpected finished events: %+v", finished)
	}
}

func TestResilience_QuotaPauseAndAutoResume(t *testing.T) {
	ctx := context.Background()
	_, client, bus, _, _, projID, nodeID := setupReconcilerEnv(t)

	now := time.Date(2026, 11, 18, 8, 0, 0, 0, time.UTC)
	svc := service.New(repo.New(client), service.Config{
		Bus: bus,
		Now: func() time.Time { return now },
	})

	run, err := svc.Create(ctx, domain.CreateInput{
		TaskID:    uuid.New(),
		ProjectID: projID,
		NodeID:    &nodeID,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Nil resetAt falls back to now + 15m.
	paused, err := svc.PauseOnQuota(ctx, run.ID, nil, "quota exhausted")
	if err != nil {
		t.Fatalf("PauseOnQuota: %v", err)
	}
	if paused.State != domain.StatePaused || paused.PausedUntil == nil {
		t.Fatalf("unexpected paused run: %+v", paused)
	}
	if !paused.PausedUntil.Equal(now.Add(15 * time.Minute)) {
		t.Fatalf("PausedUntil = %v, want %v", paused.PausedUntil, now.Add(15*time.Minute))
	}

	// Before reset time: stays paused.
	now = now.Add(10 * time.Minute)
	if err := svc.ResumePausedRuns(ctx); err != nil {
		t.Fatalf("ResumePausedRuns early: %v", err)
	}
	got, _ := svc.Get(ctx, run.ID)
	if got.State != domain.StatePaused {
		t.Fatalf("expected still paused, got %s", got.State)
	}

	// After reset time: resumes to queued.
	now = now.Add(6 * time.Minute)
	if err := svc.ResumePausedRuns(ctx); err != nil {
		t.Fatalf("ResumePausedRuns elapsed: %v", err)
	}
	got, _ = svc.Get(ctx, run.ID)
	if got.State != domain.StateQueued || got.PausedUntil != nil {
		t.Fatalf("expected resumed queued run with nil PausedUntil, got %+v", got)
	}
}

func TestResilience_CancelRecoverAndTimeout(t *testing.T) {
	ctx := context.Background()
	_, client, bus, tasks, _, projID, nodeID := setupReconcilerEnv(t)
	projects := project.New(client)
	fw := &fakeWorkerTransport{}

	now := time.Date(2026, 11, 18, 9, 0, 0, 0, time.UTC)
	sessions := session.New(client, session.Config{
		Bus: bus,
		Now: func() time.Time { return now },
	})
	svc := service.New(repo.New(client), service.Config{
		Bus: bus,
		Now: func() time.Time { return now },
	})
	cfg := service.StageConfig{
		AppHome:  "/tmp/opus-home",
		Tasks:    tasks,
		Projects: projects,
		Sessions: sessions,
		Worker:   fw,
	}

	// 1. Cancel running run.
	rec1 := importGitHubTask(t, tasks, projID, "601", task.StatusTodo, []string{"vibe"}, validDescription)
	r1, _ := svc.Create(ctx, domain.CreateInput{TaskID: uuid.MustParse(rec1.ID), ProjectID: projID, NodeID: &nodeID})
	_, _ = svc.Claim(ctx, cfg, r1.ID)
	r1, _ = svc.PrepareAndStartSession(ctx, cfg, r1.ID)

	cancelled, err := svc.CancelRun(ctx, cfg, r1.ID, "stop now")
	if err != nil {
		t.Fatalf("CancelRun: %v", err)
	}
	if cancelled.State != domain.StateCancelled {
		t.Fatalf("state = %s, want cancelled", cancelled.State)
	}
	fw.mu.Lock()
	if len(fw.cancelled) != 1 || len(fw.cleanups) != 1 {
		fw.mu.Unlock()
		t.Fatalf("expected 1 CancelRun and 1 CleanupWorktree, got %d/%d", len(fw.cancelled), len(fw.cleanups))
	}
	fw.mu.Unlock()

	// 2. Stage timeout.
	rec2 := importGitHubTask(t, tasks, projID, "602", task.StatusTodo, []string{"vibe"}, validDescription)
	r2, _ := svc.Create(ctx, domain.CreateInput{TaskID: uuid.MustParse(rec2.ID), ProjectID: projID, NodeID: &nodeID})
	_, _ = svc.Claim(ctx, cfg, r2.ID)
	now = now.Add(31 * time.Minute)
	if err := svc.EnforceStageTimeouts(ctx, cfg, 30*time.Minute); err != nil {
		t.Fatalf("EnforceStageTimeouts: %v", err)
	}
	got2, _ := svc.Get(ctx, r2.ID)
	if got2.State != domain.StateFailed {
		t.Fatalf("expected timed-out run to fail, got %s", got2.State)
	}

	// 3. Restart recovery fails active preparing/running runs.
	rec3 := importGitHubTask(t, tasks, projID, "603", task.StatusTodo, []string{"vibe"}, validDescription)
	r3, _ := svc.Create(ctx, domain.CreateInput{TaskID: uuid.MustParse(rec3.ID), ProjectID: projID, NodeID: &nodeID})
	_, _ = svc.Claim(ctx, cfg, r3.ID)
	if err := svc.RecoverActiveRuns(ctx, cfg); err != nil {
		t.Fatalf("RecoverActiveRuns: %v", err)
	}
	got3, _ := svc.Get(ctx, r3.ID)
	if got3.State != domain.StateFailed {
		t.Fatalf("expected recovered run to fail, got %s", got3.State)
	}

	// 4. Activity-based staleness: recent session_event keeps run alive past UpdatedAt cutoff.
	rec4 := importGitHubTask(t, tasks, projID, "604", task.StatusTodo, []string{"vibe"}, validDescription)
	sess4, err := sessions.Enqueue(ctx, session.CreateInput{
		TaskID:   uuid.MustParse(rec4.ID),
		NodeID:   nodeID,
		Provider: "agy",
	})
	if err != nil {
		t.Fatalf("Enqueue sess4: %v", err)
	}
	r4, _ := svc.Create(ctx, domain.CreateInput{TaskID: uuid.MustParse(rec4.ID), ProjectID: projID, NodeID: &nodeID})
	_, _ = svc.Claim(ctx, cfg, r4.ID)
	_, _ = svc.Transition(ctx, r4.ID, domain.StateRunning, domain.TransitionInput{SessionID: &sess4.ID})

	now = now.Add(25 * time.Minute)
	if _, err := sessions.RecordEvent(ctx, sess4.ID, "agent_message_chunk", []byte(`{"text":"working"}`)); err != nil {
		t.Fatalf("RecordEvent: %v", err)
	}
	// Advance past UpdatedAt timeout (30m from Claim), but within 30m of RecordEvent.
	now = now.Add(10 * time.Minute)
	if err := svc.EnforceStageTimeouts(ctx, cfg, 30*time.Minute); err != nil {
		t.Fatalf("EnforceStageTimeouts active: %v", err)
	}
	got4, _ := svc.Get(ctx, r4.ID)
	if got4.State != domain.StateRunning {
		t.Fatalf("expected run with recent session_event to remain running, got %s", got4.State)
	}

	// Now advance past 30m since RecordEvent -> must fail.
	now = now.Add(31 * time.Minute)
	if err := svc.EnforceStageTimeouts(ctx, cfg, 30*time.Minute); err != nil {
		t.Fatalf("EnforceStageTimeouts stale: %v", err)
	}
	got4After, _ := svc.Get(ctx, r4.ID)
	if got4After.State != domain.StateFailed {
		t.Fatalf("expected run with stale activity to fail, got %s", got4After.State)
	}
}
