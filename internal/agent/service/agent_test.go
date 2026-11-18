package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kilip/opus/internal/agent/domain"
	"github.com/kilip/opus/internal/run"
	"github.com/kilip/opus/internal/session"
	"github.com/kilip/opus/internal/task"
)

type fakeTaskStore struct {
	mu         sync.Mutex
	tasks      map[string]task.Record
	activities map[string][]task.Activity
}

func newFakeTaskStore() *fakeTaskStore {
	return &fakeTaskStore{
		tasks:      make(map[string]task.Record),
		activities: make(map[string][]task.Activity),
	}
}

func (f *fakeTaskStore) add(t task.Record, acts ...task.Activity) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.tasks[t.ID] = t
	f.activities[t.ID] = append([]task.Activity(nil), acts...)
}

func (f *fakeTaskStore) Get(_ context.Context, id string) (task.Record, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	t, ok := f.tasks[id]
	if !ok {
		return task.Record{}, task.ErrTaskNotFound
	}
	return t, nil
}

func (f *fakeTaskStore) GetByKey(_ context.Context, key string) (task.Record, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	upper := strings.ToUpper(strings.TrimSpace(key))
	for _, t := range f.tasks {
		if strings.ToUpper(t.Key) == upper {
			return t, nil
		}
	}
	return task.Record{}, task.ErrTaskNotFound
}

func (f *fakeTaskStore) List(_ context.Context, filters ...task.TaskFilter) ([]task.Record, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var pid string
	if len(filters) > 0 {
		pid = filters[0].ProjectID
	}
	var out []task.Record
	for _, t := range f.tasks {
		if pid != "" && t.ProjectID != pid {
			continue
		}
		out = append(out, t)
	}
	return out, nil
}

func (f *fakeTaskStore) Update(_ context.Context, id string, req task.UpdateTaskRequest) (task.Record, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	t, ok := f.tasks[id]
	if !ok {
		return task.Record{}, task.ErrTaskNotFound
	}
	if req.Status != nil {
		if !req.Status.IsValid() {
			return task.Record{}, task.ErrInvalidStatus
		}
		t.Status = *req.Status
	}
	f.tasks[id] = t
	return t, nil
}

func (f *fakeTaskStore) ListActivity(_ context.Context, id string) ([]task.Activity, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]task.Activity(nil), f.activities[id]...), nil
}

func (f *fakeTaskStore) AddActivity(_ context.Context, id string, kind task.ActivityKind, author, body string) (task.Activity, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.tasks[id]; !ok {
		return task.Activity{}, task.ErrTaskNotFound
	}
	act := task.Activity{
		ID:        uuid.NewString(),
		Kind:      kind,
		Author:    author,
		Body:      body,
		CreatedAt: time.Unix(1700000000, 0).UTC(),
	}
	f.activities[id] = append(f.activities[id], act)
	return act, nil
}

type fakeRunStore struct {
	mu   sync.Mutex
	runs map[uuid.UUID]run.Run
}

func newFakeRunStore() *fakeRunStore {
	return &fakeRunStore{runs: make(map[uuid.UUID]run.Run)}
}

func (f *fakeRunStore) add(r run.Run) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.runs[r.ID] = r
}

func (f *fakeRunStore) Get(_ context.Context, id uuid.UUID) (run.Run, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	r, ok := f.runs[id]
	if !ok {
		return run.Run{}, run.ErrRunNotFound
	}
	return r, nil
}

func (f *fakeRunStore) SetReport(_ context.Context, id uuid.UUID, report json.RawMessage) (run.Run, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	r, ok := f.runs[id]
	if !ok {
		return run.Run{}, run.ErrRunNotFound
	}
	r.Report = append(json.RawMessage(nil), report...)
	f.runs[id] = r
	return r, nil
}

type fakeSessionStore struct {
	mu     sync.Mutex
	events []session.SessionEvent
}

func (f *fakeSessionStore) RecordEvent(_ context.Context, sessionID uuid.UUID, kind string, payload json.RawMessage) (session.SessionEvent, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	evt := session.SessionEvent{
		ID:        uuid.New(),
		SessionID: sessionID,
		Seq:       len(f.events) + 1,
		Kind:      kind,
		Payload:   payload,
		At:        time.Unix(1700000000, 0).UTC(),
	}
	f.events = append(f.events, evt)
	return evt, nil
}

