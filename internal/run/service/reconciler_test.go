package service_test

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kilip/opus/ent"
	"github.com/kilip/opus/ent/node"
	"github.com/kilip/opus/internal/platform/eventbus"
	"github.com/kilip/opus/internal/project"
	"github.com/kilip/opus/internal/run/domain"
	"github.com/kilip/opus/internal/run/repo"
	"github.com/kilip/opus/internal/run/service"
	"github.com/kilip/opus/internal/session"
	"github.com/kilip/opus/internal/task"
)

const validDescription = `## Scope
Implement feature X.

## Acceptance criteria
- AC1: Works properly.

## Definition of Done
- Run ` + "`go test ./...`" + `
`

func setupReconcilerEnv(t *testing.T) (
	*service.Service,
	*ent.Client,
	*eventbus.Bus,
	*task.Task,
	*session.Session,
	uuid.UUID,
	uuid.UUID,
) {
	t.Helper()
	ctx := context.Background()
	svc, client, bus := setupService(t)

	proj, err := client.Project.Create().
		SetSlug("test-proj-" + uuid.NewString()[:8]).
		SetName("Test Project").
		SetRepoProvider("github").
		SetRepoRef("kilip/opus").
		SetTrackerProvider("github").
		SetTrackerRef("kilip/opus").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	n, err := client.Node.Create().
		SetName("alpha-node").
		SetRole(node.RoleWorker).
		SetStatus(node.StatusOnline).
		Save(ctx)
	if err != nil {
		t.Fatalf("create node: %v", err)
	}

	if _, err := client.ProjectNode.Create().
		SetProjectID(proj.ID).
		SetNodeID(n.ID).
		SetPath("/workspace/opus").
		Save(ctx); err != nil {
		t.Fatalf("create project_node: %v", err)
	}

	taskFeature := task.New(client, bus)
	sessionFeature := session.New(client, session.Config{Bus: bus})
	return svc, client, bus, taskFeature, sessionFeature, proj.ID, n.ID
}

func importGitHubTask(
	t *testing.T,
	tasks *task.Task,
	projectID uuid.UUID,
	extID string,
	status task.Status,
	labels []string,
	desc string,
) task.Record {
	t.Helper()
	rec, err := tasks.Import(context.Background(), task.ImportTaskRequest{
		ProjectID:   projectID.String(),
		Source:      task.SourceGitHub,
		ExternalID:  extID,
		Title:       "Issue #" + extID,
		Description: desc,
		Status:      status,
		Labels:      labels,
	})
	if err != nil {
		t.Fatalf("import task %s: %v", extID, err)
	}
	return rec
}

