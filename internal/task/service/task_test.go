package service_test

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
	"github.com/kilip/opus/internal/task/domain"
	"github.com/kilip/opus/internal/task/repo"
	"github.com/kilip/opus/internal/task/service"
	_ "modernc.org/sqlite"
)

func setupTestClient(t *testing.T) *ent.Client {
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
	return client
}

func createTestProject(t *testing.T, client *ent.Client) string {
	t.Helper()
	p, err := client.Project.Create().
		SetSlug(fmt.Sprintf("svc-test-proj-%s", uuid.New().String()[:8])).
		SetName("Test Project").
		SetRepoProvider("github").
		SetRepoRef("org/repo").
		SetTrackerProvider("github").
		SetTrackerRef("org/repo").
		Save(context.Background())
	if err != nil {
		t.Fatalf("create test project failed: %v", err)
	}
	return p.ID.String()
}

func newService(t *testing.T, bus ...*eventbus.Bus) (*service.Task, context.Context, string) {
	t.Helper()
	var b *eventbus.Bus
	if len(bus) > 0 {
		b = bus[0]
	} else {
		b = eventbus.New()
	}
	client := setupTestClient(t)
	projectID := createTestProject(t, client)
	return service.New(repo.New(client), b), context.Background(), projectID
}

func TestCreateValidation(t *testing.T) {
	svc, ctx, projectID := newService(t)

	// Missing project ID
	_, err := svc.Create(ctx, domain.CreateTaskRequest{Title: "Task"})
	if !errors.Is(err, domain.ErrProjectIDRequired) {
		t.Fatalf("expected ErrProjectIDRequired, got %v", err)
	}

	// Malformed project ID
	_, err = svc.Create(ctx, domain.CreateTaskRequest{ProjectID: "invalid-uuid", Title: "Task"})
	if !errors.Is(err, domain.ErrInvalidProjectID) {
		t.Fatalf("expected ErrInvalidProjectID, got %v", err)
	}

	// Unknown project ID
	_, err = svc.Create(ctx, domain.CreateTaskRequest{ProjectID: uuid.New().String(), Title: "Task"})
	if !errors.Is(err, domain.ErrProjectNotFound) {
		t.Fatalf("expected ErrProjectNotFound, got %v", err)
	}

	// Blank title
	_, err = svc.Create(ctx, domain.CreateTaskRequest{ProjectID: projectID, Title: "  "})
	if !errors.Is(err, domain.ErrTaskTitleEmpty) {
		t.Fatalf("expected ErrTaskTitleEmpty, got %v", err)
	}
}

func TestUpdateRejectsBlankTitle(t *testing.T) {
	svc, ctx, _ := newService(t)
	blank := ""
	_, err := svc.Update(ctx, "t1", domain.UpdateTaskRequest{Title: &blank})
	if !errors.Is(err, domain.ErrTaskTitleEmpty) {
		t.Fatalf("expected ErrTaskTitleEmpty, got %v", err)
	}
}

func TestCreateActivityValidatesBeforeLookup(t *testing.T) {
	svc, ctx, _ := newService(t)

	_, err := svc.CreateActivity(ctx, "t1", domain.CreateActivityRequest{Body: " "})
	if !errors.Is(err, domain.ErrActivityBodyEmpty) {
		t.Fatalf("expected ErrActivityBodyEmpty, got %v", err)
	}

	_, err = svc.CreateActivity(ctx, "missing", domain.CreateActivityRequest{Body: "hi"})
	if !errors.Is(err, domain.ErrTaskNotFound) {
		t.Fatalf("expected ErrTaskNotFound, got %v", err)
	}
}

