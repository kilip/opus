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
	"github.com/kilip/opus/internal/run/domain"
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