func TestReconciler_EligibilityAndBlockedBy(t *testing.T) {
	ctx := context.Background()
	svc, _, _, tasks, sessions, projID, nodeID := setupReconcilerEnv(t)

	// Task without vibe label -> skipped.
	importGitHubTask(t, tasks, projID, "101", task.StatusTodo, []string{"enhancement"}, validDescription)
	// Task with vibe:failed -> skipped.
	importGitHubTask(t, tasks, projID, "102", task.StatusTodo, []string{"vibe", "vibe:failed"}, validDescription)
	// Task blocked by #104 (not done yet) -> skipped.
	importGitHubTask(t, tasks, projID, "103", task.StatusTodo, []string{"vibe"}, validDescription+"\nBlocked by: #104\n")
	dep := importGitHubTask(t, tasks, projID, "104", task.StatusInProgress, nil, "dep")

	cfg := service.ReconcilerConfig{
		Tasks:    tasks,
		Sessions: sessions,
	}
	if err := svc.ReconcileOnce(ctx, cfg); err != nil {
		t.Fatalf("ReconcileOnce 1: %v", err)
	}
	runs, err := svc.List(ctx, domain.ListFilter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(runs) != 0 {
		t.Fatalf("expected 0 runs while blocked/ineligible, got %d", len(runs))
	}

	// Mark #104 done -> #103 becomes eligible!
	done := task.StatusDone
	if _, err := tasks.Update(ctx, dep.ID, task.UpdateTaskRequest{Status: &done}); err != nil {
		t.Fatalf("Update dep done: %v", err)
	}

	if err := svc.ReconcileOnce(ctx, cfg); err != nil {
		t.Fatalf("ReconcileOnce 2: %v", err)
	}
	runs, err = svc.List(ctx, domain.ListFilter{})
	if err != nil {
		t.Fatalf("List 2: %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("expected 1 run after unblock, got %d", len(runs))
	}
	if runs[0].Mode != domain.ModeVibe || runs[0].State != domain.StateQueued || runs[0].NodeID == nil || *runs[0].NodeID != nodeID {
		t.Fatalf("unexpected run created: %+v", runs[0])
	}

	// Running ReconcileOnce again must not create a duplicate active run.
	if err := svc.ReconcileOnce(ctx, cfg); err != nil {
		t.Fatalf("ReconcileOnce 3: %v", err)
	}
	runs, _ = svc.List(ctx, domain.ListFilter{})
	if len(runs) != 1 {
		t.Fatalf("expected still 1 run, got %d", len(runs))
	}

	// If a session is running, session slot is full -> new eligible task is skipped.
	if _, err := sessions.Enqueue(ctx, session.CreateInput{
		TaskID:   uuid.MustParse(runs[0].TaskID.String()),
		NodeID:   nodeID,
		Provider: "agy",
	}); err != nil {
		t.Fatalf("Enqueue session: %v", err)
	}
	importGitHubTask(t, tasks, projID, "105", task.StatusTodo, []string{"vibe"}, validDescription)
	if err := svc.ReconcileOnce(ctx, cfg); err != nil {
		t.Fatalf("ReconcileOnce 4: %v", err)
	}
	runs, _ = svc.List(ctx, domain.ListFilter{})
	if len(runs) != 1 {
		t.Fatalf("expected still 1 run when session slot busy, got %d", len(runs))
	}
}

func TestReconciler_StructureLintNeedsInfo(t *testing.T) {
	ctx := context.Background()
	svc, _, bus, tasks, sessions, projID, _ := setupReconcilerEnv(t)

	var (
		mu     sync.Mutex
		events []eventbus.Event
	)
	_ = bus.Subscribe("test-lint-finished", func(_ context.Context, e eventbus.Event) error {
		mu.Lock()
		events = append(events, e)
		mu.Unlock()
		return nil
	}, domain.EventFinished)

	badTask := importGitHubTask(t, tasks, projID, "201", task.StatusTodo, []string{"vibe"}, "Missing all sections")

	if err := svc.ReconcileOnce(ctx, service.ReconcilerConfig{
		Tasks:    tasks,
		Sessions: sessions,
	}); err != nil {
		t.Fatalf("ReconcileOnce: %v", err)
	}

	runs, err := svc.List(ctx, domain.ListFilter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(runs) != 1 || runs[0].State != domain.StateNeedsInfo {
		t.Fatalf("expected 1 needs-info run, got %+v", runs)
	}

	updatedTask, err := tasks.Get(ctx, badTask.ID)
	if err != nil {
		t.Fatalf("tasks.Get: %v", err)
	}
	hasNeedsInfoLabel := false
	for _, l := range updatedTask.Labels {
		if l == "vibe:needs-info" {
			hasNeedsInfoLabel = true
		}
	}
	if !hasNeedsInfoLabel {
		t.Fatalf("expected vibe:needs-info label on task, got %v", updatedTask.Labels)
	}

	activities, err := tasks.ListActivity(ctx, badTask.ID)
	if err != nil {
		t.Fatalf("ListActivity: %v", err)
	}
	foundVibeActivity := false
	for _, a := range activities {
		if a.Author == "vibe" && a.Kind == task.ActivityKindComment {
			foundVibeActivity = true
		}
	}
	if !foundVibeActivity {
		t.Fatalf("expected activity authored by vibe, got %+v", activities)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		n := len(events)
		mu.Unlock()
		if n > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(events) != 1 {
		t.Fatalf("expected 1 run.finished event, got %d", len(events))
	}
	var payload domain.RunFinishedPayload
	if err := json.Unmarshal(events[0].Payload, &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if payload.Outcome != domain.StateNeedsInfo {
		t.Fatalf("expected outcome needs-info, got %s", payload.Outcome)
	}
}

func TestReconciler_DeterministicNodeSelection(t *testing.T) {
	ctx := context.Background()
	svc, client, _, tasks, sessions, projID, firstNodeID := setupReconcilerEnv(t)

	// Add an unmapped-path online node with lexicographically earlier name ("00-unmapped").
	unmappedNode, err := client.Node.Create().
		SetName("00-unmapped").
		SetRole(node.RoleWorker).
		SetStatus(node.StatusOnline).
		Save(ctx)
	if err != nil {
		t.Fatalf("create unmapped node: %v", err)
	}
	if _, err := client.ProjectNode.Create().
		SetProjectID(projID).
		SetNodeID(unmappedNode.ID).
		Save(ctx); err != nil {
		t.Fatalf("create unmapped project_node: %v", err)
	}

	// Add a mapped-path online node with earlier name ("01-mapped") -> should win over "alpha-node" and "00-unmapped".
	preferredNode, err := client.Node.Create().
		SetName("01-mapped").
		SetRole(node.RoleWorker).
		SetStatus(node.StatusOnline).
		Save(ctx)
	if err != nil {
		t.Fatalf("create preferred node: %v", err)
	}
	if _, err := client.ProjectNode.Create().
		SetProjectID(projID).
		SetNodeID(preferredNode.ID).
		SetPath("/workspace/preferred").
		Save(ctx); err != nil {
		t.Fatalf("create preferred project_node: %v", err)
	}

	_ = firstNodeID
	importGitHubTask(t, tasks, projID, "301", task.StatusTodo, []string{"vibe"}, validDescription)

	if err := svc.ReconcileOnce(ctx, service.ReconcilerConfig{
		Tasks:    tasks,
		Sessions: sessions,
		IsAgyReady: func(_ context.Context, id uuid.UUID) bool {
			return id != uuid.Nil
		},
	}); err != nil {
		t.Fatalf("ReconcileOnce: %v", err)
	}

	runs, err := svc.List(ctx, domain.ListFilter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(runs) != 1 || runs[0].NodeID == nil || *runs[0].NodeID != preferredNode.ID {
		t.Fatalf("expected preferredNode %s, got %+v", preferredNode.ID, runs)
	}
}

func TestReconciler_PeriodicScanRecoversDroppedBusEvent(t *testing.T) {
	svc, _, _, tasks, sessions, projID, _ := setupReconcilerEnv(t)

	runCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- svc.Run(runCtx, service.ReconcilerConfig{
			Tasks:        tasks,
			Sessions:     sessions,
			PollInterval: 20 * time.Millisecond,
		})
	}()

	// Import task directly without publishing any eventbus event (simulating a dropped bus event).
	importGitHubTask(t, tasks, projID, "401", task.StatusTodo, []string{"vibe"}, validDescription)

	deadline := time.Now().Add(2 * time.Second)
	var runs []domain.Run
	for time.Now().Before(deadline) {
		var err error
		runs, err = svc.List(context.Background(), domain.ListFilter{})
		if err == nil && len(runs) == 1 {
			break
		}
		time.Sleep(15 * time.Millisecond)
	}
	if len(runs) != 1 {
		t.Fatalf("expected periodic scan to pick up task after dropped bus event, got %d runs", len(runs))
	}

	cancel()
	if err := <-done; err != nil {
		t.Fatalf("Run returned error on cancel: %v", err)
	}
}

func TestReconciler_SessionLifecycleAndNodeGrace(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	_, client, bus, tasks, sessions, projID, nodeID := setupReconcilerEnv(t)
	svc := service.New(repo.New(client), service.Config{
		Bus: bus,
		Now: func() time.Time { return now },
	})
	fw := &fakeWorkerTransport{}
	stageCfg := service.StageConfig{
		AppHome:  t.TempDir(),
		Tasks:    tasks,
		Projects: project.New(client),
		Sessions: sessions,
		Worker:   fw,
	}

	// 1. session.failed propagates to running Run, fails it, and cleans up worktree.
	rec1 := importGitHubTask(t, tasks, projID, "701", task.StatusTodo, []string{"vibe"}, validDescription)
	r1, _ := svc.Create(ctx, domain.CreateInput{TaskID: uuid.MustParse(rec1.ID), ProjectID: projID, NodeID: &nodeID})
	_, _ = svc.Claim(ctx, stageCfg, r1.ID)
	sess1, _ := sessions.Enqueue(ctx, session.CreateInput{TaskID: uuid.MustParse(rec1.ID), NodeID: nodeID, Provider: "agy"})
	_, _ = svc.Transition(ctx, r1.ID, domain.StateRunning, domain.TransitionInput{SessionID: &sess1.ID})

	_, _, _ = sessions.MarkFailed(ctx, sess1.ID, "agy crashed")
	payload1, _ := json.Marshal(session.SessionLifecyclePayload{
		SessionID: sess1.ID,
		TaskID:    r1.TaskID,
		NodeID:    nodeID,
		Status:    session.StatusFailed,
	})
	if err := svc.HandleSessionLifecycleEvent(ctx, stageCfg, eventbus.Event{
		Type:    session.EventFailed,
		Payload: payload1,
	}); err != nil {
		t.Fatalf("HandleSessionLifecycleEvent failed: %v", err)
	}
	got1, _ := svc.Get(ctx, r1.ID)
	if got1.State != domain.StateFailed || got1.FailureReason == nil || *got1.FailureReason != "agy crashed" {
		t.Fatalf("expected failed run with reason 'agy crashed', got %+v", got1)
	}

	// Idempotent replay on already terminal run must succeed without error.
	if err := svc.HandleSessionLifecycleEvent(ctx, stageCfg, eventbus.Event{
		Type:    session.EventFailed,
		Payload: payload1,
	}); err != nil {
		t.Fatalf("HandleSessionLifecycleEvent idempotent replay: %v", err)
	}

	// 2. session.cancelled propagates to running Run.
	rec2 := importGitHubTask(t, tasks, projID, "702", task.StatusTodo, []string{"vibe"}, validDescription)
	r2, _ := svc.Create(ctx, domain.CreateInput{TaskID: uuid.MustParse(rec2.ID), ProjectID: projID, NodeID: &nodeID})
	_, _ = svc.Claim(ctx, stageCfg, r2.ID)
	sess2, _ := sessions.Enqueue(ctx, session.CreateInput{TaskID: uuid.MustParse(rec2.ID), NodeID: nodeID, Provider: "agy"})
	_, _ = svc.Transition(ctx, r2.ID, domain.StateRunning, domain.TransitionInput{SessionID: &sess2.ID})
	_, _, _ = sessions.Cancel(ctx, sess2.ID, "cancelled externally")
	payload2, _ := json.Marshal(session.SessionLifecyclePayload{
		SessionID: sess2.ID,
		TaskID:    r2.TaskID,
		NodeID:    nodeID,
		Status:    session.StatusCancelled,
	})
	if err := svc.HandleSessionLifecycleEvent(ctx, stageCfg, eventbus.Event{
		Type:    session.EventCancelled,
		Payload: payload2,
	}); err != nil {
		t.Fatalf("HandleSessionLifecycleEvent cancelled: %v", err)
	}
	got2, _ := svc.Get(ctx, r2.ID)
	if got2.State != domain.StateCancelled {
		t.Fatalf("expected cancelled run, got %s", got2.State)
	}

	// 3. Node disconnect grace: offline > NodeGrace fails run and unblocks capacity gate.
	rec3 := importGitHubTask(t, tasks, projID, "703", task.StatusTodo, []string{"vibe"}, validDescription)
	r3, _ := svc.Create(ctx, domain.CreateInput{TaskID: uuid.MustParse(rec3.ID), ProjectID: projID, NodeID: &nodeID})
	_, _ = svc.Claim(ctx, stageCfg, r3.ID)
	sess3, _ := sessions.Enqueue(ctx, session.CreateInput{TaskID: uuid.MustParse(rec3.ID), NodeID: nodeID, Provider: "agy"})
	_, _ = svc.Transition(ctx, r3.ID, domain.StateRunning, domain.TransitionInput{SessionID: &sess3.ID})

	_ = bus
	svc.MarkNodeOffline(nodeID.String())
	now = now.Add(3 * time.Minute)
	cfg := service.ReconcilerConfig{
		Tasks:           tasks,
		Sessions:        sessions,
		Stages:          stageCfg,
		NodeGrace:       2 * time.Minute,
		IsNodeConnected: func(string) bool { return false },
	}
	if err := svc.ReconcileOnce(ctx, cfg); err != nil {
		t.Fatalf("ReconcileOnce disconnect grace: %v", err)
	}
	got3, _ := svc.Get(ctx, r3.ID)
	if got3.State != domain.StateFailed || got3.FailureReason == nil || *got3.FailureReason != "worker node offline" {
		t.Fatalf("expected run failed with 'worker node offline', got %+v", got3)
	}
}
