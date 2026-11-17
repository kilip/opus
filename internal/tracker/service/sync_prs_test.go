package service_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

	entactivity "github.com/kilip/opus/ent/activity"
	"github.com/kilip/opus/internal/platform/eventbus"
	"github.com/kilip/opus/internal/task"
	"github.com/kilip/opus/internal/tracker/domain"
	"github.com/kilip/opus/internal/tracker/provider/github"
	"github.com/kilip/opus/internal/tracker/repo"
	"github.com/kilip/opus/internal/tracker/service"
)

func TestPoller_PRSync_LinkCursorAndCrossRepoIsolation(t *testing.T) {
	client, _ := setupTrackerClient(t)
	r := repo.New(client)
	bus := eventbus.New()
	t.Cleanup(func() { _ = bus.Close(context.Background()) })
	taskFeature := task.New(client, bus)
	ctx := context.Background()

	proj := createGHProject(t, client, "opus", "kilip/opus", nil)

	// Create task for issue #104 in kilip/opus.
	linkedTask, err := taskFeature.Import(ctx, task.ImportTaskRequest{
		ProjectID:  proj.ID.String(),
		Title:      "Issue 104",
		Source:     task.SourceGitHub,
		ExternalID: "kilip/opus#104",
		Status:     task.StatusInProgress,
	})
	if err != nil {
		t.Fatalf("import task: %v", err)
	}

	prsFixture, err := os.ReadFile(filepath.Join("..", "provider", "github", "testdata", "prs.json"))
	if err != nil {
		t.Fatalf("read prs fixture: %v", err)
	}

	var prs []github.PullRequest
	if err := json.Unmarshal(prsFixture, &prs); err != nil {
		t.Fatalf("unmarshal prs fixture: %v", err)
	}
	// Add a 3rd PR whose closing reference points to a different repository's issue #104.
	prs = append(prs, github.PullRequest{
		Number:         108,
		Title:          "chore: external ref",
		State:          "OPEN",
		IsDraft:        false,
		ReviewDecision: "",
		ClosingIssuesReferences: []github.ClosingIssueRef{
			{Number: 104, URL: "https://github.com/other/repo/issues/104"},
		},
		UpdatedAt: time.Date(2026, 11, 17, 2, 5, 0, 0, time.UTC),
		URL:       "https://github.com/kilip/opus/pull/108",
	})
	prsPayload, err := json.Marshal(prs)
	if err != nil {
		t.Fatalf("marshal prs: %v", err)
	}

	var prCalls [][]string
	poller := service.NewPoller(r, service.PollerConfig{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		Tasks:  taskFeature,
		Runner: func(_ context.Context, args ...string) ([]byte, error) {
			if len(args) >= 2 && args[0] == "auth" && args[1] == "status" {
				return []byte("Logged in"), nil
			}
			if len(args) >= 2 && args[0] == "pr" && args[1] == "list" {
				prCalls = append(prCalls, slices.Clone(args))
				return prsPayload, nil
			}
			return []byte("[]"), nil
		},
	})

	poller.SyncOnce(ctx)

	if len(prCalls) != 1 {
		t.Fatalf("expected 1 pr list call, got %d", len(prCalls))
	}
	if !slices.Contains(prCalls[0], "--repo") || !slices.Contains(prCalls[0], "kilip/opus") {
		t.Fatalf("expected --repo kilip/opus in args, got %v", prCalls[0])
	}

	stored, err := r.ListPRsByProject(ctx, proj.ID.String())
	if err != nil {
		t.Fatalf("ListPRsByProject: %v", err)
	}
	if len(stored) != 3 {
		t.Fatalf("expected 3 stored PRs, got %d", len(stored))
	}

	// PR #101: no closing refs -> TaskID nil, state merged, checks none.
	if stored[0].Number != 101 || stored[0].TaskID != nil || stored[0].State != domain.PRStateMerged || stored[0].ChecksSummary != domain.ChecksNone {
		t.Fatalf("unexpected PR #101: %+v", stored[0])
	}
	// PR #105: closing ref kilip/opus#104 -> linkedTask.ID, draft, pending checks.
	if stored[1].Number != 105 || stored[1].TaskID == nil || *stored[1].TaskID != linkedTask.ID {
		t.Fatalf("expected PR #105 linked to %s, got %+v", linkedTask.ID, stored[1])
	}
	if !stored[1].IsDraft || stored[1].ChecksSummary != domain.ChecksPending || stored[1].ReviewState != "REVIEW_REQUIRED" {
		t.Fatalf("unexpected PR #105 snapshot: %+v", stored[1])
	}
	// PR #108: closing ref to other/repo#104 -> TaskID MUST remain nil.
	if stored[2].Number != 108 || stored[2].TaskID != nil {
		t.Fatalf("expected PR #108 TaskID nil for cross-repo ref, got %+v", stored[2])
	}

	// Second cycle passes cursor in --search.
	poller.SyncOnce(ctx)
	if len(prCalls) != 2 {
		t.Fatalf("expected 2 pr list calls, got %d", len(prCalls))
	}
	if !slices.Contains(prCalls[1], "--search") || !slices.Contains(prCalls[1], "updated:>=2026-11-17T02:05:00Z") {
		t.Fatalf("expected cursor --search on 2nd cycle, got %v", prCalls[1])
	}
}

