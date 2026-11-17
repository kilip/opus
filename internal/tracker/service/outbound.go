package service

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"strings"

	"github.com/kilip/opus/internal/platform/eventbus"
	"github.com/kilip/opus/internal/project"
	"github.com/kilip/opus/internal/task"
	"github.com/kilip/opus/internal/tracker/domain"
	"github.com/kilip/opus/internal/tracker/provider/github"
)

// Subscribe registers the task.status_changed outbound sync listener on bus.
func (p *Poller) Subscribe(bus *eventbus.Bus) error {
	if bus == nil {
		return nil
	}
	return bus.Subscribe("tracker-outbound-status", func(ctx context.Context, evt eventbus.Event) error {
		payload, err := eventbus.Decode[task.TaskStatusChangedPayload](evt)
		if err != nil {
			return err
		}
		p.syncOutboundTask(ctx, payload.ID)
		return nil
	}, task.EventStatusChanged)
}

func (p *Poller) syncOutboundTask(ctx context.Context, taskID string) {
	if p.tasks == nil || p.runner == nil {
		return
	}
	t, err := p.tasks.Get(ctx, taskID)
	if err != nil {
		if !errors.Is(err, task.ErrTaskNotFound) {
			p.logger.WarnContext(ctx, "outbound sync get task failed", "task_id", taskID, "error", err)
		}
		return
	}
	if t.Source != task.SourceGitHub || t.SourceStatus == "deleted" || t.SourceStatus == "transferred" {
		return
	}

	proj, err := p.repo.GetProject(ctx, t.ProjectID)
	if err != nil {
		p.logger.WarnContext(ctx, "outbound sync get project failed", "project_id", t.ProjectID, "error", err)
		return
	}
	if proj.TrackerProvider != "github" {
		return
	}

	global, err := p.repo.GetSettings(ctx)
	if err != nil {
		p.recordSyncError(ctx, "tracker outbound get settings failed", proj, err)
		return
	}
	effective := domain.ResolveEffectiveLabels(global, proj.TrackerLabels)

	if err := p.reconcileTaskOutbound(ctx, proj, effective, t); err != nil {
		p.recordSyncError(ctx, "tracker outbound sync failed", proj, err)
	}
}

func (p *Poller) reconcileProjectOutbound(ctx context.Context, proj project.Record) error {
	if p.tasks == nil {
		return nil
	}
	global, err := p.repo.GetSettings(ctx)
	if err != nil {
		return err
	}
	effective := domain.ResolveEffectiveLabels(global, proj.TrackerLabels)

	tasks, err := p.tasks.List(ctx, task.TaskFilter{ProjectID: proj.ID})
	if err != nil {
		return err
	}
	for _, t := range tasks {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if t.Source != task.SourceGitHub || t.SourceStatus == "deleted" || t.SourceStatus == "transferred" {
			continue
		}
		if err := p.reconcileTaskOutbound(ctx, proj, effective, t); err != nil {
			return err
		}
	}
	return nil
}

func (p *Poller) reconcileTaskOutbound(ctx context.Context, proj project.Record, effective domain.EffectiveLabelMap, t task.Record) error {
	issueRepo, issueNum, ok := parseExternalRef(t.ExternalID)
	if !ok {
		return nil
	}

	view, err := github.ViewIssue(ctx, p.runner, issueRepo, issueNum)
	if err != nil {
		if errors.Is(err, github.ErrIssueNotFound) {
			return p.markTaskGone(ctx, t, "deleted", "Issue deleted on GitHub; sync disabled")
		}
		if errors.Is(err, github.ErrIssueTransferred) {
			return p.markTaskGone(ctx, t, "transferred", "Issue transferred to another repository; sync disabled")
		}
		return err
	}

	remoteLabels := make(map[string]bool, len(view.Labels))
	for _, l := range view.Labels {
		remoteLabels[l.Name] = true
	}

	desiredStatusLabel := effective.StatusToLabel[string(t.Status)]
	localLabels := make(map[string]bool, len(t.Labels))
	for _, l := range t.Labels {
		localLabels[l] = true
	}

	var toRemove []string
	for _, lbl := range allConfiguredStatusLabels(effective) {
		if lbl == desiredStatusLabel || localLabels[lbl] {
			continue
		}
		if remoteLabels[lbl] {
			toRemove = append(toRemove, lbl)
		}
	}
	slices.Sort(toRemove)

	var toAdd []string
	if t.Status != task.StatusDone && desiredStatusLabel != "" && !remoteLabels[desiredStatusLabel] {
		toAdd = append(toAdd, desiredStatusLabel)
	}
	for _, lbl := range t.Labels {
		if lbl != "" && !remoteLabels[lbl] && !slices.Contains(toAdd, lbl) {
			toAdd = append(toAdd, lbl)
		}
	}
	slices.Sort(toAdd)

	if len(toAdd) > 0 {
		if err := p.ensureRepoLabels(ctx, issueRepo, toAdd); err != nil {
			return err
		}
	}

	remoteClosed := strings.EqualFold(view.State, "CLOSED")
	if t.Status == task.StatusDone {
		if len(toRemove) > 0 {
			if err := github.EditIssueLabels(ctx, p.runner, issueRepo, issueNum, nil, toRemove); err != nil {
				return err
			}
		}
		if !remoteClosed {
			if err := github.CloseIssue(ctx, p.runner, issueRepo, issueNum, "completed"); err != nil {
				return err
			}
		}
	} else {
		if remoteClosed {
			if err := github.ReopenIssue(ctx, p.runner, issueRepo, issueNum); err != nil {
				return err
			}
		}
		if len(toAdd) > 0 || len(toRemove) > 0 {
			if err := github.EditIssueLabels(ctx, p.runner, issueRepo, issueNum, toAdd, toRemove); err != nil {
				return err
			}
		}
	}

	return p.pushComments(ctx, issueRepo, issueNum, t.ID, view.Comments)
}

