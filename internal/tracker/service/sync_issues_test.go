package service_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kilip/opus/ent"
	"github.com/kilip/opus/internal/platform/eventbus"
	"github.com/kilip/opus/internal/task"
	"github.com/kilip/opus/internal/tracker/provider/github"
	"github.com/kilip/opus/internal/tracker/repo"
	"github.com/kilip/opus/internal/tracker/service"
)

func createGHProject(t *testing.T, client *ent.Client, slug, ref string, trackerLabels map[string]string) *ent.Project {
	t.Helper()
	b := client.Project.Create().
		SetSlug(slug).
		SetName(slug).
		SetRepoProvider("github").
		SetRepoRef(ref).
		SetTrackerProvider("github").
		SetTrackerRef(ref)
	if trackerLabels != nil {
		b.SetTrackerLabels(trackerLabels)
	}
	p, err := b.Save(context.Background())
	if err != nil {
		t.Fatalf("create project %s: %v", slug, err)
	}
	return p
}

// AC1 & AC4: Fake runner + fixtures test verifying cursor usage, max updatedAt advancement,
// unknown open issue creation (github, owner/repo#N, OP-<n>, projectId), and unknown closed issue skip.
func TestPoller_InboundIssueSync_CursorAndFixture(t *testing.T) {
	client, _ := setupTrackerClient(t)
	r := repo.New(client)
	bus := eventbus.New()
	t.Cleanup(func() { _ = bus.Close(context.Background()) })
	taskFeature := task.New(client, bus)
	ctx := context.Background()

	proj := createGHProject(t, client, "opus", "kilip/opus", nil)

	fixtureBytes, err := os.ReadFile(filepath.Join("..", "provider", "github", "testdata", "issues.json"))
	if err != nil {
		t.Fatalf("read issues fixture: %v", err)
	}

	var issueCalls [][]string
	payload := fixtureBytes
	var failNext bool

	now1 := time.Date(2026, 11, 17, 2, 0, 0, 0, time.UTC)
	now := now1

	poller := service.NewPoller(r, service.PollerConfig{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		Tasks:  taskFeature,
		Now:    func() time.Time { return now },
		Runner: func(_ context.Context, args ...string) ([]byte, error) {
			if len(args) >= 2 && args[0] == "auth" && args[1] == "status" {
				return []byte("Logged in"), nil
			}
			if len(args) > 0 && args[0] == "issue" {
				issueCalls = append(issueCalls, slices.Clone(args))
				if failNext {
					return nil, errors.New("gh unavailable")
				}
				return payload, nil
			}
			return []byte("[]"), nil
		},
	})

	// Cycle 1: no cursor initially, fixture has #104 (OPEN, 2026-11-17T01:30:00Z) and #99 (CLOSED, 2026-11-16T18:45:12Z)
	poller.SyncOnce(ctx)

	if len(issueCalls) != 1 {
		t.Fatalf("expected 1 issue list call, got %d", len(issueCalls))
	}
	if slices.Contains(issueCalls[0], "--search") {
		t.Fatalf("expected no --search on initial sync, got %v", issueCalls[0])
	}

	// #104 imported, #99 (unknown closed) skipped
	tsk, err := taskFeature.FindByExternal(ctx, task.SourceGitHub, "kilip/opus#104")
	if err != nil {
		t.Fatalf("FindByExternal #104: %v", err)
	}
	if tsk.Key != "OP-1" || tsk.ProjectID != proj.ID.String() || tsk.Source != task.SourceGitHub {
		t.Fatalf("unexpected imported task metadata: %+v", tsk)
	}
	if tsk.Status != task.StatusInProgress {
		t.Fatalf("expected status in-progress, got %q", tsk.Status)
	}
	if !slices.Equal(tsk.Labels, []string{"feat"}) {
		t.Fatalf("expected plain label [feat], got %v", tsk.Labels)
	}
	if _, err := taskFeature.FindByExternal(ctx, task.SourceGitHub, "kilip/opus#99"); !errors.Is(err, task.ErrTaskNotFound) {
		t.Fatalf("expected unknown closed issue #99 to be skipped, got err=%v", err)
	}

	syncState, err := r.GetSyncByProject(ctx, proj.ID.String())
	if err != nil {
		t.Fatalf("GetSyncByProject: %v", err)
	}
	if syncState.Cursor != "2026-11-17T01:30:00Z" {
		t.Fatalf("expected cursor 2026-11-17T01:30:00Z, got %q", syncState.Cursor)
	}

	// Cycle 2: uses cursor in --search and advances to newer updatedAt
	now = now1.Add(10 * time.Minute)
	payload = []byte(`[
		{"number":106,"title":"Follow-up task","body":"Details","state":"OPEN","stateReason":"","labels":[],"updatedAt":"2026-11-17T02:05:00Z","url":"https://github.com/kilip/opus/issues/106"}
	]`)
	poller.SyncOnce(ctx)

	if len(issueCalls) != 2 {
		t.Fatalf("expected 2 issue list calls, got %d", len(issueCalls))
	}
	wantSearchIdx := slices.Index(issueCalls[1], "--search")
	if wantSearchIdx == -1 || wantSearchIdx+1 >= len(issueCalls[1]) || issueCalls[1][wantSearchIdx+1] != "updated:>=2026-11-17T01:30:00Z" {
		t.Fatalf("expected --search updated:>=2026-11-17T01:30:00Z, got %v", issueCalls[1])
	}

	tsk106, err := taskFeature.FindByExternal(ctx, task.SourceGitHub, "kilip/opus#106")
	if err != nil {
		t.Fatalf("FindByExternal #106: %v", err)
	}
	if tsk106.Key != "OP-2" {
		t.Fatalf("expected OP-2, got %q", tsk106.Key)
	}

	syncState, _ = r.GetSyncByProject(ctx, proj.ID.String())
	if syncState.Cursor != "2026-11-17T02:05:00Z" {
		t.Fatalf("expected cursor advanced to 2026-11-17T02:05:00Z, got %q", syncState.Cursor)
	}

	// Cycle 3: runner fails -> cursor does NOT advance and last_error is recorded
	failNext = true
	now = now1.Add(20 * time.Minute)
	poller.SyncOnce(ctx)

	syncState, _ = r.GetSyncByProject(ctx, proj.ID.String())
	if syncState.Cursor != "2026-11-17T02:05:00Z" {
		t.Fatalf("expected cursor preserved on failure, got %q", syncState.Cursor)
	}
	if syncState.LastError == "" {
		t.Fatal("expected non-empty LastError on failure")
	}
}

