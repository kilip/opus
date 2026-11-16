package service_test

import (
	"context"
	"database/sql"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/kilip/opus/ent"
	"github.com/kilip/opus/ent/enttest"
	entnode "github.com/kilip/opus/ent/node"
	"github.com/kilip/opus/internal/node/domain"
	"github.com/kilip/opus/internal/node/repo"
	"github.com/kilip/opus/internal/node/service"
	"github.com/kilip/opus/internal/platform/eventbus"
	_ "modernc.org/sqlite"
)

func setupDetectorTestClient(t *testing.T) (*ent.Client, *sql.DB) {
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
		_ = db.Close()
	})
	return client, db
}

// AC2: Worker silent for 45 seconds is set offline and node.offline is published once.
// Tests use an injectable clock and do not sleep for real time.
func TestDetector_CheckStale_ClockInjection(t *testing.T) {
	client, _ := setupDetectorTestClient(t)
	r := repo.New(client)
	bus := eventbus.New()
	ctx := context.Background()

	var nowMu sync.Mutex
	currentTime := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	nowFunc := func() time.Time {
		nowMu.Lock()
		defer nowMu.Unlock()
		return currentTime
	}
	advanceTime := func(d time.Duration) {
		nowMu.Lock()
		defer nowMu.Unlock()
		currentTime = currentTime.Add(d)
	}

	svc := service.New(r, service.Config{
		Now: nowFunc,
		Bus: bus,
	})

	// Setup subscriber to track published events
	var (
		eventsMu sync.Mutex
		events   []eventbus.Event
	)
	err := bus.Subscribe("detector-sub", func(_ context.Context, e eventbus.Event) error {
		eventsMu.Lock()
		events = append(events, e)
		eventsMu.Unlock()
		return nil
	})
	if err != nil {
		t.Fatalf("subscribe failed: %v", err)
	}

	// Create master node (should NEVER be marked offline)
	master, err := client.Node.Create().
		SetName("master-1").
		SetRole(entnode.RoleMaster).
		SetStatus(entnode.StatusOnline).
		SetLastSeenAt(nowFunc()).
		Save(ctx)
	if err != nil {
		t.Fatalf("create master node failed: %v", err)
	}

	// Create worker 1 (stale: will exceed 45s)
	worker1, err := client.Node.Create().
		SetName("worker-1").
		SetRole(entnode.RoleWorker).
		SetStatus(entnode.StatusOnline).
		SetLastSeenAt(nowFunc()).
		Save(ctx)
	if err != nil {
		t.Fatalf("create worker 1 failed: %v", err)
	}

	// Create worker 2 (fresh: heartbeats will keep it online)
	worker2, err := client.Node.Create().
		SetName("worker-2").
		SetRole(entnode.RoleWorker).
		SetStatus(entnode.StatusOnline).
		SetLastSeenAt(nowFunc()).
		Save(ctx)
	if err != nil {
		t.Fatalf("create worker 2 failed: %v", err)
	}

	detector := service.NewDetector(svc, service.DetectorConfig{
		Now:     nowFunc,
		Timeout: service.DefaultDetectorTimeout,
	})

	// Advance time by 30 seconds (< 45s threshold)
	advanceTime(30 * time.Second)

	// Worker 2 sends heartbeat at +30s
	if err := svc.RecordHeartbeat(ctx, worker2.ID.String()); err != nil {
		t.Fatalf("record heartbeat worker 2 failed: %v", err)
	}

	// CheckStale at +30s: worker 1 was seen at 0s (age 30s < 45s), worker 2 seen at 30s.
	// Neither worker should be marked offline.
	if err := detector.CheckStale(ctx); err != nil {
		t.Fatalf("CheckStale failed: %v", err)
	}

	w1, err := client.Node.Get(ctx, worker1.ID)
	if err != nil || w1.Status != entnode.StatusOnline {
		t.Fatalf("expected worker 1 to still be online, got status=%v err=%v", w1.Status, err)
	}

	// Advance time by 20 seconds (now +50s from start; worker 1 age is 50s > 45s, worker 2 age is 20s < 45s)
	advanceTime(20 * time.Second)

	if err := detector.CheckStale(ctx); err != nil {
		t.Fatalf("CheckStale failed: %v", err)
	}

	// Worker 1 must be marked offline
	w1, err = client.Node.Get(ctx, worker1.ID)
	if err != nil || w1.Status != entnode.StatusOffline {
		t.Fatalf("expected worker 1 to be offline, got status=%v err=%v", w1.Status, err)
	}

	// Worker 2 must remain online
	w2, err := client.Node.Get(ctx, worker2.ID)
	if err != nil || w2.Status != entnode.StatusOnline {
		t.Fatalf("expected worker 2 to remain online, got status=%v err=%v", w2.Status, err)
	}

	// Master must remain online
	m, err := client.Node.Get(ctx, master.ID)
	if err != nil || m.Status != entnode.StatusOnline {
		t.Fatalf("expected master to remain online, got status=%v err=%v", m.Status, err)
	}

	// Wait briefly for event subscriber
	time.Sleep(10 * time.Millisecond)

	eventsMu.Lock()
	countOffline := 0
	for _, e := range events {
		if e.Type == domain.EventOffline && e.Subject == worker1.ID.String() {
			countOffline++
		}
	}
	eventsMu.Unlock()

	if countOffline != 1 {
		t.Fatalf("expected exactly 1 node.offline event for worker 1, got %d", countOffline)
	}

	// Second CheckStale without new heartbeats: worker 1 is already offline, publishes nothing
	if err := detector.CheckStale(ctx); err != nil {
		t.Fatalf("second CheckStale failed: %v", err)
	}

	time.Sleep(10 * time.Millisecond)
	eventsMu.Lock()
	countOfflineAfter := 0
	for _, e := range events {
		if e.Type == domain.EventOffline && e.Subject == worker1.ID.String() {
			countOfflineAfter++
		}
	}
	eventsMu.Unlock()

	if countOfflineAfter != 1 {
		t.Fatalf("expected still exactly 1 node.offline event, got %d", countOfflineAfter)
	}
}

