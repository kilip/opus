package repo_test

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/kilip/opus/ent"
	"github.com/kilip/opus/ent/enttest"
	"github.com/kilip/opus/internal/task/domain"
	"github.com/kilip/opus/internal/task/repo"
	_ "modernc.org/sqlite"
)

func setupTestClient(t *testing.T) *ent.Client {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+t.Name()+"?mode=memory&cache=shared&_pragma=foreign_keys(ON)&_pragma=busy_timeout(10000)")
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	db.SetMaxOpenConns(1)
	drv := entsql.OpenDB(dialect.SQLite, db)
	client := enttest.NewClient(t, enttest.WithOptions(ent.Driver(drv)))
	t.Cleanup(func() {
		_ = client.Close()
	})
	return client
}

func createTestProject(t *testing.T, client *ent.Client) *ent.Project {
	t.Helper()
	p, err := client.Project.Create().
		SetSlug(fmt.Sprintf("repo-test-proj-%s", uuid.New().String()[:8])).
		SetName("Test Project").
		SetRepoProvider("github").
		SetRepoRef("org/repo").
		SetTrackerProvider("github").
		SetTrackerRef("org/repo").
		Save(context.Background())
	if err != nil {
		t.Fatalf("create test project failed: %v", err)
	}
	return p
}

// AC2: Keys are OP-1, OP-2, ... from task_key counter, allocated in same transaction as insert.
// Test runs 20 concurrent creates and asserts keys OP-1..OP-20 with no gaps or duplicates.
func TestEntRepo_ConcurrentCreateKeys(t *testing.T) {
	client := setupTestClient(t)
	r := repo.New(client)
	ctx := context.Background()
	p := createTestProject(t, client)

	const n = 20
	var wg sync.WaitGroup
	var mu sync.Mutex
	keys := make([]string, 0, n)
	errs := make([]error, 0, n)

	wg.Add(n)
	for i := 1; i <= n; i++ {
		go func(idx int) {
			defer wg.Done()
			tsk, err := r.Create(ctx, repo.CreateInput{
				ProjectID: p.ID.String(),
				Title:     fmt.Sprintf("Concurrent task %d", idx),
			})
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errs = append(errs, err)
			} else {
				keys = append(keys, tsk.Key)
			}
		}(i)
	}
	wg.Wait()

	if len(errs) > 0 {
		t.Fatalf("expected 0 errors in concurrent creates, got %d: %v", len(errs), errs[0])
	}

	if len(keys) != n {
		t.Fatalf("expected %d keys, got %d", n, len(keys))
	}

	// Extract and sort counter numbers
	nums := make([]int, len(keys))
	for i, k := range keys {
		if !strings.HasPrefix(k, "OP-") {
			t.Fatalf("key %q does not have OP- prefix", k)
		}
		numStr := strings.TrimPrefix(k, "OP-")
		val, err := strconv.Atoi(numStr)
		if err != nil {
			t.Fatalf("failed to parse key number from %q: %v", k, err)
		}
		nums[i] = val
	}
	sort.Ints(nums)

	for i := 1; i <= n; i++ {
		if nums[i-1] != i {
			t.Errorf("at index %d: want OP-%d, got OP-%d", i-1, i, nums[i-1])
		}
	}
}