// AC2: Table-driven test for status precedence, type:*, priority:*, plain labels, SourceStatus, and project override.
func TestPoller_InboundIssueLabelMapping(t *testing.T) {
	tests := []struct {
		name            string
		projectLabels   map[string]string
		issueLabels     []string
		state           string
		wantStatus      task.TaskStatus
		wantType        task.TaskType
		wantPriority    task.TaskPriority
		wantPlainLabels []string
		wantSourceState string
	}{
		{
			name:            "precedence blocked beats review and in-progress",
			issueLabels:     []string{"status:backlog", "status:todo", "status:in-progress", "status:review", "status:blocked"},
			state:           "OPEN",
			wantStatus:      task.StatusBlocked,
			wantType:        task.TypeFeat,
			wantPriority:    task.PriorityP1,
			wantPlainLabels: []string{},
			wantSourceState: "open",
		},
		{
			name:            "precedence review beats in-progress",
			issueLabels:     []string{"status:in-progress", "status:review", "status:todo"},
			state:           "OPEN",
			wantStatus:      task.StatusReview,
			wantType:        task.TypeFeat,
			wantPriority:    task.PriorityP1,
			wantPlainLabels: []string{},
			wantSourceState: "open",
		},
		{
			name:            "precedence in-progress beats todo and backlog",
			issueLabels:     []string{"status:backlog", "status:todo", "status:in-progress"},
			state:           "OPEN",
			wantStatus:      task.StatusInProgress,
			wantType:        task.TypeFeat,
			wantPriority:    task.PriorityP1,
			wantPlainLabels: []string{},
			wantSourceState: "open",
		},
		{
			name:            "precedence todo beats backlog",
			issueLabels:     []string{"status:backlog", "status:todo"},
			state:           "OPEN",
			wantStatus:      task.StatusTodo,
			wantType:        task.TypeFeat,
			wantPriority:    task.PriorityP1,
			wantPlainLabels: []string{},
			wantSourceState: "open",
		},
		{
			name:            "backlog only",
			issueLabels:     []string{"status:backlog"},
			state:           "OPEN",
			wantStatus:      task.StatusBacklog,
			wantType:        task.TypeFeat,
			wantPriority:    task.PriorityP1,
			wantPlainLabels: []string{},
			wantSourceState: "open",
		},
		{
			name:            "type priority and plain labels extracted",
			issueLabels:     []string{"type:bug", "priority:p0", "backend", "status:todo", "urgent"},
			state:           "OPEN",
			wantStatus:      task.StatusTodo,
			wantType:        task.TypeBug,
			wantPriority:    task.PriorityP0,
			wantPlainLabels: []string{"backend", "urgent"},
			wantSourceState: "open",
		},
		{
			name:            "numeric priority and refactor type",
			issueLabels:     []string{"type:refactor", "priority:2", "tech-debt"},
			state:           "OPEN",
			wantStatus:      task.StatusTodo,
			wantType:        task.TypeRefactor,
			wantPriority:    task.PriorityP2,
			wantPlainLabels: []string{"tech-debt"},
			wantSourceState: "open",
		},
		{
			name:            "project override maps doing to in-progress and treats old status:in-progress as plain label",
			projectLabels:   map[string]string{"in-progress": "doing"},
			issueLabels:     []string{"doing", "status:in-progress", "type:docs", "priority:3"},
			state:           "OPEN",
			wantStatus:      task.StatusInProgress,
			wantType:        task.TypeDocs,
			wantPriority:    task.PriorityP3,
			wantPlainLabels: []string{"status:in-progress"},
			wantSourceState: "open",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client, _ := setupTrackerClient(t)
			r := repo.New(client)
			taskFeature := task.New(client, eventbus.New())
			ctx := context.Background()

			proj := createGHProject(t, client, "label-proj", "org/labels", tc.projectLabels)

			ghLabels := make([]github.Label, len(tc.issueLabels))
			for i, name := range tc.issueLabels {
				ghLabels[i] = github.Label{Name: name}
			}
			issuesJSON, err := json.Marshal([]github.Issue{{
				Number:    10,
				Title:     "Label mapping issue",
				Body:      "Body",
				State:     tc.state,
				Labels:    ghLabels,
				UpdatedAt: time.Date(2026, 11, 17, 1, 0, 0, 0, time.UTC),
				URL:       "https://github.com/org/labels/issues/10",
			}})
			if err != nil {
				t.Fatalf("marshal issues: %v", err)
			}

			poller := service.NewPoller(r, service.PollerConfig{
				Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
				Tasks:  taskFeature,
				Runner: func(_ context.Context, args ...string) ([]byte, error) {
					if len(args) >= 2 && args[0] == "auth" {
						return []byte("ok"), nil
					}
					if len(args) > 0 && args[0] == "issue" {
						return issuesJSON, nil
					}
					return []byte("[]"), nil
				},
			})

			poller.SyncOnce(ctx)

			got, err := taskFeature.FindByExternal(ctx, task.SourceGitHub, proj.TrackerRef+"#10")
			if err != nil {
				t.Fatalf("FindByExternal: %v", err)
			}
			if got.Status != tc.wantStatus {
				t.Errorf("Status = %q, want %q", got.Status, tc.wantStatus)
			}
			if got.Type != tc.wantType {
				t.Errorf("Type = %q, want %q", got.Type, tc.wantType)
			}
			if got.Priority != tc.wantPriority {
				t.Errorf("Priority = %q, want %q", got.Priority, tc.wantPriority)
			}
			if !slices.Equal(got.Labels, tc.wantPlainLabels) {
				t.Errorf("Labels = %v, want %v", got.Labels, tc.wantPlainLabels)
			}
			if got.SourceStatus != tc.wantSourceState {
				t.Errorf("SourceStatus = %q, want %q", got.SourceStatus, tc.wantSourceState)
			}
		})
	}
}

