package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/kilip/opus/internal/agent/domain"
	"github.com/kilip/opus/internal/run"
	"github.com/kilip/opus/internal/session"
	"github.com/kilip/opus/internal/task"
)

const (
	authorVibe  = "vibe"
	authorHuman = "human"

	eventKindToolCall = "tool_call"
)

// TaskStore is the subset of the task root facade used by the agent service.
type TaskStore interface {
	Get(ctx context.Context, id string) (task.Record, error)
	GetByKey(ctx context.Context, key string) (task.Record, error)
	List(ctx context.Context, filters ...task.TaskFilter) ([]task.Record, error)
	Update(ctx context.Context, id string, req task.UpdateTaskRequest) (task.Record, error)
	ListActivity(ctx context.Context, id string) ([]task.Activity, error)
	AddActivity(ctx context.Context, id string, kind task.ActivityKind, author, body string) (task.Activity, error)
}

// RunStore is the subset of the run root facade used by the agent service.
type RunStore interface {
	Get(ctx context.Context, id uuid.UUID) (run.Run, error)
	SetReport(ctx context.Context, id uuid.UUID, report json.RawMessage) (run.Run, error)
}

// SessionStore is the subset of the session root facade used by the agent service.
type SessionStore interface {
	RecordEvent(ctx context.Context, sessionID uuid.UUID, kind string, payload json.RawMessage) (session.SessionEvent, error)
}

// Config configures the agent tool service.
type Config struct {
	Tasks    TaskStore
	Runs     RunStore
	Sessions SessionStore
	Now      func() time.Time
}

// Service implements MCP tool business logic, scoping, sanitization, rate limiting, and audit events.
type Service struct {
	tasks    TaskStore
	runs     RunStore
	sessions SessionStore
	now      func() time.Time

	mu      sync.Mutex
	buckets map[string][]time.Time
}

// New constructs a new agent tool Service.
func New(cfg Config) *Service {
	now := cfg.Now
	if now == nil {
		now = time.Now
	}
	return &Service{
		tasks:    cfg.Tasks,
		runs:     cfg.Runs,
		sessions: cfg.Sessions,
		now:      now,
		buckets:  make(map[string][]time.Time),
	}
}

// GetTask resolves a task by Ref (OP-<n>, #<issue>, UUID, or bound run task) within caller scope,
// truncates >8KB descriptions, wraps untrusted fields, and marks human comments.
func (s *Service) GetTask(ctx context.Context, caller domain.CallerContext, in domain.GetTaskInput) (domain.GetTaskOutput, error) {
	var out domain.GetTaskOutput
	err := s.execute(ctx, caller, "get_task", map[string]any{"ref": in.Ref}, func() error {
		t, err := s.resolveTask(ctx, caller, in.Ref)
		if err != nil {
			return err
		}
		activities, err := s.tasks.ListActivity(ctx, t.ID)
		if err != nil {
			return err
		}

		desc, truncated := truncateUTF8(t.Description, domain.MaxDescriptionBytes)
		items := make([]domain.TaskActivityItem, len(activities))
		for i, act := range activities {
			author := act.Author
			if act.Kind == task.ActivityKindComment && author != authorVibe && author != "agent" && author != "system" {
				author = authorHuman
			}
			items[i] = domain.TaskActivityItem{
				ID:        act.ID,
				Kind:      string(act.Kind),
				Author:    author,
				Body:      wrapUntrusted(act.Body),
				CreatedAt: act.CreatedAt,
			}
		}

		labels := t.Labels
		if labels == nil {
			labels = []string{}
		}
		out = domain.GetTaskOutput{
			ID:          t.ID,
			Key:         t.Key,
			ProjectID:   t.ProjectID,
			ExternalID:  t.ExternalID,
			ExternalURL: t.ExternalURL,
			Title:       wrapUntrusted(t.Title),
			Description: wrapUntrusted(desc),
			Truncated:   truncated,
			Status:      string(t.Status),
			Priority:    string(t.Priority),
			Type:        string(t.Type),
			Labels:      labels,
			Activities:  items,
		}
		return nil
	})
	return out, err
}