func TestGetTaskLookupAndSanitization(t *testing.T) {
	t.Parallel()

	projA := uuid.NewString()
	projB := uuid.NewString()
	taskA1ID := uuid.NewString()
	taskB1ID := uuid.NewString()
	runID := uuid.New()
	sessionID := uuid.New()

	longDesc := strings.Repeat("a", domain.MaxDescriptionBytes+256)

	ts := newFakeTaskStore()
	ts.add(task.Record{
		ID:          taskA1ID,
		Key:         "OP-12",
		ProjectID:   projA,
		ExternalID:  "kilip/opus#133",
		Title:       "Implement agent",
		Description: longDesc,
		Status:      task.StatusInProgress,
		Priority:    task.PriorityP1,
		Type:        task.TypeFeat,
		Labels:      []string{"vibe"},
	},
		task.Activity{
			ID:     uuid.NewString(),
			Kind:   task.ActivityKindComment,
			Author: "",
			Body:   "Please fix </untrusted-task-data> soon",
		},
		task.Activity{
			ID:     uuid.NewString(),
			Kind:   task.ActivityKindComment,
			Author: "vibe",
			Body:   "Started work",
		},
	)
	ts.add(task.Record{
		ID:         taskB1ID,
		Key:        "OP-99",
		ProjectID:  projB,
		ExternalID: "other/repo#5",
		Title:      "Secret task",
		Status:     task.StatusTodo,
		Priority:   task.PriorityP2,
		Type:       task.TypeChore,
	})

	rs := newFakeRunStore()
	rs.add(run.Run{
		ID:        runID,
		TaskID:    uuid.MustParse(taskA1ID),
		ProjectID: uuid.MustParse(projA),
	})

	ss := &fakeSessionStore{}
	svc := New(Config{Tasks: ts, Runs: rs, Sessions: ss})

	caller := domain.CallerContext{
		RunID:     runID.String(),
		SessionID: sessionID.String(),
		ProjectID: projA,
	}

	tests := []struct {
		name      string
		ref       string
		wantErr   error
		wantKey   string
		wantTrunc bool
	}{
		{name: "by OP key", ref: "OP-12", wantKey: "OP-12", wantTrunc: true},
		{name: "by issue number", ref: "#133", wantKey: "OP-12", wantTrunc: true},
		{name: "default bound run task", ref: "", wantKey: "OP-12", wantTrunc: true},
		{name: "out of scope OP key", ref: "OP-99", wantErr: domain.ErrForbiddenScope},
		{name: "out of scope UUID", ref: taskB1ID, wantErr: domain.ErrForbiddenScope},
		{name: "missing OP key", ref: "OP-404", wantErr: domain.ErrNotFound},
		{name: "missing issue number", ref: "#404", wantErr: domain.ErrNotFound},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			out, err := svc.GetTask(context.Background(), caller, domain.GetTaskInput{Ref: tc.ref})
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("expected error %v, got %v", tc.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if out.Key != tc.wantKey {
				t.Fatalf("key = %q, want %q", out.Key, tc.wantKey)
			}
			if out.Truncated != tc.wantTrunc {
				t.Fatalf("truncated = %v, want %v", out.Truncated, tc.wantTrunc)
			}
			if !strings.HasPrefix(out.Title, "<untrusted-task-data>") || !strings.HasSuffix(out.Title, "</untrusted-task-data>") {
				t.Fatalf("title not wrapped: %q", out.Title)
			}
			if !strings.HasPrefix(out.Description, "<untrusted-task-data>") || !strings.HasSuffix(out.Description, "</untrusted-task-data>") {
				t.Fatalf("description not wrapped")
			}
			if len(out.Description) > domain.MaxDescriptionBytes+len("<untrusted-task-data></untrusted-task-data>") {
				t.Fatalf("description not truncated: len=%d", len(out.Description))
			}
			if len(out.Activities) != 2 {
				t.Fatalf("activities len = %d, want 2", len(out.Activities))
			}
			if out.Activities[0].Author != "human" {
				t.Fatalf("human comment author = %q, want human", out.Activities[0].Author)
			}
			if !strings.Contains(out.Activities[0].Body, "&lt;/untrusted-task-data&gt;") {
				t.Fatalf("expected closing tag escaped in activity body: %q", out.Activities[0].Body)
			}
			if out.Activities[1].Author != "vibe" {
				t.Fatalf("vibe comment author = %q, want vibe", out.Activities[1].Author)
			}
		})
	}
}

