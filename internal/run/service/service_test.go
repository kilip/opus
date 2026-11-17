package service_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/kilip/opus/ent"
	"github.com/kilip/opus/ent/enttest"
	"github.com/kilip/opus/internal/platform/eventbus"
	"github.com/kilip/opus/internal/run/domain"
	"github.com/kilip/opus/internal/run/repo"
	"github.com/kilip/opus/internal/run/service"
	_ "modernc.org/sqlite"
)

func setupService(t *testing.T) (*service.Service, *ent.Client, *eventbus.Bus) {
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

	svc := service.New(repo.New(client), service.Config{Bus: bus})
	return svc, client, bus
}

func TestService_LifecycleAndEvents(t *testing.T) {
	ctx := context.Background()
	svc, client, bus := setupService(t)

	var (
		mu     sync.Mutex
		events []eventbus.Event
	)
	if err := bus.Subscribe("test-run-events", func(_ context.Context, e eventbus.Event) error {
		mu.Lock()
		events = append(events, e)
		mu.Unlock()
		return nil
	}, domain.EventStarted, domain.EventFinished); err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	taskID := uuid.New()
	projectID := uuid.New()
	nodeID := uuid.New()

	sess, err := client.Session.Create().
		SetTaskID(taskID).
		SetNodeID(nodeID).
		SetProvider("agy").
		Save(ctx)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	run, err := svc.Create(ctx, domain.CreateInput{
		TaskID:    taskID,
		ProjectID: projectID,
		Mode:      domain.ModeVibe,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if run.State != domain.StateQueued || run.Mode != domain.ModeVibe {
		t.Fatalf("unexpected initial run: %+v", run)
	}

	branch := "feat/test-run"
	run, err = svc.Transition(ctx, run.ID, domain.StatePreparing, domain.TransitionInput{
		NodeID:           &nodeID,
		Branch:           &branch,
		IncrementAttempt: true,
	})
	if err != nil {
		t.Fatalf("Transition preparing: %v", err)
	}
	if run.StageAttempts != 1 || run.Branch != branch {
		t.Fatalf("unexpected preparing run: %+v", run)
	}

	run, err = svc.Transition(ctx, run.ID, domain.StateRunning, domain.TransitionInput{
		SessionID: &sess.ID,
	})
	if err != nil {
		t.Fatalf("Transition running: %v", err)
	}
	if run.StartedAt == nil || run.SessionID == nil || *run.SessionID != sess.ID {
		t.Fatalf("expected StartedAt and SessionID on running run: %+v", run)
	}

	report := json.RawMessage(`{"summary":"ok"}`)
	prNum := 42
	run, err = svc.Transition(ctx, run.ID, domain.StateMerged, domain.TransitionInput{
		PRNumber: &prNum,
		Report:   report,
	})
	if err != nil {
		t.Fatalf("Transition merged: %v", err)
	}
	if run.EndedAt == nil || run.PRNumber == nil || *run.PRNumber != 42 || string(run.Report) != `{"summary":"ok"}` {
		t.Fatalf("unexpected merged run: %+v", run)
	}

	if _, err := svc.Transition(ctx, run.ID, domain.StateRunning, domain.TransitionInput{}); !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("expected ErrInvalidTransition from merged, got %v", err)
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
	if len(events) != 2 {
		t.Fatalf("expected 2 bus events, got %d", len(events))
	}
	if events[0].Type != domain.EventStarted {
		t.Fatalf("event[0].Type = %s, want %s", events[0].Type, domain.EventStarted)
	}
	var started domain.RunStartedPayload
	if err := json.Unmarshal(events[0].Payload, &started); err != nil {
		t.Fatalf("unmarshal started: %v", err)
	}
	if started.RunID != run.ID || started.TaskID != taskID || started.ProjectID != projectID || started.Mode != domain.ModeVibe {
		t.Fatalf("unexpected started payload: %+v", started)
	}

	if events[1].Type != domain.EventFinished {
		t.Fatalf("event[1].Type = %s, want %s", events[1].Type, domain.EventFinished)
	}
	var finished domain.RunFinishedPayload
	if err := json.Unmarshal(events[1].Payload, &finished); err != nil {
		t.Fatalf("unmarshal finished: %v", err)
	}
	if finished.RunID != run.ID || finished.TaskID != taskID || finished.Mode != domain.ModeVibe || finished.Outcome != domain.StateMerged {
		t.Fatalf("unexpected finished payload: %+v", finished)
	}
}

func TestService_PauseResumeAndFailureReason(t *testing.T) {
	ctx := context.Background()
	svc, _, _ := setupService(t)

	run, err := svc.Create(ctx, domain.CreateInput{
		TaskID:    uuid.New(),
		ProjectID: uuid.New(),
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	until := time.Now().Add(5 * time.Minute).UTC()
	reason := "rate limited"
	run, err = svc.Transition(ctx, run.ID, domain.StatePaused, domain.TransitionInput{
		PausedUntil:   &until,
		FailureReason: &reason,
	})
	if err != nil {
		t.Fatalf("Transition paused: %v", err)
	}
	if run.PausedUntil == nil || run.EndedAt != nil {
		t.Fatalf("expected PausedUntil set and EndedAt nil on paused run: %+v", run)
	}

	run, err = svc.Transition(ctx, run.ID, domain.StateQueued, domain.TransitionInput{})
	if err != nil {
		t.Fatalf("Transition resume to queued: %v", err)
	}
	if run.PausedUntil != nil || run.State != domain.StateQueued {
		t.Fatalf("expected PausedUntil cleared on resume: %+v", run)
	}
}
