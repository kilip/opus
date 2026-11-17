package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/kilip/opus/internal/project"
	"github.com/kilip/opus/internal/task"
	"github.com/kilip/opus/internal/tracker/domain"
	"github.com/kilip/opus/internal/tracker/provider/github"
)

type mappedIssue struct {
	status         task.TaskStatus
	hasStatusLabel bool
	taskType       task.TaskType
	priority       task.TaskPriority
	labels         []string
}

func (p *Poller) syncIssues(ctx context.Context, proj project.Record, issues []github.Issue, now time.Time) error {
	global, err := p.repo.GetSettings(ctx)
	if err != nil {
		return err
	}
	effective := domain.ResolveEffectiveLabels(global, proj.TrackerLabels)

	for _, iss := range issues {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err := p.syncIssue(ctx, proj, effective, iss, now); err != nil {
			return err
		}
	}
	return nil
}

func (p *Poller) syncIssue(ctx context.Context, proj project.Record, effective domain.EffectiveLabelMap, iss github.Issue, now time.Time) error {
	externalID := fmt.Sprintf("%s#%d", proj.TrackerRef, iss.Number)
	existing, err := p.tasks.FindByExternal(ctx, task.SourceGitHub, externalID)
	if err != nil && !errors.Is(err, task.ErrTaskNotFound) {
		return err
	}

	if existing != nil && existing.SourceStatus == "deleted" {
		return nil
	}

	closed := strings.EqualFold(iss.State, "CLOSED")
	if existing == nil && closed {
		return nil
	}

	mapped := mapIssueLabels(effective, iss.Labels)

	remoteStatus := task.StatusTodo
	if closed {
		remoteStatus = task.StatusDone
	} else if mapped.hasStatusLabel {
		remoteStatus = mapped.status
	}

	req := task.ImportTaskRequest{
		ProjectID:    proj.ID,
		Title:        iss.Title,
		Description:  iss.Body,
		Source:       task.SourceGitHub,
		ExternalID:   externalID,
		ExternalURL:  iss.URL,
		SourceStatus: strings.ToLower(iss.State),
		Status:       remoteStatus,
		Priority:     mapped.priority,
		Type:         mapped.taskType,
		Labels:       mapped.labels,
		SyncedAt:     &now,
	}

	// Preserve local status/labels when local edits are newer than the last sync; outbound reconcile overwrites remote.
	if existing != nil && existing.SyncedAt != nil && existing.UpdatedAt.After(*existing.SyncedAt) {
		req.Status = existing.Status
		req.Priority = existing.Priority
		req.Type = existing.Type
		req.Labels = existing.Labels
		req.SyncedAt = nil
	}

	_, err = p.tasks.Import(ctx, req)
	return err
}

func mapIssueLabels(effective domain.EffectiveLabelMap, labels []github.Label) mappedIssue {
	out := mappedIssue{
		taskType: task.TypeFeat,
		priority: task.PriorityP1,
		labels:   make([]string, 0, len(labels)),
	}
	for _, lbl := range labels {
		if st, ok := effective.LabelToStatus[lbl.Name]; ok {
			if !out.hasStatusLabel || statusPrecedence(st) > statusPrecedence(out.status) {
				out.status = st
				out.hasStatusLabel = true
			}
			continue
		}
		if tp, ok := parseTypeLabel(lbl.Name); ok {
			out.taskType = tp
			continue
		}
		if pr, ok := parsePriorityLabel(lbl.Name); ok {
			out.priority = pr
			continue
		}
		out.labels = append(out.labels, lbl.Name)
	}
	return out
}

func statusPrecedence(s task.TaskStatus) int {
	switch s {
	case task.StatusBlocked:
		return 5
	case task.StatusReview:
		return 4
	case task.StatusInProgress:
		return 3
	case task.StatusTodo:
		return 2
	case task.StatusBacklog:
		return 1
	default:
		return 0
	}
}

func parseTypeLabel(name string) (task.TaskType, bool) {
	raw, ok := strings.CutPrefix(strings.ToLower(strings.TrimSpace(name)), "type:")
	if !ok {
		return "", false
	}
	switch task.TaskType(raw) {
	case task.TypeFeat, task.TypeBug, task.TypeChore, task.TypeRefactor, task.TypeDocs:
		return task.TaskType(raw), true
	default:
		return "", false
	}
}

func parsePriorityLabel(name string) (task.TaskPriority, bool) {
	raw, ok := strings.CutPrefix(strings.ToLower(strings.TrimSpace(name)), "priority:")
	if !ok {
		return "", false
	}
	switch raw {
	case "0", "p0":
		return task.PriorityP0, true
	case "1", "p1":
		return task.PriorityP1, true
	case "2", "p2":
		return task.PriorityP2, true
	case "3", "p3":
		return task.PriorityP3, true
	default:
		return "", false
	}
}
