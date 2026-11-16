package service_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/kilip/opus/internal/platform/eventbus"
	"github.com/kilip/opus/internal/task/domain"
	"github.com/kilip/opus/internal/task/repo"
	"github.com/kilip/opus/internal/task/service"
)

func newService(bus ...*eventbus.Bus) *service.Task {
	var b *eventbus.Bus
	if len(bus) > 0 {
		b = bus[0]
	} else {
		b = eventbus.New()
	}
	return service.New(repo.NewMemory(func() time.Time {
		return time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	}), b)
}

func TestCreateRejectsBlankTitle(t *testing.T) {
	_, err := newService().Create(domain.CreateTaskRequest{Title: "  "})
	if !errors.Is(err, domain.ErrTaskTitleEmpty) {
		t.Fatalf("expected ErrTaskTitleEmpty, got %v", err)
	}
}

func TestUpdateRejectsBlankTitle(t *testing.T) {
	blank := ""
	_, err := newService().Update("t1", domain.UpdateTaskRequest{Title: &blank})
	if !errors.Is(err, domain.ErrTaskTitleEmpty) {
		t.Fatalf("expected ErrTaskTitleEmpty, got %v", err)
	}
}

func TestCreateActivityValidatesBeforeLookup(t *testing.T) {
	s := newService()

	_, err := s.CreateActivity("t1", domain.CreateActivityRequest{Body: " "})
	if !errors.Is(err, domain.ErrActivityBodyEmpty) {
		t.Fatalf("expected ErrActivityBodyEmpty, got %v", err)
	}

	_, err = s.CreateActivity("missing", domain.CreateActivityRequest{Body: "hi"})
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

	svc := newService(bus)

	created, err := svc.Create(domain.CreateTaskRequest{
		Title: "Emitted and Order Task",
	})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	nextStatus := domain.TaskStatusInProgress
	updated, err := svc.Update(created.ID, domain.UpdateTaskRequest{
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
	if p0.ID != created.ID || p0.Key != created.Key || p0.Source != created.Source || p0.Status != created.Status {
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
	if p1.ID != created.ID || p1.From != domain.TaskStatusTodo || p1.To != nextStatus {
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

	svc := newService(bus)

	created, err := svc.Create(domain.CreateTaskRequest{
		Title: "Same Status Task",
	})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	// Update with same status
	sameStatus := domain.TaskStatusTodo
	_, err = svc.Update(created.ID, domain.UpdateTaskRequest{
		Status: &sameStatus,
	})
	if err != nil {
		t.Fatalf("update same status: %v", err)
	}

	// Update title only (status nil)
	newTitle := "Updated Title Only"
	_, err = svc.Update(created.ID, domain.UpdateTaskRequest{
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

	svc := newService(bus)

	// Create with empty title fails
	_, err = svc.Create(domain.CreateTaskRequest{Title: "   "})
	if err == nil {
		t.Fatal("expected error on empty title create")
	}

	// Update non-existent task fails
	nextStatus := domain.TaskStatusDone
	_, err = svc.Update("nonexistent-task", domain.UpdateTaskRequest{Status: &nextStatus})
	if err == nil {
		t.Fatal("expected error on nonexistent update")
	}

	// Update with invalid title fails
	blank := " "
	_, err = svc.Update("t1", domain.UpdateTaskRequest{Title: &blank})
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

	svc := newService(bus)

	doneStatus := domain.TaskStatusDone
	created, err := svc.Create(domain.CreateTaskRequest{
		Title:  "Task Created As Done",
		Status: &doneStatus,
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
	if p.ID != created.ID || p.Status != domain.TaskStatusDone {
		t.Fatalf("expected status %s, got %s (payload: %+v)", domain.TaskStatusDone, p.Status, p)
	}
}

func TestTaskEvents_PublishFailureLoggedAndDoesNotFail(t *testing.T) {
	bus := eventbus.New()
	// Pre-close the bus so bus.Publish returns eventbus.ErrClosed
	if err := bus.Close(context.Background()); err != nil {
		t.Fatalf("close bus: %v", err)
	}

	svc := newService(bus)

	created, err := svc.Create(domain.CreateTaskRequest{
		Title: "Task Despite Closed Bus",
	})
	if err != nil {
		t.Fatalf("expected Create to succeed even with closed bus, got: %v", err)
	}
	if created.Title != "Task Despite Closed Bus" {
		t.Fatalf("unexpected title: %s", created.Title)
	}

	nextStatus := domain.TaskStatusReview
	updated, err := svc.Update(created.ID, domain.UpdateTaskRequest{
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

	svc := newService(bus)

	initialStatus := domain.TaskStatusBacklog
	created, err := svc.Create(domain.CreateTaskRequest{
		Title:  "Concurrent Task",
		Status: &initialStatus,
	})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	// Run concurrent updates on the created task
	stages := []domain.TaskStatus{
		domain.TaskStatusTodo,
		domain.TaskStatusInProgress,
		domain.TaskStatusReview,
		domain.TaskStatusDone,
	}

	const goroutines = 20
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			targetStatus := stages[idx%len(stages)]
			_, _ = svc.Update(created.ID, domain.UpdateTaskRequest{
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
		if ev.From != prevTo {
			t.Fatalf("event %d sequence break: expected from=%s, got from=%s (to=%s)", i, prevTo, ev.From, ev.To)
		}
		prevTo = ev.To
	}
}

func domainPtr[T any](v T) *T {
	return &v
}
