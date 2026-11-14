package repo

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kilip/opus/internal/task/domain"
)

const dateLayout = "2006-01-02"

// ponytail: in-memory mutex store seeded with initial tasks; replace with Ent client when schema lands.
type Memory struct {
	mu       sync.RWMutex
	now      func() time.Time
	tasks    []domain.Task
	activity map[string][]domain.Activity
	nextID   int
}

type CreateInput struct {
	Title       string
	Description string
	Status      *domain.TaskStatus
	Priority    *domain.TaskPriority
	Type        *domain.TaskType
	Assignee    *domain.Assignee
	DueDate     string
	Labels      []string
}

type UpdateInput struct {
	Title         *string
	Description   *string
	Status        *domain.TaskStatus
	Priority      *domain.TaskPriority
	Type          *domain.TaskType
	Assignee      *domain.Assignee
	ClearAssignee bool
	DueDate       *string
	Labels        *[]string
}

func person(name string) *domain.Assignee {
	return &domain.Assignee{ID: strings.ReplaceAll(strings.ToLower(name), " ", "-"), Name: name}
}

func NewMemory(now func() time.Time) *Memory {
	if now == nil {
		now = time.Now
	}
	currentTime := now()
	ago := func(d time.Duration) time.Time { return currentTime.Add(-d) }
	hours := func(h int) time.Duration { return time.Duration(h) * time.Hour }
	days := func(d int) time.Duration { return time.Duration(d) * 24 * time.Hour }

	type seed struct {
		n          int
		title      string
		status     domain.TaskStatus
		priority   domain.TaskPriority
		kind       domain.TaskType
		assignee   string
		dueInDays  *int
		labels     []string
		desc       string
		createdAgo time.Duration
		updatedAgo time.Duration
	}

	seeds := []seed{
		{1, "Write PRD mission control", domain.TaskStatusDone, domain.TaskPriorityP3, domain.TaskTypeChore, "You", new(-3), []string{"product"}, "Scope, personas and success metrics first release.", days(14), hours(70)},
		{2, "Draft TDD workflow engine", domain.TaskStatusReview, domain.TaskPriorityP3, domain.TaskTypeChore, "planner-agent", new(1), []string{"architecture", "backend"}, "Describe how pipelines, agents and runs modelled and scheduled.", days(10), hours(5)},
		{3, "Implement omp agent runner", domain.TaskStatusInProgress, domain.TaskPriorityP3, domain.TaskTypeFeat, "coder-agent", new(4), []string{"backend", "agent"}, "Spawn omp processes, stream output and persist run logs.", days(8), hours(12)},
		{4, "Kanban board API (Fiber)", domain.TaskStatusInProgress, domain.TaskPriorityP2, domain.TaskTypeFeat, "coder-agent", new(6), []string{"backend"}, "CRUD and stage transitions tasks, optimistic ordering.", days(8), hours(20)},
		{5, "Auth: login endpoint", domain.TaskStatusTodo, domain.TaskPriorityP2, domain.TaskTypeFeat, "planner-agent", new(9), []string{"backend", "security"}, "Session cookies vs. tokens; decision pending.", days(6), hours(30)},
		{6, "Run log schema", domain.TaskStatusTodo, domain.TaskPriorityP1, domain.TaskTypeChore, "You", nil, []string{"architecture"}, "Event model agent runs: steps, tool calls, artifacts.", days(5), hours(40)},
		{7, "Set up CI (golangci-lint + biome)", domain.TaskStatusBacklog, domain.TaskPriorityP1, domain.TaskTypeChore, "coder-agent", nil, []string{"devops"}, "Run linters and tests on every push.", days(4), hours(50)},
		{8, "Review PR: runner retries", domain.TaskStatusReview, domain.TaskPriorityP2, domain.TaskTypeChore, "reviewer-agent", new(0), []string{"review", "agent"}, "Check backoff behaviour and cancellation handling.", days(2), hours(3)},
		{9, "Dashboard metrics", domain.TaskStatusBacklog, domain.TaskPriorityP0, domain.TaskTypeFeat, "You", nil, []string{"frontend"}, "Throughput, lead time and agent utilisation.", days(1), hours(2)},
	}

	tasks := make([]domain.Task, 0, len(seeds))
	for _, s := range seeds {
		t := domain.Task{
			ID:          "t" + strconv.Itoa(s.n),
			Key:         "OP-" + strconv.Itoa(s.n),
			Source:      domain.TaskSourceManual,
			Title:       s.title,
			Description: s.desc,
			Status:      s.status,
			Priority:    s.priority,
			Type:        s.kind,
			Assignee:    person(s.assignee),
			Labels:      s.labels,
			CreatedAt:   ago(s.createdAgo),
			UpdatedAt:   ago(s.updatedAgo),
		}
		if s.dueInDays != nil {
			t.DueDate = currentTime.AddDate(0, 0, *s.dueInDays).Format(dateLayout)
		}
		tasks = append(tasks, t)
	}

	activity := map[string][]domain.Activity{
		"t1": {
			{ID: "t1-a1", Kind: domain.ActivityKindComment, Author: "You", Body: "PRD approved. Moving on TDD.", CreatedAt: ago(hours(71))},
			{ID: "t1-a2", Kind: domain.ActivityKindStatus, Author: "You", Body: "Stage: In Review → Done", CreatedAt: ago(hours(70))},
		},
		"t2": {
			{ID: "t2-a1", Kind: domain.ActivityKindComment, Author: "planner-agent", Body: "Draft ready. Please check retry semantics in section 4.", CreatedAt: ago(hours(5))},
			{ID: "t2-a2", Kind: domain.ActivityKindStatus, Author: "planner-agent", Body: "Stage: Spec → In Review", CreatedAt: ago(hours(5) - time.Minute)},
		},
	}

	return &Memory{
		now:      now,
		tasks:    tasks,
		activity: activity,
		nextID:   200,
	}
}