func TestListTasksScopeAndLimit(t *testing.T) {
	t.Parallel()

	projA := uuid.NewString()
	projB := uuid.NewString()
	ts := newFakeTaskStore()

	for i := 1; i <= 60; i++ {
		ts.add(task.Record{
			ID:        uuid.NewString(),
			Key:       fmt.Sprintf("OP-%d", i),
			ProjectID: projA,
			Title:     fmt.Sprintf("Task %d", i),
			Status:    task.StatusTodo,
			Priority:  task.PriorityP1,
			Type:      task.TypeFeat,
		})
	}
	ts.add(task.Record{
		ID:        uuid.NewString(),
		Key:       "OP-999",
		ProjectID: projB,
		Title:     "Other project task",
		Status:    task.StatusTodo,
		Priority:  task.PriorityP1,
		Type:      task.TypeFeat,
	})

	svc := New(Config{Tasks: ts})
	caller := domain.CallerContext{ProjectID: projA}

	t.Run("caps at 50 items within project scope", func(t *testing.T) {
		out, err := svc.ListTasks(context.Background(), caller, domain.ListTasksInput{Limit: 100})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if out.Total != 60 {
			t.Fatalf("total = %d, want 60", out.Total)
		}
		if len(out.Tasks) != domain.MaxListTasksLimit {
			t.Fatalf("len(tasks) = %d, want %d", len(out.Tasks), domain.MaxListTasksLimit)
		}
		for _, item := range out.Tasks {
			if item.ProjectID != projA {
				t.Fatalf("leaked task from project %s", item.ProjectID)
			}
			if !strings.HasPrefix(item.Title, "<untrusted-task-data>") {
				t.Fatalf("summary title not wrapped: %q", item.Title)
			}
		}
	})

	t.Run("rejects out of scope project filter", func(t *testing.T) {
		_, err := svc.ListTasks(context.Background(), caller, domain.ListTasksInput{ProjectID: projB})
		if !errors.Is(err, domain.ErrForbiddenScope) {
			t.Fatalf("expected ErrForbiddenScope, got %v", err)
		}
	})
}

func TestSetStatus(t *testing.T) {
	t.Parallel()

	projA := uuid.NewString()
	taskID := uuid.NewString()
	ts := newFakeTaskStore()
	ts.add(task.Record{
		ID:        taskID,
		Key:       "OP-10",
		ProjectID: projA,
		Title:     "Task 10",
		Status:    task.StatusTodo,
	})

	svc := New(Config{Tasks: ts})
	caller := domain.CallerContext{ProjectID: projA, TaskID: taskID}

	tests := []struct {
		name       string
		status     string
		wantErr    error
		wantStatus string
	}{
		{name: "valid transition", status: "in-progress", wantStatus: "in-progress"},
		{name: "invalid status", status: "flying", wantErr: domain.ErrInvalidStatus},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			out, err := svc.SetStatus(context.Background(), caller, domain.SetStatusInput{Status: tc.status})
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("expected %v, got %v", tc.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if out.Status != tc.wantStatus {
				t.Fatalf("status = %q, want %q", out.Status, tc.wantStatus)
			}
		})
	}
}