// AC3: Repo behavior matches current Memory contract (defaults, activities, UpdateResult).
// Test suite runs against Ent repo using in-memory SQLite client, and author is stored as NULL.
func TestEntRepo_ContractParity(t *testing.T) {
	client := setupTestClient(t)
	r := repo.New(client)
	ctx := context.Background()
	p := createTestProject(t, client)

	// 1. Defaults
	tsk, err := r.Create(ctx, repo.CreateInput{
		ProjectID: p.ID.String(),
		Title:     "Test default task",
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	if tsk.Status != domain.StatusTodo {
		t.Errorf("status = %q, want %q", tsk.Status, domain.StatusTodo)
	}
	if tsk.Priority != domain.TaskPriorityP1 {
		t.Errorf("priority = %q, want %q", tsk.Priority, domain.TaskPriorityP1)
	}
	if tsk.Type != domain.TaskTypeFeat {
		t.Errorf("type = %q, want %q", tsk.Type, domain.TaskTypeFeat)
	}
	if tsk.Source != domain.TaskSourceManual {
		t.Errorf("source = %q, want %q", tsk.Source, domain.TaskSourceManual)
	}
	if tsk.Labels == nil {
		t.Errorf("labels is nil, want empty slice")
	}
	if tsk.Assignee != nil {
		t.Errorf("assignee = %v, want nil", tsk.Assignee)
	}

	// Verify initial activity and author is stored as NULL in Ent
	acts, err := r.ListActivity(ctx, tsk.ID)
	if err != nil {
		t.Fatalf("list activity failed: %v", err)
	}
	if len(acts) != 1 {
		t.Fatalf("expected 1 initial activity, got %d", len(acts))
	}
	if acts[0].Kind != domain.ActivityKindStatus {
		t.Errorf("act kind = %q, want %q", acts[0].Kind, domain.ActivityKindStatus)
	}
	if acts[0].Body != "created task · stage todo" {
		t.Errorf("act body = %q, want 'created task · stage todo'", acts[0].Body)
	}
	// Query Ent Activity directly to verify author is stored as NULL in DB
	actUID, err := uuid.Parse(acts[0].ID)
	if err != nil {
		t.Fatalf("parse act ID failed: %v", err)
	}
	entAct, err := client.Activity.Get(ctx, actUID)
	if err != nil {
		t.Fatalf("get ent activity failed: %v", err)
	}
	if entAct.Author != nil {
		t.Errorf("expected Author to be stored as NULL (nil), got %v", *entAct.Author)
	}

	// 2. Update status and check UpdateResult
	newStatus := domain.StatusInProgress
	upRes, err := r.Update(ctx, tsk.ID, repo.UpdateInput{
		Status: &newStatus,
	})
	if err != nil {
		t.Fatalf("update status failed: %v", err)
	}
	if !upRes.StatusChanged {
		t.Errorf("expected StatusChanged = true")
	}
	if upRes.PrevStatus != domain.StatusTodo {
		t.Errorf("PrevStatus = %q, want %q", upRes.PrevStatus, domain.StatusTodo)
	}
	if upRes.Task.Status != domain.StatusInProgress {
		t.Errorf("Task.Status = %q, want %q", upRes.Task.Status, domain.StatusInProgress)
	}

	// Transition to blocked
	blockedStatus := domain.StatusBlocked
	upRes, err = r.Update(ctx, tsk.ID, repo.UpdateInput{
		Status: &blockedStatus,
	})
	if err != nil {
		t.Fatalf("update blocked status failed: %v", err)
	}
	if !upRes.StatusChanged || upRes.PrevStatus != domain.StatusInProgress || upRes.Task.Status != domain.StatusBlocked {
		t.Errorf("unexpected blocked transition result: %+v", upRes)
	}

	// 3. Assign and unassign
	assignee := &domain.Assignee{ID: "test-user-1", Name: "Test User"}
	upRes, err = r.Update(ctx, tsk.ID, repo.UpdateInput{
		Assignee: assignee,
	})
	if err != nil {
		t.Fatalf("update assignee failed: %v", err)
	}
	if upRes.Task.Assignee == nil || upRes.Task.Assignee.Name != "Test User" {
		t.Errorf("assignee not updated properly: %v", upRes.Task.Assignee)
	}

	upRes, err = r.Update(ctx, tsk.ID, repo.UpdateInput{
		ClearAssignee: true,
	})
	if err != nil {
		t.Fatalf("clear assignee failed: %v", err)
	}
	if upRes.Task.Assignee != nil {
		t.Errorf("expected assignee to be nil after clear, got %v", upRes.Task.Assignee)
	}

	// 4. AddActivity with empty author stores NULL; non-empty author is persisted
	addedAct, err := r.AddActivity(ctx, tsk.ID, domain.ActivityKindComment, "", "A custom comment")
	if err != nil {
		t.Fatalf("add activity failed: %v", err)
	}
	if addedAct.Body != "A custom comment" {
		t.Errorf("added act body = %q, want 'A custom comment'", addedAct.Body)
	}
	// Verify empty author stored as NULL in Ent
	addedUID, err := uuid.Parse(addedAct.ID)
	if err != nil {
		t.Fatalf("parse added act ID failed: %v", err)
	}
	entAddedAct, err := client.Activity.Get(ctx, addedUID)
	if err != nil {
		t.Fatalf("get ent added activity failed: %v", err)
	}
	if entAddedAct.Author != nil {
		t.Errorf("expected added activity Author to be NULL (nil), got %v", *entAddedAct.Author)
	}

	authoredAct, err := r.AddActivity(ctx, tsk.ID, domain.ActivityKindComment, "sync-bot", "Authored comment")
	if err != nil {
		t.Fatalf("add authored activity failed: %v", err)
	}
	if authoredAct.Author != "sync-bot" {
		t.Errorf("authored act author = %q, want 'sync-bot'", authoredAct.Author)
	}
}

// AC4: (source, external_id) is unique: duplicate pair fails, while two tasks with NULL external_id both succeed.
func TestEntRepo_ExternalIDUniqueness(t *testing.T) {
	client := setupTestClient(t)
	r := repo.New(client)
	ctx := context.Background()
	p := createTestProject(t, client)

	// Two tasks with empty external_id (stored as NULL in DB) should both succeed
	t1, err := r.Create(ctx, repo.CreateInput{
		ProjectID:  p.ID.String(),
		Source:     domain.TaskSourceManual,
		ExternalID: "",
		Title:      "Task without external id 1",
	})
	if err != nil {
		t.Fatalf("first task with empty external_id failed: %v", err)
	}

	t2, err := r.Create(ctx, repo.CreateInput{
		ProjectID:  p.ID.String(),
		Source:     domain.TaskSourceManual,
		ExternalID: "",
		Title:      "Task without external id 2",
	})
	if err != nil {
		t.Fatalf("second task with empty external_id failed: %v", err)
	}
	if t1.ID == t2.ID {
		t.Fatalf("expected different IDs, got same: %s", t1.ID)
	}

	// Create task with source + external_id pair
	_, err = r.Create(ctx, repo.CreateInput{
		ProjectID:  p.ID.String(),
		Source:     domain.TaskSourceGitHub,
		ExternalID: "issue-42",
		Title:      "GitHub Issue 42",
	})
	if err != nil {
		t.Fatalf("create github task failed: %v", err)
	}

	// Duplicate pair must fail
	_, err = r.Create(ctx, repo.CreateInput{
		ProjectID:  p.ID.String(),
		Source:     domain.TaskSourceGitHub,
		ExternalID: "issue-42",
		Title:      "GitHub Issue 42 duplicate",
	})
	if err == nil {
		t.Fatalf("expected error on duplicate (source, external_id), got nil")
	}
}

// AC5: Lists are ordered by id. Unknown or malformed IDs return domain.ErrTaskNotFound
// (test covers random valid UUID and non-UUID string).
func TestEntRepo_ListOrderingAndNotFound(t *testing.T) {
	client := setupTestClient(t)
	r := repo.New(client)
	ctx := context.Background()
	p := createTestProject(t, client)

	// Create multiple tasks
	var createdIDs []string
	for i := 1; i <= 5; i++ {
		tsk, err := r.Create(ctx, repo.CreateInput{
			ProjectID: p.ID.String(),
			Title:     fmt.Sprintf("Task %d", i),
		})
		if err != nil {
			t.Fatalf("create task %d failed: %v", i, err)
		}
		createdIDs = append(createdIDs, tsk.ID)
	}

	list, err := r.List(ctx)
	if err != nil {
		t.Fatalf("list tasks failed: %v", err)
	}
	if len(list) != len(createdIDs) {
		t.Fatalf("got %d tasks, want %d", len(list), len(createdIDs))
	}

	// Verify ordered by id
	for i := 0; i < len(list)-1; i++ {
		if list[i].ID >= list[i+1].ID {
			t.Errorf("list not ordered by id: list[%d]=%s >= list[%d]=%s", i, list[i].ID, i+1, list[i+1].ID)
		}
	}

	// Test unknown valid UUID
	randomUUID := uuid.New().String()
	if _, err := r.Get(ctx, randomUUID); err != domain.ErrTaskNotFound {
		t.Errorf("Get unknown UUID: got %v, want ErrTaskNotFound", err)
	}
	if _, err := r.Update(ctx, randomUUID, repo.UpdateInput{}); err != domain.ErrTaskNotFound {
		t.Errorf("Update unknown UUID: got %v, want ErrTaskNotFound", err)
	}
	if _, err := r.ListActivity(ctx, randomUUID); err != domain.ErrTaskNotFound {
		t.Errorf("ListActivity unknown UUID: got %v, want ErrTaskNotFound", err)
	}
	if _, err := r.AddActivity(ctx, randomUUID, domain.ActivityKindComment, "author", "body"); err != domain.ErrTaskNotFound {
		t.Errorf("AddActivity unknown UUID: got %v, want ErrTaskNotFound", err)
	}

	// Test non-UUID string
	nonUUID := "invalid-uuid-string"
	if _, err := r.Get(ctx, nonUUID); err != domain.ErrTaskNotFound {
		t.Errorf("Get non-UUID: got %v, want ErrTaskNotFound", err)
	}
	if _, err := r.Update(ctx, nonUUID, repo.UpdateInput{}); err != domain.ErrTaskNotFound {
		t.Errorf("Update non-UUID: got %v, want ErrTaskNotFound", err)
	}
	if _, err := r.ListActivity(ctx, nonUUID); err != domain.ErrTaskNotFound {
		t.Errorf("ListActivity non-UUID: got %v, want ErrTaskNotFound", err)
	}
	if _, err := r.AddActivity(ctx, nonUUID, domain.ActivityKindComment, "author", "body"); err != domain.ErrTaskNotFound {
		t.Errorf("AddActivity non-UUID: got %v, want ErrTaskNotFound", err)
	}
}

// AC4: Restart persistence test: create task, close Ent client, reopen on same SQLite DB file, list back.
func TestEntRepo_RestartPersistence(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "persist.db")
	dsn := fmt.Sprintf("file:%s?_pragma=foreign_keys(ON)&_pragma=busy_timeout(10000)", dbPath)

	ctx := context.Background()

	// Initial session: open SQLite, run migrations, create task
	db1, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	db1.SetMaxOpenConns(1)
	drv1 := entsql.OpenDB(dialect.SQLite, db1)
	client1 := enttest.NewClient(t, enttest.WithOptions(ent.Driver(drv1)))

	r1 := repo.New(client1)
	p1 := createTestProject(t, client1)
	created, err := r1.Create(ctx, repo.CreateInput{
		ProjectID:   p1.ID.String(),
		Title:       "Persistent Task Across Restarts",
		Description: "Should persist after client close and reopen",
	})
	if err != nil {
		t.Fatalf("create task failed: %v", err)
	}
	if created.Key != "OP-1" {
		t.Fatalf("expected key OP-1, got %s", created.Key)
	}

	// Close client1
	if err := client1.Close(); err != nil {
		t.Fatalf("close client1 failed: %v", err)
	}

	// Reopen on same SQLite DB file
	db2, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("reopen sqlite failed: %v", err)
	}
	db2.SetMaxOpenConns(1)
	drv2 := entsql.OpenDB(dialect.SQLite, db2)
	client2 := ent.NewClient(ent.Driver(drv2))
	t.Cleanup(func() {
		_ = client2.Close()
	})

	r2 := repo.New(client2)
	tasks, err := r2.List(ctx)
	if err != nil {
		t.Fatalf("list tasks after restart failed: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task after restart, got %d", len(tasks))
	}
	if tasks[0].ID != created.ID {
		t.Errorf("task ID = %s, want %s", tasks[0].ID, created.ID)
	}
	if tasks[0].Key != "OP-1" {
		t.Errorf("task Key = %s, want OP-1", tasks[0].Key)
	}
	if tasks[0].Title != "Persistent Task Across Restarts" {
		t.Errorf("task Title = %q, want %q", tasks[0].Title, "Persistent Task Across Restarts")
	}

	// Create second task on reopened client to ensure counter persists
	created2, err := r2.Create(ctx, repo.CreateInput{
		ProjectID: p1.ID.String(),
		Title:     "Second Persistent Task",
	})
	if err != nil {
		t.Fatalf("create second task failed: %v", err)
	}
	if created2.Key != "OP-2" {
		t.Fatalf("expected key OP-2, got %s", created2.Key)
	}
}

