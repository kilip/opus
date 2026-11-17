package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/kilip/opus/internal/project"
	"github.com/kilip/opus/internal/task"
	"github.com/kilip/opus/internal/tracker/domain"
	"github.com/kilip/opus/internal/tracker/provider/github"
	"github.com/kilip/opus/internal/tracker/repo"
)

func (p *Poller) syncPRs(ctx context.Context, proj project.Record, prs []github.PullRequest) error {
	for _, pr := range prs {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err := p.syncPR(ctx, proj, pr); err != nil {
			return err
		}
	}
	return nil
}

func (p *Poller) syncPR(ctx context.Context, proj project.Record, pr github.PullRequest) error {
	linkedTask, err := p.resolveLinkedTask(ctx, proj, pr.ClosingIssuesReferences)
	if err != nil {
		return err
	}

	var taskID *string
	if linkedTask != nil {
		id := linkedTask.ID
		taskID = &id
	}

	state := domain.NormalizePRState(pr.State)
	isDraft := pr.IsDraft
	reviewState := strings.TrimSpace(pr.ReviewDecision)
	checksSummary := github.DeriveChecksSummary(pr.StatusCheckRollup)

	existing, err := p.repo.GetPR(ctx, proj.ID, pr.Number)
	if err != nil {
		return err
	}

	if existing != nil &&
		existing.State == state &&
		existing.IsDraft == isDraft &&
		existing.ReviewState == reviewState &&
		existing.ChecksSummary == checksSummary &&
		existing.URL == pr.URL &&
		sameStringPtr(existing.TaskID, taskID) {
		return nil
	}

	if _, err := p.repo.UpsertPR(ctx, repo.UpsertPRInput{
		ProjectID:     proj.ID,
		Number:        pr.Number,
		TaskID:        taskID,
		State:         state,
		IsDraft:       isDraft,
		ReviewState:   reviewState,
		ChecksSummary: checksSummary,
		URL:           pr.URL,
	}); err != nil {
		return err
	}

	snapshotChanged := existing == nil ||
		existing.State != state ||
		existing.IsDraft != isDraft ||
		existing.ReviewState != reviewState ||
		existing.ChecksSummary != checksSummary

	if snapshotChanged && linkedTask != nil && p.tasks != nil {
		body := formatPRActivity(pr.Number, existing, state, isDraft, reviewState, checksSummary)
		if _, err := p.tasks.AddActivity(ctx, linkedTask.ID, task.ActivityKindReview, "github", body); err != nil {
			return err
		}
	}

	if state == domain.PRStateOpen && !isDraft && linkedTask != nil && p.tasks != nil && isBelowReview(linkedTask.Status) {
		review := task.StatusReview
		if _, err := p.tasks.Update(ctx, linkedTask.ID, task.UpdateTaskRequest{Status: &review}); err != nil {
			return err
		}
	}

	return nil
}

func (p *Poller) resolveLinkedTask(ctx context.Context, proj project.Record, refs []github.ClosingIssueRef) (*task.Record, error) {
	if p.tasks == nil {
		return nil, nil
	}
	issueNum, ok := firstSameRepoIssueNumber(proj.TrackerRef, refs)
	if !ok {
		return nil, nil
	}
	externalID := fmt.Sprintf("%s#%d", proj.TrackerRef, issueNum)
	found, err := p.tasks.FindByExternal(ctx, task.SourceGitHub, externalID)
	if err != nil {
		if errors.Is(err, task.ErrTaskNotFound) {
			return nil, nil
		}
		return nil, err
	}
	if found == nil || found.ProjectID != proj.ID {
		return nil, nil
	}
	return found, nil
}

func firstSameRepoIssueNumber(trackerRef string, refs []github.ClosingIssueRef) (int, bool) {
	repoPath := "/" + strings.ToLower(strings.TrimSpace(trackerRef)) + "/issues/"
	for _, ref := range refs {
		if ref.Number <= 0 {
			continue
		}
		if ref.URL != "" && !strings.Contains(strings.ToLower(ref.URL), repoPath) {
			continue
		}
		return ref.Number, true
	}
	return 0, false
}

func isBelowReview(s task.TaskStatus) bool {
	return s == task.StatusBacklog || s == task.StatusTodo || s == task.StatusInProgress
}

func sameStringPtr(a, b *string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func formatPRActivity(
	number int,
	prev *domain.TrackerPR,
	state domain.PRState,
	isDraft bool,
	reviewState string,
	checksSummary domain.ChecksSummary,
) string {
	if prev == nil {
		draftLabel := ""
		if isDraft {
			draftLabel = " (draft)"
		}
		return fmt.Sprintf("PR #%d %s%s (checks: %s)", number, state, draftLabel, checksSummary)
	}
	var changes []string
	if prev.State != state {
		changes = append(changes, fmt.Sprintf("state %s -> %s", prev.State, state))
	}
	if prev.IsDraft != isDraft {
		changes = append(changes, fmt.Sprintf("draft %t -> %t", prev.IsDraft, isDraft))
	}
	if prev.ReviewState != reviewState {
		changes = append(changes, fmt.Sprintf("review %s -> %s", emptyNone(prev.ReviewState), emptyNone(reviewState)))
	}
	if prev.ChecksSummary != checksSummary {
		changes = append(changes, fmt.Sprintf("checks %s -> %s", prev.ChecksSummary, checksSummary))
	}
	return fmt.Sprintf("PR #%d updated: %s", number, strings.Join(changes, ", "))
}

func emptyNone(s string) string {
	if s == "" {
		return "none"
	}
	return s
}