func (p *Poller) ensureRepoLabels(ctx context.Context, repo string, needed []string) error {
	existing, err := github.ListRepoLabels(ctx, p.runner, repo)
	if err != nil {
		return err
	}
	have := make(map[string]bool, len(existing))
	for _, name := range existing {
		have[name] = true
	}
	for _, name := range needed {
		if have[name] {
			continue
		}
		if err := github.CreateLabel(ctx, p.runner, repo, name); err != nil {
			return err
		}
		have[name] = true
	}
	return nil
}

func (p *Poller) pushComments(ctx context.Context, repo string, number int, taskID string, existing []github.IssueComment) error {
	activities, err := p.tasks.ListActivity(ctx, taskID)
	if err != nil {
		return err
	}
	for _, act := range activities {
		if !shouldPushActivity(act) {
			continue
		}
		marker := github.FormatActivityMarker(act.ID)
		if hasCommentWithMarker(existing, marker) {
			continue
		}
		body := act.Body + "\n\n" + marker
		if err := github.CommentIssue(ctx, p.runner, repo, number, body); err != nil {
			return err
		}
	}
	return nil
}

func (p *Poller) markTaskGone(ctx context.Context, t task.Record, sourceStatus, body string) error {
	if _, err := p.tasks.Import(ctx, task.ImportTaskRequest{
		ProjectID:    t.ProjectID,
		Title:        t.Title,
		Description:  t.Description,
		Source:       t.Source,
		ExternalID:   t.ExternalID,
		ExternalURL:  t.ExternalURL,
		SourceStatus: sourceStatus,
		Key:          t.Key,
		Labels:       t.Labels,
		Type:         t.Type,
		Priority:     t.Priority,
		Status:       t.Status,
		SyncedAt:     t.SyncedAt,
	}); err != nil {
		return err
	}
	_, err := p.tasks.AddActivity(ctx, t.ID, task.ActivityKindStatus, "github", body)
	return err
}

func shouldPushActivity(act task.Activity) bool {
	if act.Author == "github" || act.Author == "vibe" {
		return true
	}
	return act.Kind == task.ActivityKindStatus && !strings.HasPrefix(act.Body, "created task")
}

func hasCommentWithMarker(comments []github.IssueComment, marker string) bool {
	for _, c := range comments {
		if strings.Contains(c.Body, marker) {
			return true
		}
	}
	return false
}

func allConfiguredStatusLabels(effective domain.EffectiveLabelMap) []string {
	seen := make(map[string]bool)
	var out []string
	for _, lbl := range effective.StatusToLabel {
		if lbl != "" && !seen[lbl] {
			seen[lbl] = true
			out = append(out, lbl)
		}
	}
	for _, lbl := range domain.DefaultStatusLabels() {
		if lbl != "" && !seen[lbl] {
			seen[lbl] = true
			out = append(out, lbl)
		}
	}
	return out
}

func parseExternalRef(externalID string) (string, int, bool) {
	idx := strings.LastIndex(externalID, "#")
	if idx <= 0 || idx >= len(externalID)-1 {
		return "", 0, false
	}
	repo := strings.TrimSpace(externalID[:idx])
	num, err := strconv.Atoi(strings.TrimSpace(externalID[idx+1:]))
	if err != nil || num <= 0 || repo == "" {
		return "", 0, false
	}
	return repo, num, true
}