func TestEntRepo_CountAndFilterByProject(t *testing.T) {
	client := setupTestClient(t)
	r := repo.New(client)
	ctx := context.Background()

	p1 := createTestProject(t, client)
	p2 := createTestProject(t, client)

	// Initially 0 count
	c0, err := r.CountByProject(ctx, p1.ID.String())
	if err != nil || c0 != 0 {
		t.Fatalf("expected count 0, got %d (err: %v)", c0, err)
	}

	// Create 2 tasks for p1, 1 task for p2
	_, err = r.Create(ctx, repo.CreateInput{ProjectID: p1.ID.String(), Title: "T1"})
	if err != nil {
		t.Fatalf("create T1: %v", err)
	}
	_, err = r.Create(ctx, repo.CreateInput{ProjectID: p1.ID.String(), Title: "T2"})
	if err != nil {
		t.Fatalf("create T2: %v", err)
	}
	_, err = r.Create(ctx, repo.CreateInput{ProjectID: p2.ID.String(), Title: "T3"})
	if err != nil {
		t.Fatalf("create T3: %v", err)
	}

	// Count check
	cP1, err := r.CountByProject(ctx, p1.ID.String())
	if err != nil || cP1 != 2 {
		t.Fatalf("expected count 2 for p1, got %d (err: %v)", cP1, err)
	}
	cP2, err := r.CountByProject(ctx, p2.ID.String())
	if err != nil || cP2 != 1 {
		t.Fatalf("expected count 1 for p2, got %d (err: %v)", cP2, err)
	}

	// Filtered list check
	p1Tasks, err := r.List(ctx, domain.TaskFilter{ProjectID: p1.ID.String()})
	if err != nil {
		t.Fatalf("list p1 tasks: %v", err)
	}
	if len(p1Tasks) != 2 {
		t.Fatalf("expected 2 tasks for p1, got %d", len(p1Tasks))
	}
	for _, tk := range p1Tasks {
		if tk.ProjectID != p1.ID.String() {
			t.Errorf("task project ID = %s, want %s", tk.ProjectID, p1.ID.String())
		}
	}

	// Unfiltered list check
	allTasks, err := r.List(ctx)
	if err != nil {
		t.Fatalf("list all tasks: %v", err)
	}
	if len(allTasks) != 3 {
		t.Fatalf("expected 3 total tasks, got %d", len(allTasks))
	}
}