func TestTaskEvents_EmittedAndOrder(t *testing.T) {
	bus := eventbus.New()

	var (
		mu     sync.Mutex
		events []eventbus.Event
	)
	err := bus.Subscribe("test-subscriber", func(ctx context.Context, e eventbus.Event) error {
		mu.Lock()
		events = append(events, e)
		mu.Unlock()
		return nil
	})
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	svc, ctx, projectID := newService(t, bus)

	created, err := svc.Create(ctx, domain.CreateTaskRequest{
		ProjectID: projectID,
		Title:     "Emitted and Order Task",
	})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	nextStatus := domain.StatusInProgress
	updated, err := svc.Update(ctx, created.ID, domain.UpdateTaskRequest{
		Status: &nextStatus,
	})
	if err != nil {
		t.Fatalf("update task: %v", err)
	}
	if updated.Status != nextStatus {
		t.Fatalf("expected status %s, got %s", nextStatus, updated.Status)
	}

	if err := bus.Close(context.Background()); err != nil {
		t.Fatalf("close bus: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}

	// 1. task.received
	e0 := events[0]
	if e0.Type != domain.EventReceived {
		t.Fatalf("expected event[0] type %s, got %s", domain.EventReceived, e0.Type)
	}
	if e0.Subject != created.ID {
		t.Fatalf("expected event[0] subject %s, got %s", created.ID, e0.Subject)
	}
	p0, err := eventbus.Decode[domain.TaskReceivedPayload](e0)
	if err != nil {
		t.Fatalf("decode payload 0: %v", err)
	}
	if p0.ID != created.ID || p0.Key != created.Key || p0.ProjectID != created.ProjectID || p0.Source != created.Source || p0.Status != created.Status {
		t.Fatalf("unexpected task.received payload: %+v", p0)
	}

	// 2. task.status_changed
	e1 := events[1]
	if e1.Type != domain.EventStatusChanged {
		t.Fatalf("expected event[1] type %s, got %s", domain.EventStatusChanged, e1.Type)
	}
	if e1.Subject != created.ID {
		t.Fatalf("expected event[1] subject %s, got %s", created.ID, e1.Subject)
	}
	p1, err := eventbus.Decode[domain.TaskStatusChangedPayload](e1)
	if err != nil {
		t.Fatalf("decode payload 1: %v", err)
	}
	if p1.ID != created.ID || p1.ProjectID != created.ProjectID || p1.From != domain.StatusTodo || p1.To != nextStatus {
		t.Fatalf("unexpected task.status_changed payload: %+v", p1)
	}
}

func TestTaskEvents_NoEventOnSameStatusUpdate(t *testing.T) {
	bus := eventbus.New()

	var (
		mu     sync.Mutex
		events []eventbus.Event
	)
	err := bus.Subscribe("test-subscriber", func(ctx context.Context, e eventbus.Event) error {
		mu.Lock()
		events = append(events, e)
		mu.Unlock()
		return nil
	})
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	svc, ctx, projectID := newService(t, bus)

	created, err := svc.Create(ctx, domain.CreateTaskRequest{
		ProjectID: projectID,
		Title:     "Same Status Task",
	})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	// Update with same status
	sameStatus := domain.StatusTodo
	_, err = svc.Update(ctx, created.ID, domain.UpdateTaskRequest{
		Status: &sameStatus,
	})
	if err != nil {
		t.Fatalf("update same status: %v", err)
	}

	// Update title only (status nil)
	newTitle := "Updated Title Only"
	_, err = svc.Update(ctx, created.ID, domain.UpdateTaskRequest{
		Title: &newTitle,
	})
	if err != nil {
		t.Fatalf("update title only: %v", err)
	}

	if err := bus.Close(context.Background()); err != nil {
		t.Fatalf("close bus: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	// Only task.received from Create should be emitted
	if len(events) != 1 {
		t.Fatalf("expected only 1 event (task.received), got %d events: %+v", len(events), events)
	}
	if events[0].Type != domain.EventReceived {
		t.Fatalf("expected event type %s, got %s", domain.EventReceived, events[0].Type)
	}
}

func TestTaskEvents_NoEventOnFailedWrite(t *testing.T) {
	bus := eventbus.New()

	var (
		mu     sync.Mutex
		events []eventbus.Event
	)
	err := bus.Subscribe("test-subscriber", func(ctx context.Context, e eventbus.Event) error {
		mu.Lock()
		events = append(events, e)
		mu.Unlock()
		return nil
	})
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	svc, ctx, projectID := newService(t, bus)

	// Create with empty title fails
	_, err = svc.Create(ctx, domain.CreateTaskRequest{ProjectID: projectID, Title: "   "})
	if err == nil {
		t.Fatal("expected error on empty title create")
	}

	// Update non-existent task fails
	nextStatus := domain.StatusDone
	_, err = svc.Update(ctx, "nonexistent-task", domain.UpdateTaskRequest{Status: &nextStatus})
	if err == nil {
		t.Fatal("expected error on nonexistent update")
	}

	// Update with invalid title fails
	blank := " "
	_, err = svc.Update(ctx, "t1", domain.UpdateTaskRequest{Title: &blank})
	if err == nil {
		t.Fatal("expected error on blank title update")
	}

	if err := bus.Close(context.Background()); err != nil {
		t.Fatalf("close bus: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	if len(events) != 0 {
		t.Fatalf("expected 0 events on failed writes, got %d", len(events))
	}
}

func TestTaskEvents_CreatedAsDone(t *testing.T) {
	bus := eventbus.New()

	var (
		mu     sync.Mutex
		events []eventbus.Event
	)
	err := bus.Subscribe("test-subscriber", func(ctx context.Context, e eventbus.Event) error {
		mu.Lock()
		events = append(events, e)
		mu.Unlock()
		return nil
	})
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	svc, ctx, projectID := newService(t, bus)

	doneStatus := domain.StatusDone
	created, err := svc.Create(ctx, domain.CreateTaskRequest{
		ProjectID: projectID,
		Title:     "Task Created As Done",
		Status:    &doneStatus,
	})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	if err := bus.Close(context.Background()); err != nil {
		t.Fatalf("close bus: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	if len(events) != 1 {
		t.Fatalf("expected exactly 1 event, got %d", len(events))
	}
	if events[0].Type != domain.EventReceived {
		t.Fatalf("expected event type %s, got %s", domain.EventReceived, events[0].Type)
	}

	p, err := eventbus.Decode[domain.TaskReceivedPayload](events[0])
	if err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if p.ID != created.ID || p.ProjectID != created.ProjectID || p.Status != domain.StatusDone {
		t.Fatalf("expected status %s, got %s (payload: %+v)", domain.StatusDone, p.Status, p)
	}
}

func TestTaskEvents_PublishFailureLoggedAndDoesNotFail(t *testing.T) {
	bus := eventbus.New()
	// Pre-close the bus so bus.Publish returns eventbus.ErrClosed
	if err := bus.Close(context.Background()); err != nil {
		t.Fatalf("close bus: %v", err)
	}

	svc, ctx, projectID := newService(t, bus)

	created, err := svc.Create(ctx, domain.CreateTaskRequest{
		ProjectID: projectID,
		Title:     "Task Despite Closed Bus",
	})
	if err != nil {
		t.Fatalf("expected Create to succeed even with closed bus, got: %v", err)
	}
	if created.Title != "Task Despite Closed Bus" {
		t.Fatalf("unexpected title: %s", created.Title)
	}

	nextStatus := domain.StatusReview
	updated, err := svc.Update(ctx, created.ID, domain.UpdateTaskRequest{
		Status: &nextStatus,
	})
	if err != nil {
		t.Fatalf("expected Update to succeed even with closed bus, got: %v", err)
	}
	if updated.Status != nextStatus {
		t.Fatalf("expected status %s, got %s", nextStatus, updated.Status)
	}
}

func TestTaskEvents_ConcurrentUpdates_Race(t *testing.T) {
	bus := eventbus.New()

	var (
		mu     sync.Mutex
		events []domain.TaskStatusChangedPayload
	)
	err := bus.Subscribe("test-subscriber", func(ctx context.Context, e eventbus.Event) error {
		if e.Type != domain.EventStatusChanged {
			return nil
		}
		p, err := eventbus.Decode[domain.TaskStatusChangedPayload](e)
		if err != nil {
			return err
		}
		mu.Lock()
		events = append(events, p)
		mu.Unlock()
		return nil
	}, domain.EventStatusChanged)
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	svc, ctx, projectID := newService(t, bus)

	initialStatus := domain.StatusBacklog
	created, err := svc.Create(ctx, domain.CreateTaskRequest{
		ProjectID: projectID,
		Title:     "Concurrent Task",
		Status:    &initialStatus,
	})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	// Run concurrent updates on the created task
	stages := []domain.TaskStatus{
		domain.StatusTodo,
		domain.StatusInProgress,
		domain.StatusReview,
		domain.StatusBlocked,
		domain.StatusDone,
	}

	const goroutines = 20
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			targetStatus := stages[idx%len(stages)]
			_, _ = svc.Update(ctx, created.ID, domain.UpdateTaskRequest{
				Status: &targetStatus,
				Title:  domainPtr(fmt.Sprintf("Update from worker %d", idx)),
			})
		}(i)
	}

	wg.Wait()

	if err := bus.Close(context.Background()); err != nil {
		t.Fatalf("close bus: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	// Every status_changed event must chain: event[i].From == event[i-1].To
	// (where event[0].From is initialStatus)
	if len(events) == 0 {
		t.Fatal("expected at least one status_changed event")
	}

	prevTo := initialStatus
	for i, ev := range events {
		if ev.ID != created.ID {
			t.Fatalf("event %d ID mismatch: expected %s, got %s", i, created.ID, ev.ID)
		}
		if ev.ProjectID != created.ProjectID {
			t.Fatalf("event %d ProjectID mismatch: expected %s, got %s", i, created.ProjectID, ev.ProjectID)
		}
		if ev.From != prevTo {
			t.Fatalf("event %d sequence break: expected from=%s, got from=%s (to=%s)", i, prevTo, ev.From, ev.To)
		}
		prevTo = ev.To
	}
}

func domainPtr[T any](v T) *T {
	return &v
}

func TestImport_ValidationEventsAndLabels(t *testing.T) {
	bus := eventbus.New()
	var (
		mu     sync.Mutex
		events []eventbus.Event
	)
	if err := bus.Subscribe("test-import", func(ctx context.Context, e eventbus.Event) error {
		mu.Lock()
		events = append(events, e)
		mu.Unlock()
		return nil
	}); err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	svc, ctx, projectID := newService(t, bus)

	// Invalid status on Create and Update
	badStatus := domain.TaskStatus("unknown")
	if _, err := svc.Create(ctx, domain.CreateTaskRequest{
		ProjectID: projectID,
		Title:     "Bad status",
		Status:    &badStatus,
	}); !errors.Is(err, domain.ErrInvalidStatus) {
		t.Fatalf("Create bad status = %v, want ErrInvalidStatus", err)
	}
	if _, err := svc.Update(ctx, "missing", domain.UpdateTaskRequest{
		Status: &badStatus,
	}); !errors.Is(err, domain.ErrInvalidStatus) {
		t.Fatalf("Update bad status = %v, want ErrInvalidStatus", err)
	}

	// Import validation
	if _, err := svc.Import(ctx, domain.ImportTaskRequest{
		ProjectID:  projectID,
		ExternalID: "1",
		Title:      "No source",
	}); !errors.Is(err, domain.ErrSourceRequired) {
		t.Fatalf("Import no source = %v, want ErrSourceRequired", err)
	}
	if _, err := svc.Import(ctx, domain.ImportTaskRequest{
		ProjectID: projectID,
		Source:    domain.TaskSourceGitHub,
		Title:     "No ext id",
	}); !errors.Is(err, domain.ErrExternalIDRequired) {
		t.Fatalf("Import no ext id = %v, want ErrExternalIDRequired", err)
	}
	if _, err := svc.Import(ctx, domain.ImportTaskRequest{
		ProjectID:  projectID,
		Source:     domain.TaskSourceGitHub,
		ExternalID: "1",
		Title:      "Bad status",
		Status:     badStatus,
	}); !errors.Is(err, domain.ErrInvalidStatus) {
		t.Fatalf("Import bad status = %v, want ErrInvalidStatus", err)
	}

	// 1. Create via Import -> emits task.received
	req := domain.ImportTaskRequest{
		ProjectID:    projectID,
		Source:       domain.TaskSourceGitHub,
		ExternalID:   "42",
		ExternalURL:  "https://github.com/org/repo/issues/42",
		SourceStatus: "open",
		Title:        "Imported issue",
		Description:  "Body",
		Labels:       []string{"bug"},
		Type:         domain.TaskTypeBug,
		Priority:     domain.TaskPriorityP2,
		Status:       domain.StatusTodo,
	}
	tsk, err := svc.Import(ctx, req)
	if err != nil {
		t.Fatalf("Import create failed: %v", err)
	}

	// 2. Identical Import -> no event
	if _, err := svc.Import(ctx, req); err != nil {
		t.Fatalf("Import identical failed: %v", err)
	}

	// 3. Title-only Import update -> no status_changed event
	req.Title = "Imported issue v2"
	if _, err := svc.Import(ctx, req); err != nil {
		t.Fatalf("Import title update failed: %v", err)
	}

	// 4. Status change to blocked -> emits task.status_changed
	req.Status = domain.StatusBlocked
	if _, err := svc.Import(ctx, req); err != nil {
		t.Fatalf("Import status update failed: %v", err)
	}

	// 5. Label operations -> emit NO events
	if err := svc.AddLabel(ctx, tsk.ID, "triage"); err != nil {
		t.Fatalf("AddLabel failed: %v", err)
	}
	if err := svc.RemoveLabel(ctx, tsk.ID, "bug"); err != nil {
		t.Fatalf("RemoveLabel failed: %v", err)
	}
	if err := svc.SetLabels(ctx, tsk.ID, []string{"blocked"}); err != nil {
		t.Fatalf("SetLabels failed: %v", err)
	}

	if err := bus.Close(context.Background()); err != nil {
		t.Fatalf("close bus: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	if len(events) != 2 {
		t.Fatalf("expected 2 events (received + status_changed), got %d: %+v", len(events), events)
	}
	if events[0].Type != domain.EventReceived {
		t.Fatalf("event[0] = %s, want %s", events[0].Type, domain.EventReceived)
	}
	if events[1].Type != domain.EventStatusChanged {
		t.Fatalf("event[1] = %s, want %s", events[1].Type, domain.EventStatusChanged)
	}
	p1, err := eventbus.Decode[domain.TaskStatusChangedPayload](events[1])
	if err != nil {
		t.Fatalf("decode status_changed: %v", err)
	}
	if p1.ProjectID != projectID || p1.From != domain.StatusTodo || p1.To != domain.StatusBlocked {
		t.Fatalf("unexpected status_changed payload: %+v", p1)
	}
}
