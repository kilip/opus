package repo_test

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"

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

// AC2: Keys are OP-1, OP-2, ... from task_key counter, allocated in same transaction as insert.
// Test runs 20 concurrent creates and asserts keys OP-1..OP-20 with no gaps or duplicates.
func TestEntRepo_ConcurrentCreateKeys(t *testing.T) {
	client := setupTestClient(t)
	r := repo.New(client)
	ctx := context.Background()

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
				Title: fmt.Sprintf("Concurrent task %d", idx),
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

	// 1. Defaults
	tsk, err := r.Create(ctx, repo.CreateInput{
		Title: "Test default task",
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	if tsk.Status != domain.TaskStatusTodo {
		t.Errorf("status = %q, want %q", tsk.Status, domain.TaskStatusTodo)
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
	newStatus := domain.TaskStatusInProgress
	upRes, err := r.Update(ctx, tsk.ID, repo.UpdateInput{
		Status: &newStatus,
	})
	if err != nil {
		t.Fatalf("update status failed: %v", err)
	}
	if !upRes.StatusChanged {
		t.Errorf("expected StatusChanged = true")
	}
	if upRes.PrevStatus != domain.TaskStatusTodo {
		t.Errorf("PrevStatus = %q, want %q", upRes.PrevStatus, domain.TaskStatusTodo)
	}
	if upRes.Task.Status != domain.TaskStatusInProgress {
		t.Errorf("Task.Status = %q, want %q", upRes.Task.Status, domain.TaskStatusInProgress)
	}

	// 3. Assign and unassign
	assignee := &domain.Assignee{ID: "toni", Name: "Toni"}
	upRes, err = r.Update(ctx, tsk.ID, repo.UpdateInput{
		Assignee: assignee,
	})
	if err != nil {
		t.Fatalf("update assignee failed: %v", err)
	}
	if upRes.Task.Assignee == nil || upRes.Task.Assignee.Name != "Toni" {
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

	// 4. AddActivity
	addedAct, err := r.AddActivity(ctx, tsk.ID, domain.ActivityKindComment, "ignored", "A custom comment")
	if err != nil {
		t.Fatalf("add activity failed: %v", err)
	}
	if addedAct.Body != "A custom comment" {
		t.Errorf("added act body = %q, want 'A custom comment'", addedAct.Body)
	}
	// Verify added activity also stored author as NULL in Ent
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
}

// AC4: (source, external_id) is unique: duplicate pair fails, while two tasks with NULL external_id both succeed.
func TestEntRepo_ExternalIDUniqueness(t *testing.T) {
	client := setupTestClient(t)
	r := repo.New(client)
	ctx := context.Background()

	// Two tasks with empty external_id (stored as NULL in DB) should both succeed
	t1, err := r.Create(ctx, repo.CreateInput{
		Source:     domain.TaskSourceManual,
		ExternalID: "",
		Title:      "Task without external id 1",
	})
	if err != nil {
		t.Fatalf("first task with empty external_id failed: %v", err)
	}

	t2, err := r.Create(ctx, repo.CreateInput{
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
		Source:     domain.TaskSourceGitHub,
		ExternalID: "issue-42",
		Title:      "GitHub Issue 42",
	})
	if err != nil {
		t.Fatalf("create github task failed: %v", err)
	}

	// Duplicate pair must fail
	_, err = r.Create(ctx, repo.CreateInput{
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

	// Create multiple tasks
	var createdIDs []string
	for i := 1; i <= 5; i++ {
		tsk, err := r.Create(ctx, repo.CreateInput{
			Title: fmt.Sprintf("Task %d", i),
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
