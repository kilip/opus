package service_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os/exec"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	entactivity "github.com/kilip/opus/ent/activity"
	"github.com/kilip/opus/internal/platform/eventbus"
	"github.com/kilip/opus/internal/task"
	"github.com/kilip/opus/internal/tracker/provider/github"
	"github.com/kilip/opus/internal/tracker/repo"
	"github.com/kilip/opus/internal/tracker/service"
)

func TestOutbound_SubscriberCustomLabelsMissingLabelsCloseReopen(t *testing.T) {
	client, _ := setupTrackerClient(t)
	r := repo.New(client)
	bus := eventbus.New()
	t.Cleanup(func() { _ = bus.Close(context.Background()) })
	taskFeature := task.New(client, bus)
	ctx := context.Background()

	proj := createGHProject(t, client, "opus", "kilip/opus", map[string]string{
		"in-progress": "wip",
	})

	manualTask, err := client.Task.Create().
		SetProjectID(proj.ID).
		SetKey("OP-99").
		SetTitle("Manual task").
		SetSource(string(task.SourceManual)).
		Save(ctx)
	if err != nil {
		t.Fatalf("create manual task: %v", err)
	}

	ghTask, err := taskFeature.Import(ctx, task.ImportTaskRequest{
		ProjectID:  proj.ID.String(),
		Title:      "GH task",
		Source:     task.SourceGitHub,
		ExternalID: "kilip/opus#42",
		Status:     task.StatusTodo,
		Labels:     []string{"vibe:failed"},
	})
	if err != nil {
		t.Fatalf("import gh task: %v", err)
	}

	var mu sync.Mutex
	var calls [][]string
	remoteState := "OPEN"
	remoteLabels := []github.Label{{Name: "status:todo"}}
	repoLabels := []string{"status:todo", "status:review"}

	poller := service.NewPoller(r, service.PollerConfig{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		Tasks:  taskFeature,
		Runner: func(_ context.Context, args ...string) ([]byte, error) {
			mu.Lock()
			defer mu.Unlock()
			calls = append(calls, slices.Clone(args))
			if len(args) >= 2 && args[0] == "issue" && args[1] == "view" {
				return json.Marshal(github.IssueView{
					Number: 42,
					State:  remoteState,
					Labels: remoteLabels,
					URL:    "https://github.com/kilip/opus/issues/42",
				})
			}
			if len(args) >= 2 && args[0] == "label" && args[1] == "list" {
				var out []github.Label
				for _, l := range repoLabels {
					out = append(out, github.Label{Name: l})
				}
				return json.Marshal(out)
			}
			if len(args) >= 3 && args[0] == "label" && args[1] == "create" {
				repoLabels = append(repoLabels, args[2])
				return []byte(""), nil
			}
			if len(args) >= 2 && args[0] == "issue" && args[1] == "edit" {
				for i := 0; i < len(args)-1; i++ {
					if args[i] == "--add-label" {
						remoteLabels = append(remoteLabels, github.Label{Name: args[i+1]})
					}
					if args[i] == "--remove-label" {
						rm := args[i+1]
						remoteLabels = slices.DeleteFunc(remoteLabels, func(l github.Label) bool { return l.Name == rm })
					}
				}
				return []byte(""), nil
			}
			if len(args) >= 2 && args[0] == "issue" && args[1] == "close" {
				remoteState = "CLOSED"
				return []byte(""), nil
			}
			if len(args) >= 2 && args[0] == "issue" && args[1] == "reopen" {
				remoteState = "OPEN"
				return []byte(""), nil
			}
			return []byte("[]"), nil
		},
	})
	if err := poller.Subscribe(bus); err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	// Manual task status change -> no gh calls.
	inProg := task.StatusInProgress
	if _, err := taskFeature.Update(ctx, manualTask.ID.String(), task.UpdateTaskRequest{Status: &inProg}); err != nil {
		t.Fatalf("update manual task: %v", err)
	}
	time.Sleep(20 * time.Millisecond)
	mu.Lock()
	manualCalls := len(calls)
	mu.Unlock()
	if manualCalls != 0 {
		t.Fatalf("expected 0 gh calls for manual task, got %d", manualCalls)
	}

	// GitHub task todo -> in-progress (custom label "wip" + ordinary "vibe:failed").
	if _, err := taskFeature.Update(ctx, ghTask.ID, task.UpdateTaskRequest{Status: &inProg}); err != nil {
		t.Fatalf("update gh task: %v", err)
	}

	waitUntil(t, "custom label wip and vibe:failed applied", func() bool {
		mu.Lock()
		defer mu.Unlock()
		return slices.Contains(repoLabels, "wip") &&
			slices.Contains(repoLabels, "vibe:failed") &&
			slices.ContainsFunc(remoteLabels, func(l github.Label) bool { return l.Name == "wip" }) &&
			!slices.ContainsFunc(remoteLabels, func(l github.Label) bool { return l.Name == "status:todo" }) &&
			slices.ContainsFunc(remoteLabels, func(l github.Label) bool { return l.Name == "vibe:failed" })
	})

	// Move to done -> removes "wip" and closes issue with reason completed.
	done := task.StatusDone
	if _, err := taskFeature.Update(ctx, ghTask.ID, task.UpdateTaskRequest{Status: &done}); err != nil {
		t.Fatalf("update gh task to done: %v", err)
	}

	waitUntil(t, "issue closed with reason completed and wip removed", func() bool {
		mu.Lock()
		defer mu.Unlock()
		if remoteState != "CLOSED" {
			return false
		}
		if slices.ContainsFunc(remoteLabels, func(l github.Label) bool { return l.Name == "wip" }) {
			return false
		}
		for _, c := range calls {
			if len(c) >= 2 && c[0] == "issue" && c[1] == "close" && slices.Contains(c, "completed") {
				return true
			}
		}
		return false
	})

	// Move out of done -> review: reopens issue and adds status:review.
	review := task.StatusReview
	if _, err := taskFeature.Update(ctx, ghTask.ID, task.UpdateTaskRequest{Status: &review}); err != nil {
		t.Fatalf("update gh task to review: %v", err)
	}

	waitUntil(t, "issue reopened with status:review", func() bool {
		mu.Lock()
		defer mu.Unlock()
		return remoteState == "OPEN" &&
			slices.ContainsFunc(remoteLabels, func(l github.Label) bool { return l.Name == "status:review" })
	})
}