func TestEntRepo_ImportFindAndLabels(t *testing.T) {
	client := setupTestClient(t)
	r := repo.New(client)
	ctx := context.Background()
	p := createTestProject(t, client)

	// FindByExternal on missing returns ErrTaskNotFound
	if _, err := r.FindByExternal(ctx, domain.TaskSourceGitHub, "gh-99"); err != domain.ErrTaskNotFound {
		t.Fatalf("expected ErrTaskNotFound, got %v", err)
	}

	// Import without key allocates OP-1
	syncAt1 := time.Date(2026, 11, 17, 1, 0, 0, 0, time.UTC)
	st := domain.StatusInProgress
	prio := domain.TaskPriorityP2
	tp := domain.TaskTypeBug
	in := repo.CreateInput{
		ProjectID:    p.ID.String(),
		Source:       domain.TaskSourceGitHub,
		ExternalID:   "gh-99",
		ExternalURL:  "https://github.com/org/repo/issues/99",
		SourceStatus: "open",
		Title:        "Imported Bug",
		Description:  "Fix bug",
		Status:       &st,
		Priority:     &prio,
		Type:         &tp,
		Labels:       []string{"bug"},
		SyncedAt:     &syncAt1,
	}
	res1, err := r.Import(ctx, in)
	if err != nil {
		t.Fatalf("first import failed: %v", err)
	}
	if !res1.Created || !res1.Changed || res1.Task.Key != "OP-1" {
		t.Fatalf("unexpected first import result: %+v", res1)
	}
	if res1.Task.SyncedAt == nil || !res1.Task.SyncedAt.Equal(syncAt1) {
		t.Fatalf("expected SyncedAt=%v, got %v", syncAt1, res1.Task.SyncedAt)
	}
	if !res1.Task.UpdatedAt.Equal(syncAt1) {
		t.Fatalf("expected UpdatedAt=%v, got %v", syncAt1, res1.Task.UpdatedAt)
	}

	// FindByExternal finds it
	found, err := r.FindByExternal(ctx, domain.TaskSourceGitHub, "gh-99")
	if err != nil || found.ID != res1.Task.ID {
		t.Fatalf("FindByExternal = %+v, err=%v", found, err)
	}

	// Second Import with identical fields and only a newer SyncedAt does NO write (Changed=false, UpdatedAt/SyncedAt unchanged)
	syncAt2 := syncAt1.Add(5 * time.Minute)
	in.SyncedAt = &syncAt2
	res2, err := r.Import(ctx, in)
	if err != nil {
		t.Fatalf("second import failed: %v", err)
	}
	if res2.Created || res2.Changed || res2.StatusChanged {
		t.Fatalf("expected no-op on identical import, got %+v", res2)
	}
	if !res2.Task.UpdatedAt.Equal(res1.Task.UpdatedAt) {
		t.Fatalf("expected UpdatedAt unchanged on no-op import: %v vs %v", res2.Task.UpdatedAt, res1.Task.UpdatedAt)
	}
	if res2.Task.SyncedAt == nil || !res2.Task.SyncedAt.Equal(syncAt1) {
		t.Fatalf("expected SyncedAt unchanged (%v) on no-op import, got %v", syncAt1, res2.Task.SyncedAt)
	}

	// Label helpers
	if err := r.AddLabel(ctx, res1.Task.ID, "urgent"); err != nil {
		t.Fatalf("AddLabel failed: %v", err)
	}
	if err := r.AddLabel(ctx, res1.Task.ID, "urgent"); err != nil {
		t.Fatalf("duplicate AddLabel failed: %v", err)
	}
	got, _ := r.Get(ctx, res1.Task.ID)
	if len(got.Labels) != 2 || got.Labels[0] != "bug" || got.Labels[1] != "urgent" {
		t.Fatalf("unexpected labels after AddLabel: %v", got.Labels)
	}
	if err := r.RemoveLabel(ctx, res1.Task.ID, "bug"); err != nil {
		t.Fatalf("RemoveLabel failed: %v", err)
	}
	got, _ = r.Get(ctx, res1.Task.ID)
	if len(got.Labels) != 1 || got.Labels[0] != "urgent" {
		t.Fatalf("unexpected labels after RemoveLabel: %v", got.Labels)
	}
	if err := r.SetLabels(ctx, res1.Task.ID, []string{"triage", "backend"}); err != nil {
		t.Fatalf("SetLabels failed: %v", err)
	}
	got, _ = r.Get(ctx, res1.Task.ID)
	if len(got.Labels) != 2 || got.Labels[0] != "triage" || got.Labels[1] != "backend" {
		t.Fatalf("unexpected labels after SetLabels: %v", got.Labels)
	}
}