// ListTasks lists tasks within the caller's allowed project scope, capped at 50 items.
func (s *Service) ListTasks(ctx context.Context, caller domain.CallerContext, in domain.ListTasksInput) (domain.ListTasksOutput, error) {
	var out domain.ListTasksOutput
	args := map[string]any{
		"project_id": in.ProjectID,
		"status":     in.Status,
		"limit":      in.Limit,
	}
	err := s.execute(ctx, caller, "list_tasks", args, func() error {
		var projectIDs []string
		if in.ProjectID != "" {
			if !caller.AllowsProject(in.ProjectID) {
				return fmt.Errorf("%w: project %s is outside caller scope", domain.ErrForbiddenScope, in.ProjectID)
			}
			projectIDs = []string{in.ProjectID}
		} else {
			projectIDs = caller.ScopeProjectIDs()
			if len(projectIDs) == 0 {
				return fmt.Errorf("%w: no projects in caller scope", domain.ErrForbiddenScope)
			}
		}

		var statusFilter task.TaskStatus
		if in.Status != "" {
			statusFilter = task.TaskStatus(strings.TrimSpace(in.Status))
			if !statusFilter.IsValid() {
				return fmt.Errorf("%w: %q", domain.ErrInvalidStatus, in.Status)
			}
		}

		var matched []task.Record
		for _, pid := range projectIDs {
			items, err := s.tasks.List(ctx, task.TaskFilter{ProjectID: pid})
			if err != nil {
				return err
			}
			for _, t := range items {
				if statusFilter != "" && t.Status != statusFilter {
					continue
				}
				matched = append(matched, t)
			}
		}

		limit := in.Limit
		if limit <= 0 || limit > domain.MaxListTasksLimit {
			limit = domain.MaxListTasksLimit
		}
		total := len(matched)
		if len(matched) > limit {
			matched = matched[:limit]
		}

		summaries := make([]domain.TaskSummary, len(matched))
		for i, t := range matched {
			labels := t.Labels
			if labels == nil {
				labels = []string{}
			}
			summaries[i] = domain.TaskSummary{
				ID:         t.ID,
				Key:        t.Key,
				ProjectID:  t.ProjectID,
				ExternalID: t.ExternalID,
				Title:      wrapUntrusted(t.Title),
				Status:     string(t.Status),
				Priority:   string(t.Priority),
				Type:       string(t.Type),
				Labels:     labels,
			}
		}
		out = domain.ListTasksOutput{
			Tasks: summaries,
			Total: total,
		}
		return nil
	})
	return out, err
}

// SetStatus validates and transitions a task's status within caller scope.
func (s *Service) SetStatus(ctx context.Context, caller domain.CallerContext, in domain.SetStatusInput) (domain.SetStatusOutput, error) {
	var out domain.SetStatusOutput
	args := map[string]any{
		"ref":    in.Ref,
		"status": in.Status,
	}
	err := s.execute(ctx, caller, "set_status", args, func() error {
		target := task.TaskStatus(strings.TrimSpace(in.Status))
		if !target.IsValid() {
			return fmt.Errorf("%w: %q", domain.ErrInvalidStatus, in.Status)
		}
		t, err := s.resolveTask(ctx, caller, in.Ref)
		if err != nil {
			return err
		}
		updated, err := s.tasks.Update(ctx, t.ID, task.UpdateTaskRequest{Status: &target})
		if err != nil {
			if errors.Is(err, task.ErrInvalidStatus) {
				return fmt.Errorf("%w: %v", domain.ErrInvalidStatus, err)
			}
			if errors.Is(err, task.ErrTaskNotFound) {
				return fmt.Errorf("%w: %v", domain.ErrNotFound, err)
			}
			return err
		}
		out = domain.SetStatusOutput{
			ID:     updated.ID,
			Key:    updated.Key,
			Status: string(updated.Status),
		}
		return nil
	})
	return out, err
}