func waitUntil(t *testing.T, desc string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for: %s", desc)
}

func TestOutbound_CommentsMarkerDeduplicationAndIdempotency(t *testing.T) {
	client, _ := setupTrackerClient(t)
	r := repo.New(client)
	bus := eventbus.New()
	t.Cleanup(func() { _ = bus.Close(context.Background()) })
	taskFeature := task.New(client, bus)
	ctx := context.Background()

	proj := createGHProject(t, client, "opus", "kilip/opus", nil)
	ghTask, err := taskFeature.Import(ctx, task.ImportTaskRequest{
		ProjectID:  proj.ID.String(),
		Title:      "GH task",
		Source:     task.SourceGitHub,
		ExternalID: "kilip/opus#43",
		Status:     task.StatusInProgress,
	})
	if err != nil {
		t.Fatalf("import gh task: %v", err)
	}

	if _, err := taskFeature.AddActivity(ctx, ghTask.ID, task.ActivityKindComment, "vibe", "Agent started"); err != nil {
		t.Fatalf("add vibe activity: %v", err)
	}
	if _, err := taskFeature.AddActivity(ctx, ghTask.ID, task.ActivityKindReview, "github", "PR #10 opened"); err != nil {
		t.Fatalf("add github activity: %v", err)
	}
	if _, err := taskFeature.AddActivity(ctx, ghTask.ID, task.ActivityKindComment, "alice", "Human note"); err != nil {
		t.Fatalf("add human activity: %v", err)
	}

	var mu sync.Mutex
	var remoteComments []github.IssueComment
	var writeCalls [][]string

	poller := service.NewPoller(r, service.PollerConfig{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		Tasks:  taskFeature,
		Runner: func(_ context.Context, args ...string) ([]byte, error) {
			mu.Lock()
			defer mu.Unlock()
			if len(args) >= 2 && args[0] == "auth" && args[1] == "status" {
				return []byte("ok"), nil
			}
			if len(args) >= 2 && args[0] == "issue" && args[1] == "view" {
				return json.Marshal(github.IssueView{
					Number:   43,
					State:    "OPEN",
					Labels:   []github.Label{{Name: "status:in-progress"}},
					URL:      "https://github.com/kilip/opus/issues/43",
					Comments: remoteComments,
				})
			}
			if len(args) >= 2 && args[0] == "issue" && args[1] == "comment" {
				writeCalls = append(writeCalls, slices.Clone(args))
				for i := 0; i < len(args)-1; i++ {
					if args[i] == "--body" {
						remoteComments = append(remoteComments, github.IssueComment{Body: args[i+1]})
					}
				}
				return []byte(""), nil
			}
			if len(args) >= 2 && args[0] == "issue" && (args[1] == "edit" || args[1] == "close" || args[1] == "reopen") {
				writeCalls = append(writeCalls, slices.Clone(args))
				return []byte(""), nil
			}
			if len(args) >= 2 && args[0] == "label" && args[1] == "create" {
				writeCalls = append(writeCalls, slices.Clone(args))
				return []byte(""), nil
			}
			return []byte("[]"), nil
		},
	})

	// Cycle 1: posts 2 comments (vibe + github), skips human ("alice") and initial created task activity.
	poller.SyncOnce(ctx)

	mu.Lock()
	if len(remoteComments) != 2 {
		t.Fatalf("expected 2 pushed comments, got %d: %+v", len(remoteComments), remoteComments)
	}
	for _, c := range remoteComments {
		if !github.HasOpusActivityMarker(c.Body) {
			t.Fatalf("expected opus activity marker in comment: %q", c.Body)
		}
		if strings.Contains(c.Body, "Human note") {
			t.Fatalf("human comment should not be pushed: %q", c.Body)
		}
	}
	writeCalls = nil
	mu.Unlock()

	// Cycle 2 (AC4 idempotency): remote state & comments match desired -> 0 gh writes.
	poller.SyncOnce(ctx)
	mu.Lock()
	if len(writeCalls) != 0 {
		t.Fatalf("expected 0 gh write calls on idempotent cycle, got %v", writeCalls)
	}
	mu.Unlock()
}