// AC3 & AC5: Closed issue -> done (wins over status label); reopened issue -> todo; title & description updated.
func TestPoller_InboundIssueCloseReopenAndContentUpdate(t *testing.T) {
	client, _ := setupTrackerClient(t)
	r := repo.New(client)
	taskFeature := task.New(client, eventbus.New())
	ctx := context.Background()

	proj := createGHProject(t, client, "lifecycle", "org/lifecycle", nil)

	var current []github.Issue
	now := time.Date(2026, 11, 17, 1, 0, 0, 0, time.UTC)

	poller := service.NewPoller(r, service.PollerConfig{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		Tasks:  taskFeature,
		Now:    func() time.Time { return now },
		Runner: func(_ context.Context, args ...string) ([]byte, error) {
			if len(args) >= 2 && args[0] == "auth" {
				return []byte("ok"), nil
			}
			if len(args) > 0 && args[0] == "issue" {
				b, _ := json.Marshal(current)
				return b, nil
			}
			return []byte("[]"), nil
		},
	})

	// Cycle 1: Open issue imported
	current = []github.Issue{{
		Number:    5,
		Title:     "Initial title",
		Body:      "Initial body",
		State:     "OPEN",
		Labels:    []github.Label{{Name: "status:in-progress"}},
		UpdatedAt: now,
		URL:       "https://github.com/org/lifecycle/issues/5",
	}}
	poller.SyncOnce(ctx)

	tsk, err := taskFeature.FindByExternal(ctx, task.SourceGitHub, proj.TrackerRef+"#5")
	if err != nil {
		t.Fatalf("FindByExternal: %v", err)
	}
	if tsk.Status != task.StatusInProgress {
		t.Fatalf("cycle 1 status = %q, want in-progress", tsk.Status)
	}

	// Cycle 2: Closed issue (still carries status:in-progress label) -> done, title & body updated
	now = now.Add(5 * time.Minute)
	current = []github.Issue{{
		Number:    5,
		Title:     "Updated title",
		Body:      "Updated body",
		State:     "CLOSED",
		Labels:    []github.Label{{Name: "status:in-progress"}},
		UpdatedAt: now,
		URL:       "https://github.com/org/lifecycle/issues/5",
	}}
	poller.SyncOnce(ctx)

	tsk, _ = taskFeature.FindByExternal(ctx, task.SourceGitHub, proj.TrackerRef+"#5")
	if tsk.Status != task.StatusDone {
		t.Fatalf("cycle 2 closed status = %q, want done", tsk.Status)
	}
	if tsk.Title != "Updated title" || tsk.Description != "Updated body" {
		t.Fatalf("expected title/body updated, got title=%q body=%q", tsk.Title, tsk.Description)
	}
	if tsk.SourceStatus != "closed" {
		t.Fatalf("expected sourceStatus closed, got %q", tsk.SourceStatus)
	}

	// Cycle 3: Reopened issue (open, no status labels) -> todo
	now = now.Add(5 * time.Minute)
	current = []github.Issue{{
		Number:    5,
		Title:     "Reopened title",
		Body:      "Reopened body",
		State:     "OPEN",
		Labels:    []github.Label{},
		UpdatedAt: now,
		URL:       "https://github.com/org/lifecycle/issues/5",
	}}
	poller.SyncOnce(ctx)

	tsk, _ = taskFeature.FindByExternal(ctx, task.SourceGitHub, proj.TrackerRef+"#5")
	if tsk.Status != task.StatusTodo {
		t.Fatalf("cycle 3 reopened status = %q, want todo", tsk.Status)
	}
	if tsk.SourceStatus != "open" {
		t.Fatalf("expected sourceStatus open, got %q", tsk.SourceStatus)
	}
}