// AddActivity strips HTML comment markers, enforces size bounds, and records a vibe-authored activity.
func (s *Service) AddActivity(ctx context.Context, caller domain.CallerContext, in domain.AddActivityInput) (domain.AddActivityOutput, error) {
	var out domain.AddActivityOutput
	args := map[string]any{
		"ref":        in.Ref,
		"kind":       in.Kind,
		"body_bytes": len(in.Body),
	}
	err := s.execute(ctx, caller, "add_activity", args, func() error {
		if len(in.Body) > domain.MaxActivityBodyBytes {
			return fmt.Errorf("%w: activity body exceeds %d bytes", domain.ErrTooLarge, domain.MaxActivityBodyBytes)
		}
		body := stripHTMLComments(in.Body)
		if strings.TrimSpace(body) == "" {
			return fmt.Errorf("%w: activity body is empty", domain.ErrInvalidInput)
		}

		kind := task.ActivityKindComment
		if k := strings.TrimSpace(in.Kind); k != "" {
			kind = task.ActivityKind(k)
			if !isValidActivityKind(kind) {
				return fmt.Errorf("%w: invalid activity kind %q", domain.ErrInvalidInput, in.Kind)
			}
		}

		t, err := s.resolveTask(ctx, caller, in.Ref)
		if err != nil {
			return err
		}
		act, err := s.tasks.AddActivity(ctx, t.ID, kind, authorVibe, body)
		if err != nil {
			if errors.Is(err, task.ErrTaskNotFound) {
				return fmt.Errorf("%w: %v", domain.ErrNotFound, err)
			}
			return err
		}
		out = domain.AddActivityOutput{
			ID:        act.ID,
			TaskID:    t.ID,
			Kind:      string(act.Kind),
			Author:    act.Author,
			Body:      act.Body,
			CreatedAt: act.CreatedAt,
		}
		return nil
	})
	return out, err
}

// SubmitReport sanitizes, validates, and persists the structured report on the bound Run,
// and records a vibe-authored summary activity on the bound task.
func (s *Service) SubmitReport(ctx context.Context, caller domain.CallerContext, in domain.SubmitReportInput) (domain.SubmitReportOutput, error) {
	var out domain.SubmitReportOutput
	args := map[string]any{
		"summary_bytes": len(in.Summary),
		"criteria":      len(in.AcceptanceCriteria),
		"assumptions":   len(in.Assumptions),
	}
	err := s.execute(ctx, caller, "submit_report", args, func() error {
		if caller.RunID == "" {
			return fmt.Errorf("%w: no active run bound to caller", domain.ErrNotFound)
		}
		runID, err := uuid.Parse(caller.RunID)
		if err != nil {
			return fmt.Errorf("%w: invalid run id", domain.ErrNotFound)
		}
		r, err := s.runs.Get(ctx, runID)
		if err != nil {
			if errors.Is(err, run.ErrRunNotFound) {
				return fmt.Errorf("%w: run %s not found", domain.ErrNotFound, caller.RunID)
			}
			return err
		}
		if !caller.AllowsProject(r.ProjectID.String()) {
			return fmt.Errorf("%w: run project %s is outside caller scope", domain.ErrForbiddenScope, r.ProjectID)
		}

		sanitized := sanitizeReport(in.Report)
		if strings.TrimSpace(sanitized.Summary) == "" {
			return fmt.Errorf("%w: report summary is required", domain.ErrInvalidInput)
		}
		raw, err := json.Marshal(sanitized)
		if err != nil {
			return err
		}
		if len(raw) > domain.MaxReportBytes {
			return fmt.Errorf("%w: report exceeds %d bytes", domain.ErrTooLarge, domain.MaxReportBytes)
		}

		if _, err := s.runs.SetReport(ctx, runID, raw); err != nil {
			if errors.Is(err, run.ErrRunNotFound) {
				return fmt.Errorf("%w: run %s not found", domain.ErrNotFound, caller.RunID)
			}
			return err
		}

		actBody := formatReportActivity(sanitized)
		act, err := s.tasks.AddActivity(ctx, r.TaskID.String(), task.ActivityKindComment, authorVibe, actBody)
		if err != nil {
			return err
		}

		out = domain.SubmitReportOutput{
			RunID:      runID.String(),
			TaskID:     r.TaskID.String(),
			ActivityID: act.ID,
			Report:     sanitized,
		}
		return nil
	})
	return out, err
}