func TestOutbound_InboundEchoNoWriteOrEvent(t *testing.T) {
	client, _ := setupTrackerClient(t)
	r := repo.New(client)
	bus := eventbus.New()
	t.Cleanup(func() { _ = bus.Close(context.Background()) })
	taskFeature := task.New(client, bus)
	ctx := context.Background()

	proj := createGHProject(t, client, "opus", "kilip/opus", nil)
	now := time.Date(2026, 11, 17, 12, 0, 0, 0, time.UTC)

	ghTask, err := taskFeature.Import(ctx, task.ImportTaskRequest{
		ProjectID:    proj.ID.String(),
		Title:        "Echo test",
		Description:  "Body",
		Source:       task.SourceGitHub,
		ExternalID:   "kilip/opus#44",
		ExternalURL:  "https://github.com/kilip/opus/issues/44",
		SourceStatus: "open",
		Status:       task.StatusReview,
		SyncedAt:     &now,
	})
	if err != nil {
		t.Fatalf("import task: %v", err)
	}

	var statusEvents int
	if err := bus.Subscribe("test-echo", func(_ context.Context, _ eventbus.Event) error {
		statusEvents++
		return nil
	}, task.EventStatusChanged); err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	echoIssues, _ := json.Marshal([]github.Issue{
		{
			Number:    44,
			Title:     "Echo test",
			Body:      "Body",
			State:     "OPEN",
			Labels:    []github.Label{{Name: "status:review"}},
			UpdatedAt: now.Add(time.Minute),
			URL:       "https://github.com/kilip/opus/issues/44",
		},
	})

	poller := service.NewPoller(r, service.PollerConfig{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		Tasks:  taskFeature,
		Runner: func(_ context.Context, args ...string) ([]byte, error) {
			if len(args) >= 2 && args[0] == "auth" {
				return []byte("ok"), nil
			}
			if len(args) >= 2 && args[0] == "issue" && args[1] == "list" {
				return echoIssues, nil
			}
			if len(args) >= 2 && args[0] == "issue" && args[1] == "view" {
				return json.Marshal(github.IssueView{
					Number: 44,
					State:  "OPEN",
					Labels: []github.Label{{Name: "status:review"}},
					URL:    "https://github.com/kilip/opus/issues/44",
				})
			}
			return []byte("[]"), nil
		},
	})

	poller.SyncOnce(ctx)

	got, _ := taskFeature.Get(ctx, ghTask.ID)
	if got.Status != task.StatusReview {
		t.Fatalf("expected task status unchanged, got %s", got.Status)
	}
	if statusEvents != 0 {
		t.Fatalf("expected 0 status_changed events on echo, got %d", statusEvents)
	}
}