// AC6: Local UpdatedAt > SyncedAt ignores remote status/labels; SourceStatus == "deleted" is skipped.
func TestPoller_InboundIssueConflictAndTombstone(t *testing.T) {
	client, _ := setupTrackerClient(t)
	r := repo.New(client)
	taskFeature := task.New(client, eventbus.New())
	ctx := context.Background()

	proj := createGHProject(t, client, "conflict", "org/conflict", nil)

	// Pre-create tombstoned task (#1)
	deletedStatus := task.StatusDone
	_, err := taskFeature.Import(ctx, task.ImportTaskRequest{
		ProjectID:    proj.ID.String(),
		Title:        "Tombstoned",
		Description:  "Do not touch",
		Source:       task.SourceGitHub,
		ExternalID:   "org/conflict#1",
		SourceStatus: "deleted",
		Status:       deletedStatus,
	})
	if err != nil {
		t.Fatalf("create tombstoned task: %v", err)
	}

	t0 := time.Date(2026, 11, 17, 1, 0, 0, 0, time.UTC)
	now := t0

	issues := []github.Issue{
		{
			Number:    1,
			Title:     "Remote changed tombstoned",
			Body:      "New body",
			State:     "OPEN",
			Labels:    []github.Label{{Name: "status:blocked"}},
			UpdatedAt: t0,
			URL:       "https://github.com/org/conflict/issues/1",
		},
		{
			Number:    2,
			Title:     "Conflict issue",
			Body:      "Initial body",
			State:     "OPEN",
			Labels:    []github.Label{{Name: "status:todo"}, {Name: "keep-me"}},
			UpdatedAt: t0,
			URL:       "https://github.com/org/conflict/issues/2",
		},
	}

	poller := service.NewPoller(r, service.PollerConfig{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		Tasks:  taskFeature,
		Now:    func() time.Time { return now },
		Runner: func(_ context.Context, args ...string) ([]byte, error) {
			if len(args) >= 2 && args[0] == "auth" {
				return []byte("ok"), nil
			}
			if len(args) > 0 && args[0] == "issue" {
				b, _ := json.Marshal(issues)
				return b, nil
			}
			return []byte("[]"), nil
		},
	})

	// Cycle 1: imports #2, skips #1
	poller.SyncOnce(ctx)

	tomb, _ := taskFeature.FindByExternal(ctx, task.SourceGitHub, "org/conflict#1")
	if tomb.Title != "Tombstoned" || tomb.Status != task.StatusDone || tomb.SourceStatus != "deleted" {
		t.Fatalf("tombstoned task was modified: %+v", tomb)
	}

	tsk2, _ := taskFeature.FindByExternal(ctx, task.SourceGitHub, "org/conflict#2")
	if tsk2.SyncedAt == nil || !tsk2.SyncedAt.Equal(t0) {
		t.Fatalf("expected SyncedAt=%v, got %v", t0, tsk2.SyncedAt)
	}

	// Simulate local edit AFTER t0 (set UpdatedAt > SyncedAt in DB alongside local status/labels)
	localEditTime := t0.Add(2 * time.Minute)
	err = client.Task.UpdateOneID(uuid.MustParse(tsk2.ID)).
		SetStatus("in-progress").
		SetLabels([]string{"local-label"}).
		SetUpdatedAt(localEditTime).
		Exec(ctx)
	if err != nil {
		t.Fatalf("local update failed: %v", err)
	}

	// Cycle 2: remote changes title, description, AND status:blocked + remote-label
	now = t0.Add(5 * time.Minute)
	issues[1].Title = "Remote title updated"
	issues[1].Body = "Remote body updated"
	issues[1].Labels = []github.Label{{Name: "status:blocked"}, {Name: "remote-label"}}
	issues[1].UpdatedAt = now
	poller.SyncOnce(ctx)

	tsk2After, _ := taskFeature.FindByExternal(ctx, task.SourceGitHub, "org/conflict#2")
	if tsk2After.Title != "Remote title updated" || tsk2After.Description != "Remote body updated" {
		t.Fatalf("expected remote title/description applied, got title=%q body=%q", tsk2After.Title, tsk2After.Description)
	}
	if tsk2After.Status != task.StatusInProgress {
		t.Fatalf("expected local status in-progress preserved, got %q", tsk2After.Status)
	}
	if !slices.Equal(tsk2After.Labels, []string{"local-label"}) {
		t.Fatalf("expected local labels [local-label] preserved, got %v", tsk2After.Labels)
	}
	if tsk2After.SyncedAt == nil || !tsk2After.SyncedAt.Equal(t0) {
		t.Fatalf("expected SyncedAt preserved at %v while local edit is newer, got %v", t0, tsk2After.SyncedAt)
	}
}

