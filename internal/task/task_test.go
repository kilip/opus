package task_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"testing"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/kilip/opus/ent"
	"github.com/kilip/opus/ent/enttest"
	"github.com/kilip/opus/internal/platform/eventbus"
	"github.com/kilip/opus/internal/task"
	_ "modernc.org/sqlite"
)

func setupFacade(t *testing.T) (*task.Task, *ent.Client, *eventbus.Bus, string) {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+t.Name()+"?mode=memory&cache=shared&_pragma=foreign_keys(ON)&_pragma=busy_timeout(10000)")
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	db.SetMaxOpenConns(1)
	drv := entsql.OpenDB(dialect.SQLite, db)
	client := enttest.NewClient(t, enttest.WithOptions(ent.Driver(drv)))
	t.Cleanup(func() {
		_ = client.Close()
	})

	p, err := client.Project.Create().
		SetSlug(fmt.Sprintf("facade-proj-%s", uuid.New().String()[:8])).
		SetName("Facade Project").
		SetRepoProvider("github").
		SetRepoRef("org/repo").
		SetTrackerProvider("github").
		SetTrackerRef("org/repo").
		Save(context.Background())
	if err != nil {
		t.Fatalf("create project failed: %v", err)
	}

	bus := eventbus.New()
	return task.New(client, bus), client, bus, p.ID.String()
}