func TestOutbound_ReconcileHealsDroppedEventAndRetriesWriteFailure(t *testing.T) {
	client, _ := setupTrackerClient(t)
	r := repo.New(client)
	bus := eventbus.New()
	t.Cleanup(func() { _ = bus.Close(context.Background()) })
	taskFeature := task.New(client, bus)
	ctx := context.Background()

	proj := createGHProject(t, client, "opus", "kilip/opus", nil)
	// Import task at status=review while remote still has status:todo (simulating dropped event).
	if _, err := taskFeature.Import(ctx, task.ImportTaskRequest{
		ProjectID:  proj.ID.String(),
		Title:      "Self heal",
		Source:     task.SourceGitHub,
		ExternalID: "kilip/opus#45",
		Status:     task.StatusReview,
	}); err != nil {
		t.Fatalf("import task: %v", err)
	}

	failEdit := true
	remoteLabels := []github.Label{{Name: "status:todo"}}

	poller := service.NewPoller(r, service.PollerConfig{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		Tasks:  taskFeature,
		Runner: func(_ context.Context, args ...string) ([]byte, error) {
			if len(args) >= 2 && args[0] == "auth" {
				return []byte("ok"), nil
			}
			if len(args) >= 2 && args[0] == "issue" && args[1] == "view" {
				return json.Marshal(github.IssueView{
					Number: 45,
					State:  "OPEN",
					Labels: remoteLabels,
					URL:    "https://github.com/kilip/opus/issues/45",
				})
			}
			if len(args) >= 2 && args[0] == "label" && args[1] == "list" {
				return json.Marshal([]github.Label{{Name: "status:todo"}, {Name: "status:review"}})
			}
			if len(args) >= 2 && args[0] == "issue" && args[1] == "edit" {
				if failEdit {
					return nil, errors.New("gh temporary failure")
				}
				remoteLabels = []github.Label{{Name: "status:review"}}
				return []byte(""), nil
			}
			return []byte("[]"), nil
		},
	})

	// Cycle 1: gh edit fails -> recorded in last_error.
	poller.SyncOnce(ctx)
	st, err := r.GetSyncByProject(ctx, proj.ID.String())
	if err != nil {
		t.Fatalf("GetSyncByProject: %v", err)
	}
	if !strings.Contains(st.LastError, "gh temporary failure") {
		t.Fatalf("expected LastError recorded, got %q", st.LastError)
	}

	// Cycle 2: gh recovers -> self-heals remote label & clears LastError.
	failEdit = false
	poller.SyncOnce(ctx)
	st, err = r.GetSyncByProject(ctx, proj.ID.String())
	if err != nil {
		t.Fatalf("GetSyncByProject: %v", err)
	}
	if st.LastError != "" {
		t.Fatalf("expected LastError cleared, got %q", st.LastError)
	}
	if len(remoteLabels) != 1 || remoteLabels[0].Name != "status:review" {
		t.Fatalf("expected remote healed to status:review, got %+v", remoteLabels)
	}
}