// AC7: Second cycle with identical remote data causes 0 DB writes and 0 bus events.
func TestPoller_InboundIssueSync_IdempotentSecondCycle(t *testing.T) {
	client, _ := setupTrackerClient(t)
	r := repo.New(client)
	bus := eventbus.New()
	taskFeature := task.New(client, bus)
	ctx := context.Background()

	var (
		evMu   sync.Mutex
		events []eventbus.Event
	)
	if err := bus.Subscribe("count-sub", func(_ context.Context, e eventbus.Event) error {
		evMu.Lock()
		events = append(events, e)
		evMu.Unlock()
		return nil
	}); err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	_ = createGHProject(t, client, "idempotent", "org/idempotent", nil)

	var taskMutations atomic.Int32
	client.Task.Use(func(next ent.Mutator) ent.Mutator {
		return ent.MutateFunc(func(ctx context.Context, m ent.Mutation) (ent.Value, error) {
			taskMutations.Add(1)
			return next.Mutate(ctx, m)
		})
	})

	t0 := time.Date(2026, 11, 17, 1, 0, 0, 0, time.UTC)
	now := t0
	payload := []byte(`[{"number":12,"title":"Stable issue","body":"Same body","state":"OPEN","stateReason":"","labels":[{"name":"status:todo"},{"name":"api"}],"updatedAt":"2026-11-17T01:00:00Z","url":"https://github.com/org/idempotent/issues/12"}]`)

	poller := service.NewPoller(r, service.PollerConfig{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		Tasks:  taskFeature,
		Now:    func() time.Time { return now },
		Runner: func(_ context.Context, args ...string) ([]byte, error) {
			if len(args) >= 2 && args[0] == "auth" {
				return []byte("ok"), nil
			}
			if len(args) > 0 && args[0] == "issue" {
				return payload, nil
			}
			return []byte("[]"), nil
		},
	})

	// Cycle 1: creates 1 task (1 Task mutation, 1 event)
	poller.SyncOnce(ctx)
	if taskMutations.Load() != 1 {
		t.Fatalf("expected 1 task mutation on cycle 1, got %d", taskMutations.Load())
	}

	// Reset mutation counter and advance wall clock `now`
	taskMutations.Store(0)
	now = t0.Add(10 * time.Minute)

	// Cycle 2: identical remote data -> 0 Task DB writes and 0 new events!
	poller.SyncOnce(ctx)
	if got := taskMutations.Load(); got != 0 {
		t.Fatalf("expected 0 task DB writes on cycle 2, got %d", got)
	}

	if err := bus.Close(ctx); err != nil {
		t.Fatalf("close bus: %v", err)
	}
	evMu.Lock()
	defer evMu.Unlock()
	if len(events) != 1 || events[0].Type != task.EventReceived {
		t.Fatalf("expected exactly 1 event (task.received from cycle 1) and 0 on cycle 2, got %d: %+v", len(events), events)
	}
}

