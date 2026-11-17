package service_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kilip/opus/internal/tracker/domain"
	"github.com/kilip/opus/internal/tracker/repo"
	"github.com/kilip/opus/internal/tracker/service"
)

func TestPoller_AuthFailureSkipsSyncAndRecordsError(t *testing.T) {
	client, _ := setupTrackerClient(t)
	r := repo.New(client)
	ctx := context.Background()

	ghProj, err := client.Project.Create().
		SetSlug("gh-proj").
		SetName("GH Project").
		SetRepoProvider("github").
		SetRepoRef("org/gh-proj").
		SetTrackerProvider("github").
		SetTrackerRef("org/gh-proj").
		Save(ctx)
	if err != nil {
		t.Fatalf("create gh project: %v", err)
	}

	otherProj, err := client.Project.Create().
		SetSlug("jira-proj").
		SetName("Jira Project").
		SetRepoProvider("github").
		SetRepoRef("org/jira-proj").
		SetTrackerProvider("jira").
		SetTrackerRef("JIRA-1").
		Save(ctx)
	if err != nil {
		t.Fatalf("create non-gh project: %v", err)
	}

	var listCalls atomic.Int32
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, nil))

	poller := service.NewPoller(r, service.PollerConfig{
		Logger: logger,
		Runner: func(_ context.Context, args ...string) ([]byte, error) {
			if len(args) >= 2 && args[0] == "auth" && args[1] == "status" {
				return nil, errors.New("not logged in")
			}
			listCalls.Add(1)
			return []byte("[]"), nil
		},
	})

	poller.SyncOnce(ctx)

	if listCalls.Load() != 0 {
		t.Fatalf("expected 0 issue/pr list calls when auth fails, got %d", listCalls.Load())
	}

	ghSync, err := r.GetSyncByProject(ctx, ghProj.ID.String())
	if err != nil {
		t.Fatalf("GetSyncByProject gh: %v", err)
	}
	if !strings.Contains(ghSync.LastError, "not logged in") {
		t.Fatalf("expected auth error on github project, got %q", ghSync.LastError)
	}

	otherSync, err := r.GetSyncByProject(ctx, otherProj.ID.String())
	if err != nil {
		t.Fatalf("GetSyncByProject other: %v", err)
	}
	if otherSync.LastError != "" {
		t.Fatalf("expected no error on non-github project, got %q", otherSync.LastError)
	}

	if !strings.Contains(logBuf.String(), "github auth status failed") {
		t.Fatalf("expected slog auth failure message, got %q", logBuf.String())
	}
}