func (s *Service) execute(
	ctx context.Context,
	caller domain.CallerContext,
	tool string,
	args map[string]any,
	fn func() error,
) error {
	start := s.now()
	var err error
	if !s.allowCall(caller.RateLimitKey(), start) {
		err = fmt.Errorf("%w: exceeded %d calls per minute", domain.ErrRateLimited, domain.RateLimitPerMinute)
	} else {
		err = fn()
	}
	s.recordToolCall(ctx, caller, tool, args, err, s.now().Sub(start))
	return err
}

func (s *Service) allowCall(key string, now time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := now.Add(-domain.RateLimitWindow)

	// Evict expired buckets when approaching or at capacity so memory stays bounded.
	if len(s.buckets) >= domain.MaxRateLimitBuckets {
		for k, timestamps := range s.buckets {
			kept := timestamps[:0]
			for _, ts := range timestamps {
				if ts.After(cutoff) {
					kept = append(kept, ts)
				}
			}
			if len(kept) == 0 {
				delete(s.buckets, k)
			} else {
				s.buckets[k] = kept
			}
		}
	}

	entries, exists := s.buckets[key]
	if !exists && len(s.buckets) >= domain.MaxRateLimitBuckets {
		return false
	}

	kept := entries[:0]
	for _, ts := range entries {
		if ts.After(cutoff) {
			kept = append(kept, ts)
		}
	}
	if len(kept) >= domain.RateLimitPerMinute {
		s.buckets[key] = kept
		return false
	}
	s.buckets[key] = append(kept, now)
	return true
}

// Close releases any resources held by the agent service.
func (s *Service) Close() {}

func (s *Service) recordToolCall(
	ctx context.Context,
	caller domain.CallerContext,
	tool string,
	args map[string]any,
	callErr error,
	duration time.Duration,
) {
	if s.sessions == nil || caller.SessionID == "" {
		return
	}
	sessionID, err := uuid.Parse(caller.SessionID)
	if err != nil {
		return
	}
	status := "ok"
	errCode := ""
	if callErr != nil {
		status = "error"
		errCode = domain.Code(callErr)
		if errCode == "" {
			errCode = callErr.Error()
		}
	}
	payload, err := json.Marshal(map[string]any{
		"tool":        tool,
		"args":        args,
		"status":      status,
		"error":       errCode,
		"duration_ms": duration.Milliseconds(),
	})
	if err != nil {
		return
	}
	_, _ = s.sessions.RecordEvent(ctx, sessionID, eventKindToolCall, payload)
}

func (s *Service) resolveTask(ctx context.Context, caller domain.CallerContext, ref string) (task.Record, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return s.resolveBoundTask(ctx, caller)
	}
	if strings.HasPrefix(ref, "#") {
		return s.resolveExternalIssue(ctx, caller, ref)
	}

	var (
		t   task.Record
		err error
	)
	if strings.HasPrefix(strings.ToUpper(ref), "OP-") {
		t, err = s.tasks.GetByKey(ctx, ref)
	} else {
		t, err = s.tasks.Get(ctx, ref)
	}
	if err != nil {
		if errors.Is(err, task.ErrTaskNotFound) {
			return task.Record{}, fmt.Errorf("%w: task %q not found", domain.ErrNotFound, ref)
		}
		return task.Record{}, err
	}
	if !caller.AllowsProject(t.ProjectID) {
		return task.Record{}, fmt.Errorf("%w: task %s belongs to project %s", domain.ErrForbiddenScope, t.Key, t.ProjectID)
	}
	return t, nil
}

func (s *Service) resolveBoundTask(ctx context.Context, caller domain.CallerContext) (task.Record, error) {
	taskID := caller.TaskID
	if taskID == "" && caller.RunID != "" && s.runs != nil {
		runID, err := uuid.Parse(caller.RunID)
		if err == nil {
			r, err := s.runs.Get(ctx, runID)
			if err == nil {
				taskID = r.TaskID.String()
			}
		}
	}
	if taskID == "" {
		return task.Record{}, fmt.Errorf("%w: task ref is required", domain.ErrNotFound)
	}
	t, err := s.tasks.Get(ctx, taskID)
	if err != nil {
		if errors.Is(err, task.ErrTaskNotFound) {
			return task.Record{}, fmt.Errorf("%w: bound task %s not found", domain.ErrNotFound, taskID)
		}
		return task.Record{}, err
	}
	if !caller.AllowsProject(t.ProjectID) {
		return task.Record{}, fmt.Errorf("%w: task %s belongs to project %s", domain.ErrForbiddenScope, t.Key, t.ProjectID)
	}
	return t, nil
}