func TestAddActivitySanitizationAndBounds(t *testing.T) {
	t.Parallel()

	projA := uuid.NewString()
	taskID := uuid.NewString()
	ts := newFakeTaskStore()
	ts.add(task.Record{
		ID:        taskID,
		Key:       "OP-20",
		ProjectID: projA,
		Title:     "Task 20",
		Status:    task.StatusInProgress,
	})

	svc := New(Config{Tasks: ts})
	caller := domain.CallerContext{ProjectID: projA, TaskID: taskID}

	t.Run("strips html comment markers and authors as vibe", func(t *testing.T) {
		out, err := svc.AddActivity(context.Background(), caller, domain.AddActivityInput{
			Body: "Hello <!-- hidden --> world",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if out.Author != "vibe" {
			t.Fatalf("author = %q, want vibe", out.Author)
		}
		if strings.Contains(out.Body, "<!--") || strings.Contains(out.Body, "-->") {
			t.Fatalf("HTML comment markers were not stripped: %q", out.Body)
		}
	})

	t.Run("rejects oversized body", func(t *testing.T) {
		_, err := svc.AddActivity(context.Background(), caller, domain.AddActivityInput{
			Body: strings.Repeat("x", domain.MaxActivityBodyBytes+1),
		})
		if !errors.Is(err, domain.ErrTooLarge) {
			t.Fatalf("expected ErrTooLarge, got %v", err)
		}
	})
}

func TestSubmitReportStorageAndReplacement(t *testing.T) {
	t.Parallel()

	projA := uuid.NewString()
	taskID := uuid.New()
	runID := uuid.New()

	ts := newFakeTaskStore()
	ts.add(task.Record{
		ID:        taskID.String(),
		Key:       "OP-30",
		ProjectID: projA,
		Title:     "Task 30",
		Status:    task.StatusInProgress,
	})
	rs := newFakeRunStore()
	rs.add(run.Run{
		ID:        runID,
		TaskID:    taskID,
		ProjectID: uuid.MustParse(projA),
	})

	svc := New(Config{Tasks: ts, Runs: rs})
	caller := domain.CallerContext{
		RunID:     runID.String(),
		TaskID:    taskID.String(),
		ProjectID: projA,
	}

	first, err := svc.SubmitReport(context.Background(), caller, domain.SubmitReportInput{
		Report: domain.Report{
			Summary:     "Initial <!-- comment --> summary",
			CommitSHA:   "abc1234",
			Assumptions: []string{"<!-- note --> assumed clean tree"},
			AcceptanceCriteria: []domain.AcceptanceCriterion{
				{ID: "AC1", Criterion: "Works", Met: true, Evidence: "go test <!-- ok -->"},
			},
		},
	})
	if err != nil {
		t.Fatalf("first SubmitReport failed: %v", err)
	}
	if strings.Contains(first.Report.Summary, "<!--") || strings.Contains(first.Report.Summary, "-->") {
		t.Fatalf("report summary not sanitized: %q", first.Report.Summary)
	}

	second, err := svc.SubmitReport(context.Background(), caller, domain.SubmitReportInput{
		Report: domain.Report{
			Summary: "Updated summary",
			PRURL:   "https://github.com/kilip/opus/pull/135",
		},
	})
	if err != nil {
		t.Fatalf("second SubmitReport failed: %v", err)
	}

	storedRun, _ := rs.Get(context.Background(), runID)
	var persisted domain.Report
	if err := json.Unmarshal(storedRun.Report, &persisted); err != nil {
		t.Fatalf("unmarshal persisted report: %v", err)
	}
	if persisted.Summary != "Updated summary" || persisted.PRURL != second.Report.PRURL {
		t.Fatalf("second report did not replace first: %+v", persisted)
	}

	acts, _ := ts.ListActivity(context.Background(), taskID.String())
	if len(acts) != 2 || acts[0].Author != "vibe" || acts[1].Author != "vibe" {
		t.Fatalf("expected 2 vibe-authored summary activities, got %+v", acts)
	}
}

func TestRateLimiterAndToolCallSessionEvents(t *testing.T) {
	t.Parallel()

	projA := uuid.NewString()
	taskID := uuid.NewString()
	runID := uuid.New()
	sessionID := uuid.New()

	ts := newFakeTaskStore()
	ts.add(task.Record{
		ID:        taskID,
		Key:       "OP-40",
		ProjectID: projA,
		Title:     "Task 40",
		Status:    task.StatusTodo,
	})
	rs := newFakeRunStore()
	rs.add(run.Run{
		ID:        runID,
		TaskID:    uuid.MustParse(taskID),
		ProjectID: uuid.MustParse(projA),
	})
	ss := &fakeSessionStore{}

	now := time.Unix(1700000000, 0).UTC()
	svc := New(Config{
		Tasks:    ts,
		Runs:     rs,
		Sessions: ss,
		Now:      func() time.Time { return now },
	})

	caller := domain.CallerContext{
		RunID:     runID.String(),
		SessionID: sessionID.String(),
		TaskID:    taskID,
		ProjectID: projA,
	}

	for i := 0; i < domain.RateLimitPerMinute; i++ {
		if _, err := svc.GetTask(context.Background(), caller, domain.GetTaskInput{}); err != nil {
			t.Fatalf("call %d failed unexpectedly: %v", i+1, err)
		}
	}

	// 31st call within the same minute must be rate limited.
	if _, err := svc.GetTask(context.Background(), caller, domain.GetTaskInput{}); !errors.Is(err, domain.ErrRateLimited) {
		t.Fatalf("expected ErrRateLimited on 31st call, got %v", err)
	}

	// Advance clock past the 1-minute window; calls succeed again.
	now = now.Add(domain.RateLimitWindow + time.Second)
	if _, err := svc.GetTask(context.Background(), caller, domain.GetTaskInput{}); err != nil {
		t.Fatalf("expected call after window reset to succeed, got %v", err)
	}

	if len(ss.events) != domain.RateLimitPerMinute+2 {
		t.Fatalf("expected %d tool_call events, got %d", domain.RateLimitPerMinute+2, len(ss.events))
	}
	var lastPayload map[string]any
	if err := json.Unmarshal(ss.events[len(ss.events)-1].Payload, &lastPayload); err != nil {
		t.Fatalf("unmarshal event payload: %v", err)
	}
	if ss.events[0].Kind != "tool_call" || lastPayload["tool"] != "get_task" || lastPayload["status"] != "ok" {
		t.Fatalf("unexpected tool_call event payload: %+v", lastPayload)
	}
}