func TestPoller_AuthRecoveryAndPerProjectIsolation(t *testing.T) {
	client, _ := setupTrackerClient(t)
	r := repo.New(client)
	ctx := context.Background()

	p1, err := client.Project.Create().
		SetSlug("proj-one").
		SetName("Project One").
		SetRepoProvider("github").
		SetRepoRef("org/one").
		SetTrackerProvider("github").
		SetTrackerRef("org/one").
		Save(ctx)
	if err != nil {
		t.Fatalf("create p1: %v", err)
	}
	p2, err := client.Project.Create().
		SetSlug("proj-two").
		SetName("Project Two").
		SetRepoProvider("github").
		SetRepoRef("org/two").
		SetTrackerProvider("github").
		SetTrackerRef("org/two").
		Save(ctx)
	if err != nil {
		t.Fatalf("create p2: %v", err)
	}

	// Pre-seed p1 with an existing cursor and prior auth error
	prevSync := time.Date(2026, 11, 16, 10, 0, 0, 0, time.UTC)
	if _, err := r.SaveSyncSuccess(ctx, p1.ID.String(), "2026-11-16T10:00:00Z", prevSync); err != nil {
		t.Fatalf("seed p1 sync: %v", err)
	}
	if err := r.UpsertLastError(ctx, p1.ID.String(), "stale auth error"); err != nil {
		t.Fatalf("seed p1 error: %v", err)
	}
	if err := r.UpsertLastError(ctx, p2.ID.String(), "stale auth error"); err != nil {
		t.Fatalf("seed p2 error: %v", err)
	}

	fixedNow := time.Date(2026, 11, 17, 2, 0, 0, 0, time.UTC)
	poller := service.NewPoller(r, service.PollerConfig{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		Now:    func() time.Time { return fixedNow },
		Runner: func(_ context.Context, args ...string) ([]byte, error) {
			if len(args) >= 2 && args[0] == "auth" && args[1] == "status" {
				return []byte("Logged in"), nil
			}
			repoArg := ""
			for i := 0; i < len(args)-1; i++ {
				if args[i] == "--repo" {
					repoArg = args[i+1]
				}
			}
			// p1 hits rate limit; p2 succeeds with an issue
			if repoArg == "org/one" {
				return nil, domain.ErrRateLimited
			}
			if args[0] == "issue" {
				return []byte(`[{"number":7,"title":"Fix bug","body":"","state":"OPEN","stateReason":"","labels":[],"updatedAt":"2026-11-17T01:45:00Z","url":"https://github.com/org/two/issues/7"}]`), nil
			}
			return []byte(`[]`), nil
		},
	})

	poller.SyncOnce(ctx)

	// p1 failed with ErrRateLimited: last_error recorded, cursor and last_sync_at preserved!
	s1, err := r.GetSyncByProject(ctx, p1.ID.String())
	if err != nil {
		t.Fatalf("GetSyncByProject p1: %v", err)
	}
	if !strings.Contains(s1.LastError, domain.ErrRateLimited.Error()) {
		t.Fatalf("expected rate limit error on p1, got %q", s1.LastError)
	}
	if s1.Cursor != "2026-11-16T10:00:00Z" {
		t.Fatalf("expected p1 cursor preserved on failure, got %q", s1.Cursor)
	}
	if s1.LastSyncAt == nil || !s1.LastSyncAt.Equal(prevSync) {
		t.Fatalf("expected p1 LastSyncAt preserved, got %v", s1.LastSyncAt)
	}

	// p2 succeeded despite p1 failing: last_error cleared, cursor and last_sync_at updated!
	s2, err := r.GetSyncByProject(ctx, p2.ID.String())
	if err != nil {
		t.Fatalf("GetSyncByProject p2: %v", err)
	}
	if s2.LastError != "" {
		t.Fatalf("expected cleared LastError on p2, got %q", s2.LastError)
	}
	if s2.Cursor != "2026-11-17T01:45:00Z" {
		t.Fatalf("expected p2 cursor updated to issue updatedAt, got %q", s2.Cursor)
	}
	if s2.LastSyncAt == nil || !s2.LastSyncAt.Equal(fixedNow) {
		t.Fatalf("expected p2 LastSyncAt %v, got %v", fixedNow, s2.LastSyncAt)
	}
}

func TestPoller_StartStop_StartupCycleAndNoGoroutineLeak(t *testing.T) {
	client, db := setupTrackerClient(t)
	r := repo.New(client)

	runtime.GC()
	baseline := runtime.NumGoroutine()

	var authCalls atomic.Int32
	var mu sync.Mutex
	var stopped bool

	poller := service.NewPoller(r, service.PollerConfig{
		PollInterval: 15 * time.Millisecond,
		Logger:       slog.New(slog.NewTextHandler(io.Discard, nil)),
		Runner: func(_ context.Context, args ...string) ([]byte, error) {
			if len(args) >= 2 && args[0] == "auth" && args[1] == "status" {
				authCalls.Add(1)
			}
			return []byte("[]"), nil
		},
		OnStop: func() {
			mu.Lock()
			stopped = true
			mu.Unlock()
		},
	})

	ctx := context.Background()
	if err := poller.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	if !poller.Running() {
		t.Fatal("expected Running() to be true")
	}
	// Idempotent Start
	if err := poller.Start(ctx); err != nil {
		t.Fatalf("second Start failed: %v", err)
	}

	// Wait for startup cycle + at least one tick
	deadline := time.Now().Add(time.Second)
	for authCalls.Load() < 2 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if authCalls.Load() < 2 {
		t.Fatalf("expected startup + periodic poll cycles, got %d auth calls", authCalls.Load())
	}

	poller.Stop()
	if poller.Running() {
		t.Fatal("expected Running() to be false after Stop")
	}
	// Idempotent Stop
	poller.Stop()

	mu.Lock()
	wasStopped := stopped
	mu.Unlock()
	if !wasStopped {
		t.Fatal("expected OnStop callback to fire")
	}

	// Close DB safely after Stop
	_ = client.Close()
	_ = db.Close()

	// Verify no goroutine leak
	leakDeadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(leakDeadline) {
		runtime.GC()
		if runtime.NumGoroutine() <= baseline {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	if got := runtime.NumGoroutine(); got > baseline {
		t.Fatalf("goroutine leak: baseline=%d final=%d", baseline, got)
	}
}