func (s *Service) resolveExternalIssue(ctx context.Context, caller domain.CallerContext, ref string) (task.Record, error) {
	issueNum := strings.TrimPrefix(ref, "#")
	if issueNum == "" {
		return task.Record{}, fmt.Errorf("%w: empty issue reference", domain.ErrNotFound)
	}
	projectIDs := caller.ScopeProjectIDs()
	if len(projectIDs) == 0 {
		return task.Record{}, fmt.Errorf("%w: no projects in caller scope", domain.ErrForbiddenScope)
	}
	for _, pid := range projectIDs {
		items, err := s.tasks.List(ctx, task.TaskFilter{ProjectID: pid})
		if err != nil {
			return task.Record{}, err
		}
		for _, t := range items {
			if t.ExternalID == issueNum || t.ExternalID == ref || strings.HasSuffix(t.ExternalID, "#"+issueNum) {
				return t, nil
			}
		}
	}
	return task.Record{}, fmt.Errorf("%w: issue %s not found in project scope", domain.ErrNotFound, ref)
}

func truncateUTF8(s string, maxBytes int) (string, bool) {
	if len(s) <= maxBytes {
		return s, false
	}
	cut := s[:maxBytes]
	for len(cut) > 0 && !utf8.ValidString(cut) {
		cut = cut[:len(cut)-1]
	}
	return cut, true
}

func wrapUntrusted(s string) string {
	escaped := strings.ReplaceAll(s, "</untrusted-task-data>", "&lt;/untrusted-task-data&gt;")
	return "<untrusted-task-data>" + escaped + "</untrusted-task-data>"
}

var htmlCommentReplacer = strings.NewReplacer("<!--", "", "-->", "")

func stripHTMLComments(s string) string {
	return htmlCommentReplacer.Replace(s)
}

func sanitizeReport(r domain.Report) domain.Report {
	out := domain.Report{
		Summary:   stripHTMLComments(r.Summary),
		PRURL:     stripHTMLComments(r.PRURL),
		CommitSHA: stripHTMLComments(r.CommitSHA),
	}
	if len(r.Assumptions) > 0 {
		out.Assumptions = make([]string, len(r.Assumptions))
		for i, a := range r.Assumptions {
			out.Assumptions[i] = stripHTMLComments(a)
		}
	}
	if len(r.AcceptanceCriteria) > 0 {
		out.AcceptanceCriteria = make([]domain.AcceptanceCriterion, len(r.AcceptanceCriteria))
		for i, ac := range r.AcceptanceCriteria {
			out.AcceptanceCriteria[i] = domain.AcceptanceCriterion{
				ID:        stripHTMLComments(ac.ID),
				Criterion: stripHTMLComments(ac.Criterion),
				Met:       ac.Met,
				Evidence:  stripHTMLComments(ac.Evidence),
			}
		}
	}
	return out
}

func formatReportActivity(r domain.Report) string {
	var b strings.Builder
	b.WriteString("Run report submitted: ")
	b.WriteString(strings.TrimSpace(r.Summary))
	if r.PRURL != "" {
		b.WriteString("\nPR: ")
		b.WriteString(r.PRURL)
	}
	if r.CommitSHA != "" {
		b.WriteString("\nCommit: ")
		b.WriteString(r.CommitSHA)
	}
	return b.String()
}

func isValidActivityKind(kind task.ActivityKind) bool {
	switch kind {
	case task.ActivityKindComment,
		task.ActivityKindCommit,
		task.ActivityKindReview,
		task.ActivityKindStatus,
		task.ActivityKindAssign:
		return true
	default:
		return false
	}
}