// AC8: Malformed JSON / failing project records last_error, preserves cursor, and does not block other projects.
func TestPoller_InboundIssueSync_MalformedJSONDoesNotBlockOtherProjects(t *testing.T) {
	client, _ := setupTrackerClient(t)
	r := repo.New(client)
	taskFeature := task.New(client, eventbus.New())
	ctx := context.Background()

	pBad := createGHProject(t, client, "bad-proj", "org/bad", nil)
	pGood := createGHProject(t, client, "good-proj", "org/good", nil)

	prevSync := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	if _, err := r.SaveSyncSuccess(ctx, pBad.ID.String(), "2026-11-16T12:00:00Z", prevSync); err != nil {
		t.Fatalf("seed pBad sync: %v", err)
	}

	now := time.Date(2026, 11, 17, 2, 0, 0, 0, time.UTC)
	poller := service.NewPoller(r, service.PollerConfig{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		Tasks:  taskFeature,
		Now:    func() time.Time { return now },
		Runner: func(_ context.Context, args ...string) ([]byte, error) {
			if len(args) >= 2 && args[0] == "auth" {
				return []byte("ok"), nil
			}
			repoArg := ""
			for i := 0; i < len(args)-1; i++ {
				if args[i] == "--repo" {
					repoArg = args[i+1]
				}
			}
			if repoArg == "org/bad" && len(args) > 0 && args[0] == "issue" {
				return []byte(`{malformed-json`), nil
			}
			if repoArg == "org/good" && len(args) > 0 && args[0] == "issue" {
				return []byte(`[{"number":21,"title":"Good issue","body":"Synced","state":"OPEN","stateReason":"","labels":[],"updatedAt":"2026-11-17T01:55:00Z","url":"https://github.com/org/good/issues/21"}]`), nil
			}
			return []byte("[]"), nil
		},
	})

	poller.SyncOnce(ctx)

	badSync, err := r.GetSyncByProject(ctx, pBad.ID.String())
	if err != nil {
		t.Fatalf("GetSyncByProject pBad: %v", err)
	}
	if badSync.LastError == "" {
		t.Fatal("expected LastError recorded on malformed JSON project")
	}
	if badSync.Cursor != "2026-11-16T12:00:00Z" {
		t.Fatalf("expected pBad cursor preserved, got %q", badSync.Cursor)
	}

	goodTask, err := taskFeature.FindByExternal(ctx, task.SourceGitHub, "org/good#21")
	if err != nil {
		t.Fatalf("expected good project issue imported despite bad project failure: %v", err)
	}
	if goodTask.Title != "Good issue" {
		t.Fatalf("unexpected good task: %+v", goodTask)
	}

	goodSync, err := r.GetSyncByProject(ctx, pGood.ID.String())
	if err != nil {
		t.Fatalf("GetSyncByProject pGood: %v", err)
	}
	if goodSync.LastError != "" || goodSync.Cursor != "2026-11-17T01:55:00Z" {
		t.Fatalf("unexpected good project sync state: %+v", goodSync)
	}
}