// AC3: The detector goroutine stops on app shutdown (no goroutine leak, verified with a test).
func TestDetector_StartStop_NoGoroutineLeak(t *testing.T) {
	client, _ := setupDetectorTestClient(t)
	r := repo.New(client)
	svc := service.New(r, service.Config{})

	// Warm up runtime
	runtime.GC()
	baselineGoroutines := runtime.NumGoroutine()

	detector := service.NewDetector(svc, service.DetectorConfig{
		Interval: 10 * time.Millisecond,
		Timeout:  45 * time.Second,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := detector.Start(ctx); err != nil {
		t.Fatalf("detector start failed: %v", err)
	}

	// Detector is running, NumGoroutine should increase
	runningGoroutines := runtime.NumGoroutine()
	if runningGoroutines <= baselineGoroutines {
		t.Logf("running goroutines: %d, baseline: %d", runningGoroutines, baselineGoroutines)
	}

	// Stop detector
	detector.Stop()

	// Calling Stop again should be safe and idempotent
	detector.Stop()

	// Wait for goroutines to settle and verify no leak
	deadline := time.Now().Add(500 * time.Millisecond)
	var finalGoroutines int
	for time.Now().Before(deadline) {
		runtime.GC()
		finalGoroutines = runtime.NumGoroutine()
		if finalGoroutines <= baselineGoroutines {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if finalGoroutines > baselineGoroutines {
		t.Fatalf("expected <= %d goroutines, got %d (goroutine leak)", baselineGoroutines, finalGoroutines)
	}
}

// AC3: Detector never runs after database is closed.
func TestDetector_Stop_BeforeDBClose(t *testing.T) {
	client, db := setupDetectorTestClient(t)
	r := repo.New(client)
	svc := service.New(r, service.Config{})

	var dbClosed atomic.Bool
	var detectorRanAfterClose atomic.Bool

	detector := service.NewDetector(svc, service.DetectorConfig{
		Interval: 5 * time.Millisecond,
		Timeout:  45 * time.Second,
	})

	ctx := context.Background()
	if err := detector.Start(ctx); err != nil {
		t.Fatalf("detector start failed: %v", err)
	}

	// Let detector tick once or twice
	time.Sleep(20 * time.Millisecond)

	// Stop detector BEFORE closing DB
	detector.Stop()

	// Now close DB
	dbClosed.Store(true)
	_ = client.Close()
	_ = db.Close()

	if detectorRanAfterClose.Load() {
		t.Fatal("detector ran query after database was closed")
	}
}

// Test worker with nil LastSeenAt is detected as stale if online.
func TestDetector_CheckStale_NilLastSeenAt(t *testing.T) {
	client, _ := setupDetectorTestClient(t)
	r := repo.New(client)
	bus := eventbus.New()
	ctx := context.Background()

	svc := service.New(r, service.Config{
		Bus: bus,
	})

	// Create worker with nil LastSeenAt
	worker, err := client.Node.Create().
		SetID(uuid.Must(uuid.NewV7())).
		SetName("worker-nil-lastseen").
		SetRole(entnode.RoleWorker).
		SetStatus(entnode.StatusOnline).
		Save(ctx)
	if err != nil {
		t.Fatalf("create worker failed: %v", err)
	}

	detector := service.NewDetector(svc, service.DetectorConfig{})
	if err := detector.CheckStale(ctx); err != nil {
		t.Fatalf("CheckStale failed: %v", err)
	}

	w, err := client.Node.Get(ctx, worker.ID)
	if err != nil || w.Status != entnode.StatusOffline {
		t.Fatalf("expected worker to be marked offline, got status=%v err=%v", w.Status, err)
	}
}