// AC1, AC2, AC3, AC4, AC5, AC7: Sibling features import only internal/task and exercise
// Import, FindByExternal, AddLabel/RemoveLabel/SetLabels, and AddActivity.
func TestTaskFacade_ExternalPackageAccess(t *testing.T) {
	facade, client, bus, projectID := setupFacade(t)
	ctx := context.Background()

	var (
		mu     sync.Mutex
		events []eventbus.Event
	)
	if err := bus.Subscribe("sibling-subscriber", func(ctx context.Context, e eventbus.Event) error {
		mu.Lock()
		events = append(events, e)
		mu.Unlock()
		return nil
	}); err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	// AC3: FindByExternal returns ErrTaskNotFound when missing
	if _, err := facade.FindByExternal(ctx, task.SourceGitHub, "issue-103"); !errors.Is(err, task.ErrTaskNotFound) {
		t.Fatalf("expected ErrTaskNotFound, got %v", err)
	}

	// AC2: Import creates task with all fields, defaulting Key to OP-1 when omitted
	req := task.ImportTaskRequest{
		ProjectID:    projectID,
		Title:        "Expose task service",
		Description:  "Allow sibling features to import tasks",
		Source:       task.SourceGitHub,
		ExternalID:   "issue-103",
		ExternalURL:  "https://github.com/org/repo/issues/103",
		SourceStatus: "in_progress",
		Labels:       []string{"api", "sync"},
		Type:         task.TypeFeat,
		Priority:     task.PriorityP1,
		Status:       task.StatusInProgress,
	}
	created, err := facade.Import(ctx, req)
	if err != nil {
		t.Fatalf("Import create failed: %v", err)
	}
	if created.Key != "OP-1" {
		t.Fatalf("expected default key OP-1, got %q", created.Key)
	}
	if created.Status != task.StatusInProgress {
		t.Fatalf("expected status %q, got %q", task.StatusInProgress, created.Status)
	}
	if created.SourceStatus != "in_progress" {
		t.Fatalf("expected sourceStatus 'in_progress', got %q", created.SourceStatus)
	}

	// AC3: FindByExternal locates imported task
	found, err := facade.FindByExternal(ctx, task.SourceGitHub, "issue-103")
	if err != nil {
		t.Fatalf("FindByExternal failed: %v", err)
	}
	if found.ID != created.ID {
		t.Fatalf("FindByExternal ID = %s, want %s", found.ID, created.ID)
	}

	// AC2 & AC5: Identical Import performs no write and emits no event
	again, err := facade.Import(ctx, req)
	if err != nil {
		t.Fatalf("Import identical failed: %v", err)
	}
	if !again.UpdatedAt.Equal(created.UpdatedAt) {
		t.Fatalf("expected UpdatedAt unchanged on identical Import: %v vs %v", again.UpdatedAt, created.UpdatedAt)
	}

	// AC7: Transition through blocked status via Import emits task.status_changed with projectId
	req.Status = task.StatusBlocked
	req.SourceStatus = "blocked"
	req.Key = "GH-103"
	blocked, err := facade.Import(ctx, req)
	if err != nil {
		t.Fatalf("Import transition to blocked failed: %v", err)
	}
	if blocked.Status != task.StatusBlocked || blocked.Key != "GH-103" {
		t.Fatalf("unexpected blocked task: %+v", blocked)
	}

	// AC1 & AC5: Label helpers mutate labels without publishing events
	if err := facade.AddLabel(ctx, blocked.ID, "needs-review"); err != nil {
		t.Fatalf("AddLabel failed: %v", err)
	}
	if err := facade.RemoveLabel(ctx, blocked.ID, "sync"); err != nil {
		t.Fatalf("RemoveLabel failed: %v", err)
	}
	afterLabelOps, err := facade.Get(ctx, blocked.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if len(afterLabelOps.Labels) != 2 || afterLabelOps.Labels[0] != "api" || afterLabelOps.Labels[1] != "needs-review" {
		t.Fatalf("unexpected labels after Add/Remove: %v", afterLabelOps.Labels)
	}
	if err := facade.SetLabels(ctx, blocked.ID, []string{"blocked", "p1"}); err != nil {
		t.Fatalf("SetLabels failed: %v", err)
	}
	afterSet, err := facade.Get(ctx, blocked.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if len(afterSet.Labels) != 2 || afterSet.Labels[0] != "blocked" || afterSet.Labels[1] != "p1" {
		t.Fatalf("unexpected labels after SetLabels: %v", afterSet.Labels)
	}

	// AC4: Internal AddActivity persists optional author
	act, err := facade.AddActivity(ctx, blocked.ID, task.ActivityKindComment, "github-sync", "Synced blocked state from tracker")
	if err != nil {
		t.Fatalf("AddActivity failed: %v", err)
	}
	if act.Author != "github-sync" {
		t.Fatalf("expected author 'github-sync', got %q", act.Author)
	}

	// Verify DB activity count (1 initial + 1 blocked transition + 1 authored comment = 3)
	actCount, err := client.Activity.Query().Count(ctx)
	if err != nil {
		t.Fatalf("count activities: %v", err)
	}
	if actCount != 3 {
		t.Fatalf("expected 3 activities in DB (no extra from no-op import), got %d", actCount)
	}

	if err := bus.Close(ctx); err != nil {
		t.Fatalf("close bus: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(events) != 2 {
		t.Fatalf("expected exactly 2 events (task.received + task.status_changed), got %d", len(events))
	}
	if events[0].Type != task.EventReceived {
		t.Fatalf("event[0] = %s, want %s", events[0].Type, task.EventReceived)
	}
	recv, err := eventbus.Decode[task.TaskReceivedPayload](events[0])
	if err != nil {
		t.Fatalf("decode received: %v", err)
	}
	if recv.ProjectID != projectID || recv.Status != task.StatusInProgress {
		t.Fatalf("unexpected received payload: %+v", recv)
	}

	if events[1].Type != task.EventStatusChanged {
		t.Fatalf("event[1] = %s, want %s", events[1].Type, task.EventStatusChanged)
	}
	chg, err := eventbus.Decode[task.TaskStatusChangedPayload](events[1])
	if err != nil {
		t.Fatalf("decode status_changed: %v", err)
	}
	if chg.ProjectID != projectID || chg.From != task.StatusInProgress || chg.To != task.StatusBlocked {
		t.Fatalf("unexpected status_changed payload: %+v", chg)
	}
}