func TestPoller_PRSync_ActivitiesStatusTransitionAndIdempotency(t *testing.T) {
	client, _ := setupTrackerClient(t)
	r := repo.New(client)
	bus := eventbus.New()
	t.Cleanup(func() { _ = bus.Close(context.Background()) })

	var mu sync.Mutex
	var statusEvents []eventbus.Event
	if err := bus.Subscribe("test-status-changed", func(_ context.Context, evt eventbus.Event) error {
		mu.Lock()
		defer mu.Unlock()
		statusEvents = append(statusEvents, evt)
		return nil
	}, task.EventStatusChanged); err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	taskFeature := task.New(client, bus)
	ctx := context.Background()

	proj := createGHProject(t, client, "opus", "kilip/opus", nil)
	linkedTask, err := taskFeature.Import(ctx, task.ImportTaskRequest{
		ProjectID:  proj.ID.String(),
		Title:      "Issue 104",
		Source:     task.SourceGitHub,
		ExternalID: "kilip/opus#104",
		Status:     task.StatusInProgress,
	})
	if err != nil {
		t.Fatalf("import task: %v", err)
	}

	currentPRs := []github.PullRequest{
		{
			Number:         105,
			Title:          "feat: pr",
			State:          "OPEN",
			IsDraft:        true,
			ReviewDecision: "REVIEW_REQUIRED",
			StatusCheckRollup: []github.StatusCheck{
				{TypeName: "CheckRun", Name: "test", Status: "IN_PROGRESS"},
			},
			ClosingIssuesReferences: []github.ClosingIssueRef{
				{Number: 104, URL: "https://github.com/kilip/opus/issues/104"},
			},
			UpdatedAt: time.Date(2026, 11, 17, 2, 0, 0, 0, time.UTC),
			URL:       "https://github.com/kilip/opus/pull/105",
		},
	}

	poller := service.NewPoller(r, service.PollerConfig{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		Tasks:  taskFeature,
		Runner: func(_ context.Context, args ...string) ([]byte, error) {
			if len(args) >= 2 && args[0] == "auth" && args[1] == "status" {
				return []byte("Logged in"), nil
			}
			if len(args) >= 2 && args[0] == "pr" && args[1] == "list" {
				return json.Marshal(currentPRs)
			}
			return []byte("[]"), nil
		},
	})

	// Cycle 1: Draft open PR -> records 1 activity authored "github", task stays in-progress.
	poller.SyncOnce(ctx)

	gotTask, err := taskFeature.Get(ctx, linkedTask.ID)
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if gotTask.Status != task.StatusInProgress {
		t.Fatalf("expected draft PR to keep task in-progress, got %s", gotTask.Status)
	}

	activities, err := client.Activity.Query().Where(entactivity.AuthorEQ("github")).Order(entactivity.ByCreatedAt()).All(ctx)
	if err != nil {
		t.Fatalf("query activities: %v", err)
	}
	if len(activities) != 1 || activities[0].Author == nil || *activities[0].Author != "github" {
		t.Fatalf("expected 1 activity authored github, got %+v", activities)
	}

	// Cycle 2 (AC6): No remote change -> no writes, no new activities, no events.
	prBefore, err := r.GetPR(ctx, proj.ID.String(), 105)
	if err != nil || prBefore == nil {
		t.Fatalf("GetPR before cycle 2: %v", err)
	}
	poller.SyncOnce(ctx)

	prAfter, err := r.GetPR(ctx, proj.ID.String(), 105)
	if err != nil || prAfter == nil {
		t.Fatalf("GetPR after cycle 2: %v", err)
	}
	if !prAfter.UpdatedAt.Equal(prBefore.UpdatedAt) {
		t.Fatalf("expected no PR write on unchanged cycle 2: before=%v after=%v", prBefore.UpdatedAt, prAfter.UpdatedAt)
	}
	activities, _ = client.Activity.Query().Where(entactivity.AuthorEQ("github")).All(ctx)
	if len(activities) != 1 {
		t.Fatalf("expected still 1 activity after unchanged cycle 2, got %d", len(activities))
	}
	mu.Lock()
	evCount := len(statusEvents)
	mu.Unlock()
	if evCount != 0 {
		t.Fatalf("expected 0 status events so far, got %d", evCount)
	}

	// Cycle 3 (AC3 & AC4): PR becomes ready for review (isDraft=false) and checks pass ->
	// records activity authored "github", moves task to review, publishes status_changed event.
	currentPRs[0].IsDraft = false
	currentPRs[0].ReviewDecision = "APPROVED"
	currentPRs[0].StatusCheckRollup = []github.StatusCheck{
		{TypeName: "CheckRun", Name: "test", Status: "COMPLETED", Conclusion: "SUCCESS"},
	}
	currentPRs[0].UpdatedAt = time.Date(2026, 11, 17, 2, 10, 0, 0, time.UTC)

	poller.SyncOnce(ctx)

	gotTask, _ = taskFeature.Get(ctx, linkedTask.ID)
	if gotTask.Status != task.StatusReview {
		t.Fatalf("expected non-draft open PR to move task to review, got %s", gotTask.Status)
	}
	activities, _ = client.Activity.Query().Where(entactivity.AuthorEQ("github")).All(ctx)
	if len(activities) != 2 {
		t.Fatalf("expected 2 activities after ready-for-review, got %d", len(activities))
	}
	mu.Lock()
	evCount = len(statusEvents)
	mu.Unlock()
	if evCount != 1 {
		t.Fatalf("expected 1 status_changed event, got %d", evCount)
	}

	// Cycle 4 (AC4): PR merged -> updates PR record + activity, task stays review (not done).
	currentPRs[0].State = "MERGED"
	currentPRs[0].UpdatedAt = time.Date(2026, 11, 17, 2, 20, 0, 0, time.UTC)

	poller.SyncOnce(ctx)

	gotTask, _ = taskFeature.Get(ctx, linkedTask.ID)
	if gotTask.Status != task.StatusReview {
		t.Fatalf("expected merged PR not to move task to done, got %s", gotTask.Status)
	}
	prMerged, _ := r.GetPR(ctx, proj.ID.String(), 105)
	if prMerged.State != domain.PRStateMerged {
		t.Fatalf("expected PR state merged, got %s", prMerged.State)
	}
	activities, _ = client.Activity.Query().Where(entactivity.AuthorEQ("github")).All(ctx)
	if len(activities) != 3 {
		t.Fatalf("expected 3 activities after merge, got %d", len(activities))
	}
}
