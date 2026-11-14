package handler

import (
	"strconv"
	"strings"
	"time"
)

const dateLayout = "2006-01-02"

// fixtures is the static data served until tasks are backed by storage. Dates
// are relative to now so the board always has overdue and upcoming work.
type fixtures struct {
	tasks    []Task
	activity map[string][]Activity
}

func (f fixtures) task(id string) (Task, bool) {
	for _, t := range f.tasks {
		if t.ID == id {
			return t, true
		}
	}
	return Task{}, false
}

func person(name string) *Assignee {
	return &Assignee{ID: strings.ReplaceAll(strings.ToLower(name), " ", "-"), Name: name}
}

func due(days int) *int { return &days }

func newFixtures(now time.Time) fixtures {
	ago := func(d time.Duration) time.Time { return now.Add(-d) }
	hours := func(h int) time.Duration { return time.Duration(h) * time.Hour }
	days := func(d int) time.Duration { return time.Duration(d) * 24 * time.Hour }

	type seed struct {
		n          int
		title      string
		status     TaskStatus
		priority   TaskPriority
		kind       TaskType
		assignee   string
		dueInDays  *int
		labels     []string
		desc       string
		createdAgo time.Duration
		updatedAgo time.Duration
	}

	seeds := []seed{
		{1, "Write PRD for mission control", TaskStatusDone, TaskPriorityHigh, TaskTypeChore, "You", due(-3), []string{"product"}, "Scope, personas and success metrics for the first release.", days(14), hours(70)},
		{2, "Draft TDD for the workflow engine", TaskStatusInReview, TaskPriorityHigh, TaskTypeChore, "planner-agent", due(1), []string{"architecture", "backend"}, "Describe how pipelines, agents and runs are modelled and scheduled.", days(10), hours(5)},
		{3, "Implement omp agent runner", TaskStatusInProgress, TaskPriorityHigh, TaskTypeFeature, "coder-agent", due(4), []string{"backend", "agent"}, "Spawn omp processes, stream their output and persist run logs.", days(8), hours(12)},
		{4, "Kanban board API (Fiber)", TaskStatusInProgress, TaskPriorityMedium, TaskTypeFeature, "coder-agent", due(6), []string{"backend"}, "CRUD and stage transitions for tasks, with optimistic ordering.", days(8), hours(20)},
		{5, "Auth: login endpoint", TaskStatusSpec, TaskPriorityMedium, TaskTypeFeature, "planner-agent", due(9), []string{"backend", "security"}, "Session cookies vs. tokens; decision pending.", days(6), hours(30)},
		{6, "Run log schema", TaskStatusSpec, TaskPriorityLow, TaskTypeChore, "You", nil, []string{"architecture"}, "Event model for agent runs: steps, tool calls, artifacts.", days(5), hours(40)},
		{7, "Set up CI (golangci-lint + biome)", TaskStatusBacklog, TaskPriorityLow, TaskTypeChore, "coder-agent", nil, []string{"devops"}, "Run linters and tests on every push.", days(4), hours(50)},
		{8, "Review PR: runner retries", TaskStatusInReview, TaskPriorityMedium, TaskTypeChore, "reviewer-agent", due(0), []string{"review", "agent"}, "Check backoff behaviour and cancellation handling.", days(2), hours(3)},
		{9, "Dashboard metrics", TaskStatusBacklog, TaskPriorityNone, TaskTypeFeature, "You", nil, []string{"frontend"}, "Throughput, lead time and agent utilisation.", days(1), hours(2)},
	}

	tasks := make([]Task, 0, len(seeds))
	for _, s := range seeds {
		t := Task{
			ID:          "t" + strconv.Itoa(s.n),
			Key:         "OPUS-" + strconv.Itoa(s.n),
			Source:      TaskSourceCustom,
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
			t.DueDate = now.AddDate(0, 0, *s.dueInDays).Format(dateLayout)
		}
		tasks = append(tasks, t)
	}

	return fixtures{
		tasks: tasks,
		activity: map[string][]Activity{
			"t1": {
				{ID: "t1-a1", Kind: ActivityKindNote, Author: "You", Body: "PRD approved. Moving on to the TDD.", CreatedAt: ago(hours(71))},
				{ID: "t1-a2", Kind: ActivityKindTracking, Author: "You", Body: "Stage: In Review → Done", CreatedAt: ago(hours(70))},
			},
			"t2": {
				{ID: "t2-a1", Kind: ActivityKindMessage, Author: "planner-agent", Body: "Draft is ready. Please check the retry semantics in section 4.", CreatedAt: ago(hours(5))},
				{ID: "t2-a2", Kind: ActivityKindTracking, Author: "planner-agent", Body: "Stage: Spec → In Review", CreatedAt: ago(hours(5) - time.Minute)},
			},
		},
	}
}