func TestOutbound_DeletedAndTransferredIssues(t *testing.T) {
	client, _ := setupTrackerClient(t)
	r := repo.New(client)
	bus := eventbus.New()
	t.Cleanup(func() { _ = bus.Close(context.Background()) })
	taskFeature := task.New(client, bus)
	ctx := context.Background()

	proj := createGHProject(t, client, "opus", "kilip/opus", nil)
	delTask, err := taskFeature.Import(ctx, task.ImportTaskRequest{
		ProjectID:  proj.ID.String(),
		Title:      "Deleted issue",
		Source:     task.SourceGitHub,
		ExternalID: "kilip/opus#46",
		Status:     task.StatusTodo,
	})
	if err != nil {
		t.Fatalf("import deleted task: %v", err)
	}
	xferTask, err := taskFeature.Import(ctx, task.ImportTaskRequest{
		ProjectID:  proj.ID.String(),
		Title:      "Transferred issue",
		Source:     task.SourceGitHub,
		ExternalID: "kilip/opus#47",
		Status:     task.StatusTodo,
	})
	if err != nil {
		t.Fatalf("import transferred task: %v", err)
	}

	var viewCalls int
	poller := service.NewPoller(r, service.PollerConfig{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		Tasks:  taskFeature,
		Runner: func(_ context.Context, args ...string) ([]byte, error) {
			if len(args) >= 2 && args[0] == "auth" {
				return []byte("ok"), nil
			}
			if len(args) >= 3 && args[0] == "issue" && args[1] == "view" {
				viewCalls++
				if args[2] == "46" {
					return nil, errors.New("gh issue view: exit status 1: Could not resolve to an Issue")
				}
				return json.Marshal(github.IssueView{
					Number: 47,
					State:  "OPEN",
					URL:    "https://github.com/other/repo/issues/99",
				})
			}
			return []byte("[]"), nil
		},
	})

	poller.SyncOnce(ctx)

	gotDel, err := taskFeature.Get(ctx, delTask.ID)
	if err != nil {
		t.Fatalf("deleted task must be kept: %v", err)
	}
	if gotDel.SourceStatus != "deleted" {
		t.Fatalf("expected SourceStatus=deleted, got %q", gotDel.SourceStatus)
	}

	gotXfer, err := taskFeature.Get(ctx, xferTask.ID)
	if err != nil {
		t.Fatalf("transferred task must be kept: %v", err)
	}
	if gotXfer.SourceStatus != "transferred" {
		t.Fatalf("expected SourceStatus=transferred, got %q", gotXfer.SourceStatus)
	}

	acts, err := client.Activity.Query().Where(entactivity.AuthorEQ("github")).All(ctx)
	if err != nil {
		t.Fatalf("query activities: %v", err)
	}
	if len(acts) != 2 {
		t.Fatalf("expected 2 github activities for deleted+transferred, got %d", len(acts))
	}

	// Second cycle: no further outbound sync calls for deleted/transferred tasks.
	viewCalls = 0
	poller.SyncOnce(ctx)
	if viewCalls != 0 {
		t.Fatalf("expected 0 view calls on 2nd cycle for deleted/transferred tasks, got %d", viewCalls)
	}
}

func TestTracker_NoInternalRunImport(t *testing.T) {
	out, err := exec.Command("go", "list", "-deps", "github.com/kilip/opus/internal/tracker/...").CombinedOutput()
	if err != nil {
		t.Fatalf("go list -deps failed: %v\n%s", err, out)
	}
	if strings.Contains(string(out), "github.com/kilip/opus/internal/run") {
		t.Fatalf("internal/tracker must not import internal/run")
	}
}
