package service_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"sync"
	"testing"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/kilip/opus/ent"
	"github.com/kilip/opus/ent/enttest"
	"github.com/kilip/opus/internal/node"
	"github.com/kilip/opus/internal/platform/eventbus"
	"github.com/kilip/opus/internal/session/domain"
	"github.com/kilip/opus/internal/session/repo"
	"github.com/kilip/opus/internal/session/service"
	_ "modernc.org/sqlite"
)

func setupService(t *testing.T, onPromote func(domain.Session)) (*service.Service, *eventbus.Bus) {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+t.Name()+"?mode=memory&cache=shared&_pragma=foreign_keys(ON)&_pragma=busy_timeout(10000)")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db.SetMaxOpenConns(1)
	drv := entsql.OpenDB(dialect.SQLite, db)
	client := enttest.NewClient(t, enttest.WithOptions(ent.Driver(drv)))
	t.Cleanup(func() {
		_ = client.Close()
	})

	bus := eventbus.New()
	t.Cleanup(func() {
		_ = bus.Close(context.Background())
	})
	svc := service.New(repo.New(client), service.Config{
		Bus:       bus,
		OnPromote: onPromote,
	})
	return svc, bus
}

func TestService_FIFOQueuePromotion(t *testing.T) {
	ctx := context.Background()
	var promotedIDs []uuid.UUID
	svc, _ := setupService(t, func(s domain.Session) {
		promotedIDs = append(promotedIDs, s.ID)
	})

	taskID := uuid.New()
	nodeID := uuid.New()

	// 1. First session starts running immediately.
	s1, err := svc.Enqueue(ctx, domain.CreateInput{TaskID: taskID, NodeID: nodeID, Provider: "agy"})
	if err != nil {
		t.Fatalf("Enqueue s1: %v", err)
	}
	if s1.Status != domain.StatusRunning || s1.ProviderAccount != "default" {
		t.Fatalf("expected s1 running with default account, got status=%s account=%s", s1.Status, s1.ProviderAccount)
	}

	// 2. Second and third sessions stay queued in FIFO order.
	s2, err := svc.Enqueue(ctx, domain.CreateInput{TaskID: taskID, NodeID: nodeID, Provider: "agy"})
	if err != nil {
		t.Fatalf("Enqueue s2: %v", err)
	}
	s3, err := svc.Enqueue(ctx, domain.CreateInput{TaskID: taskID, NodeID: nodeID, Provider: "agy"})
	if err != nil {
		t.Fatalf("Enqueue s3: %v", err)
	}
	if s2.Status != domain.StatusQueued || s3.Status != domain.StatusQueued {
		t.Fatalf("expected s2,s3 queued, got %s, %s", s2.Status, s3.Status)
	}

	// 3. Completing s1 promotes s2 (oldest queued), leaving s3 queued.
	ended1, promoted1, err := svc.Complete(ctx, s1.ID)
	if err != nil {
		t.Fatalf("Complete s1: %v", err)
	}
	if ended1.Status != domain.StatusCompleted || promoted1 == nil || promoted1.ID != s2.ID {
		t.Fatalf("expected s2 promoted, got ended=%s promoted=%v", ended1.Status, promoted1)
	}
	gotS3, _ := svc.Get(ctx, s3.ID)
	if gotS3.Status != domain.StatusQueued {
		t.Fatalf("expected s3 still queued, got %s", gotS3.Status)
	}

	// 4. Failing s2 promotes s3.
	ended2, promoted2, err := svc.Fail(ctx, s2.ID, "agent crashed")
	if err != nil {
		t.Fatalf("Fail s2: %v", err)
	}
	if ended2.Status != domain.StatusFailed || ended2.FailureReason == nil || *ended2.FailureReason != "agent crashed" {
		t.Fatalf("unexpected failed s2: %+v", ended2)
	}
	if promoted2 == nil || promoted2.ID != s3.ID {
		t.Fatalf("expected s3 promoted, got %v", promoted2)
	}

	// 5. Cancelling s3 leaves nothing queued.
	ended3, promoted3, err := svc.Cancel(ctx, s3.ID, "user cancelled")
	if err != nil {
		t.Fatalf("Cancel s3: %v", err)
	}
	if ended3.Status != domain.StatusCancelled || promoted3 != nil {
		t.Fatalf("unexpected cancel result: ended=%s promoted=%v", ended3.Status, promoted3)
	}

	if len(promotedIDs) != 2 || promotedIDs[0] != s2.ID || promotedIDs[1] != s3.ID {
		t.Errorf("unexpected promotion order: %v", promotedIDs)
	}
}

func TestService_OrderedEventPersistenceAndWorkerDelivery(t *testing.T) {
	ctx := context.Background()
	svc, _ := setupService(t, nil)

	sess, err := svc.Enqueue(ctx, domain.CreateInput{
		TaskID:   uuid.New(),
		NodeID:   uuid.New(),
		Provider: "agy",
	})
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	// Concurrent RecordEvent calls must produce strictly monotonic 1..N sequences.
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			payload, _ := json.Marshal(map[string]int{"i": idx})
			if _, err := svc.RecordEvent(ctx, sess.ID, "agent_message_chunk", payload); err != nil {
				t.Errorf("RecordEvent %d: %v", idx, err)
			}
		}(i)
	}
	wg.Wait()

	// Also deliver an event via HandleWorkerEvent (node control channel path).
	raw, _ := json.Marshal(map[string]any{
		"sessionId": sess.ID.String(),
		"kind":      "tool_call",
		"name":      "run_command",
	})
	svc.HandleWorkerEvent(ctx, sess.NodeID.String(), node.Envelope{
		Type:    node.EnvelopeTypeEvent,
		ID:      "evt-1",
		Payload: raw,
	})

	events, err := svc.ListEventsAfter(ctx, sess.ID, 0)
	if err != nil {
		t.Fatalf("ListEventsAfter: %v", err)
	}
	if len(events) != 11 {
		t.Fatalf("expected 11 events, got %d", len(events))
	}
	for i, evt := range events {
		if evt.Seq != i+1 {
			t.Errorf("expected seq %d, got %d", i+1, evt.Seq)
		}
	}
	if events[10].Kind != "tool_call" {
		t.Errorf("expected last event kind tool_call, got %s", events[10].Kind)
	}

	// Mismatched nodeID must be ignored without mutating session or recording events.
	wrongNode := uuid.New().String()
	badRaw, _ := json.Marshal(map[string]any{
		"sessionId": sess.ID.String(),
		"kind":      "session_failed",
		"reason":    "spoofed",
	})
	svc.HandleWorkerEvent(ctx, wrongNode, node.Envelope{
		Type:    node.EnvelopeTypeEvent,
		ID:      "evt-spoof",
		Payload: badRaw,
	})

	afterSess, err := svc.Get(ctx, sess.ID)
	if err != nil {
		t.Fatalf("Get session: %v", err)
	}
	if afterSess.Status != domain.StatusRunning {
		t.Fatalf("expected session status to remain running, got %s", afterSess.Status)
	}
	afterEvents, err := svc.ListEventsAfter(ctx, sess.ID, 0)
	if err != nil {
		t.Fatalf("ListEventsAfter: %v", err)
	}
	if len(afterEvents) != 11 {
		t.Fatalf("expected event count to remain 11, got %d", len(afterEvents))
	}
}