func cloneAssignee(a *domain.Assignee) *domain.Assignee {
	if a == nil {
		return nil
	}
	cp := *a
	return &cp
}

func cloneLabels(labels []string) []string {
	if labels == nil {
		return []string{}
	}
	cp := make([]string, len(labels))
	copy(cp, labels)
	return cp
}

func cloneTask(t domain.Task) domain.Task {
	t.Assignee = cloneAssignee(t.Assignee)
	t.Labels = cloneLabels(t.Labels)
	return t
}

func (m *Memory) List() []domain.Task {
	m.mu.RLock()
	defer m.mu.RUnlock()

	out := make([]domain.Task, len(m.tasks))
	for i, t := range m.tasks {
		out[i] = cloneTask(t)
	}
	return out
}

func (m *Memory) Get(id string) (domain.Task, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, t := range m.tasks {
		if t.ID == id {
			return cloneTask(t), nil
		}
	}
	return domain.Task{}, domain.ErrTaskNotFound
}

func (m *Memory) Create(in CreateInput) domain.Task {
	m.mu.Lock()
	defer m.mu.Unlock()

	n := m.nextID
	m.nextID++
	id := fmt.Sprintf("OP-%d", n)

	status := domain.TaskStatusTodo
	if in.Status != nil && *in.Status != "" {
		status = *in.Status
	}

	priority := domain.TaskPriorityP1
	if in.Priority != nil && *in.Priority != "" {
		priority = *in.Priority
	}

	kind := domain.TaskTypeFeat
	if in.Type != nil && *in.Type != "" {
		kind = *in.Type
	}

	labels := []string{}
	if in.Labels != nil {
		labels = cloneLabels(in.Labels)
	}

	now := m.now()
	t := domain.Task{
		ID:          id,
		Key:         id,
		Source:      domain.TaskSourceManual,
		Title:       in.Title,
		Description: in.Description,
		Status:      status,
		Priority:    priority,
		Type:        kind,
		Assignee:    cloneAssignee(in.Assignee),
		DueDate:     in.DueDate,
		Labels:      labels,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	m.tasks = append(m.tasks, t)

	// initial activity
	actID := fmt.Sprintf("%s-a1", id)
	act := domain.Activity{
		ID:        actID,
		Kind:      domain.ActivityKindStatus,
		Author:    "toni",
		Body:      "created task · stage " + string(status),
		CreatedAt: now,
	}
	m.activity[id] = []domain.Activity{act}

	return cloneTask(t)
}

func (m *Memory) Update(id string, in UpdateInput) (domain.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	idx := -1
	for i, t := range m.tasks {
		if t.ID == id {
			idx = i
			break
		}
	}
	if idx == -1 {
		return domain.Task{}, domain.ErrTaskNotFound
	}

	t := &m.tasks[idx]
	now := m.now()

	if in.Title != nil {
		t.Title = *in.Title
	}
	if in.Description != nil {
		t.Description = *in.Description
	}
	if in.Priority != nil && *in.Priority != "" {
		t.Priority = *in.Priority
	}
	if in.Type != nil && *in.Type != "" {
		t.Type = *in.Type
	}
	if in.DueDate != nil {
		t.DueDate = *in.DueDate
	}
	if in.Labels != nil {
		t.Labels = cloneLabels(*in.Labels)
	}

	if in.ClearAssignee {
		if t.Assignee != nil {
			t.Assignee = nil
			m.appendActivityLocked(id, domain.ActivityKindAssign, "toni", "unassigned", now)
		}
	} else if in.Assignee != nil {
		prevName := ""
		if t.Assignee != nil {
			prevName = t.Assignee.Name
		}
		if in.Assignee.Name != prevName {
			t.Assignee = cloneAssignee(in.Assignee)
			m.appendActivityLocked(id, domain.ActivityKindAssign, "toni", "assigned to "+in.Assignee.Name, now)
		}
	}

	if in.Status != nil && *in.Status != "" && *in.Status != t.Status {
		prev := t.Status
		next := *in.Status
		t.Status = next
		m.appendActivityLocked(id, domain.ActivityKindStatus, "toni", "moved stage "+string(prev)+" → "+string(next), now)
	}

	t.UpdatedAt = now
	return cloneTask(*t), nil
}

func (m *Memory) appendActivityLocked(taskID string, kind domain.ActivityKind, author, body string, at time.Time) domain.Activity {
	acts := m.activity[taskID]
	actID := fmt.Sprintf("%s-a%d", taskID, len(acts)+1)
	act := domain.Activity{
		ID:        actID,
		Kind:      kind,
		Author:    author,
		Body:      body,
		CreatedAt: at,
	}
	m.activity[taskID] = append(acts, act)
	return act
}

func (m *Memory) ListActivity(taskID string) ([]domain.Activity, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	found := false
	for _, t := range m.tasks {
		if t.ID == taskID {
			found = true
			break
		}
	}
	if !found {
		return nil, domain.ErrTaskNotFound
	}

	acts := m.activity[taskID]
	out := make([]domain.Activity, len(acts))
	copy(out, acts)
	return out, nil
}

func (m *Memory) AddActivity(taskID string, kind domain.ActivityKind, author, body string) (domain.Activity, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	idx := -1
	for i, t := range m.tasks {
		if t.ID == taskID {
			idx = i
			break
		}
	}
	if idx == -1 {
		return domain.Activity{}, domain.ErrTaskNotFound
	}

	if kind == "" {
		kind = domain.ActivityKindComment
	}
	if author == "" {
		author = "toni"
	}

	now := m.now()
	m.tasks[idx].UpdatedAt = now

	act := m.appendActivityLocked(taskID, kind, author, body, now)
	return act, nil
}
