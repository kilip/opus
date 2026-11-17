package handler_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/kilip/opus/ent"
	"github.com/kilip/opus/ent/enttest"
	"github.com/kilip/opus/internal/platform/httpx"
	"github.com/kilip/opus/internal/task/domain"
	"github.com/kilip/opus/internal/task/handler"
	"github.com/kilip/opus/internal/task/repo"
	"github.com/kilip/opus/internal/task/service"
	_ "modernc.org/sqlite"
)

func setupApp(t *testing.T) (*fiber.App, *ent.Client) {
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

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	app := fiber.New(fiber.Config{
		ErrorHandler: httpx.ErrorHandler(logger),
	})

	r := repo.New(client)
	h := handler.New(service.New(r, nil))
	h.Register(app.Group("/api"))
	return app, client
}

func createTestProject(t *testing.T, client *ent.Client) string {
	t.Helper()
	p, err := client.Project.Create().
		SetSlug(fmt.Sprintf("handler-test-proj-%s", uuid.New().String()[:8])).
		SetName("Test Project").
		SetRepoProvider("github").
		SetRepoRef("org/repo").
		SetTrackerProvider("github").
		SetTrackerRef("org/repo").
		Save(context.Background())
	if err != nil {
		t.Fatalf("create test project failed: %v", err)
	}
	return p.ID.String()
}

func req(t *testing.T, app *fiber.App, method, path string, body any) *http.Response {
	t.Helper()

	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	r := httptest.NewRequest(method, path, bodyReader)
	if body != nil {
		r.Header.Set("Content-Type", "application/json")
	}

	resp, err := app.Test(r)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	return resp
}

func decode[T any](t *testing.T, resp *http.Response) T {
	t.Helper()
	var v T
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	return v
}

func TestListTasks(t *testing.T) {
	app, client := setupApp(t)
	p1 := createTestProject(t, client)
	p2 := createTestProject(t, client)

	// Empty list initially
	resp := req(t, app, http.MethodGet, "/api/tasks", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
	envelope := decode[httpx.Response[[]domain.Task]](t, resp)
	if *envelope.Meta.Total != 0 || len(envelope.Data) != 0 {
		t.Fatalf("expected 0 tasks initially")
	}

	// Create 2 tasks in p1, 1 task in p2
	_ = req(t, app, http.MethodPost, "/api/tasks", domain.CreateTaskRequest{
		ProjectID: p1,
		Title:     "P1 Task 1",
	})
	_ = req(t, app, http.MethodPost, "/api/tasks", domain.CreateTaskRequest{
		ProjectID: p1,
		Title:     "P1 Task 2",
	})
	_ = req(t, app, http.MethodPost, "/api/tasks", domain.CreateTaskRequest{
		ProjectID: p2,
		Title:     "P2 Task 1",
	})

	// Unfiltered list returns all 3 tasks
	respAll := req(t, app, http.MethodGet, "/api/tasks", nil)
	if respAll.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", respAll.StatusCode)
	}
	envAll := decode[httpx.Response[[]domain.Task]](t, respAll)
	if len(envAll.Data) != 3 {
		t.Fatalf("expected 3 tasks in unfiltered list, got %d", len(envAll.Data))
	}

	// Filtered list by p1 returns 2 tasks (AC3)
	respP1 := req(t, app, http.MethodGet, "/api/tasks?projectId="+p1, nil)
	if respP1.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", respP1.StatusCode)
	}
	envP1 := decode[httpx.Response[[]domain.Task]](t, respP1)
	if len(envP1.Data) != 2 {
		t.Fatalf("expected 2 tasks for p1, got %d", len(envP1.Data))
	}
	for _, tsk := range envP1.Data {
		if tsk.ProjectID != p1 {
			t.Fatalf("expected project ID %s, got %s", p1, tsk.ProjectID)
		}
	}

	// Malformed projectId query param returns 400 (AC3)
	respBad := req(t, app, http.MethodGet, "/api/tasks?projectId=not-a-uuid", nil)
	if respBad.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400 for malformed projectId, got %d", respBad.StatusCode)
	}
}

func TestGetTask(t *testing.T) {
	app, client := setupApp(t)
	pID := createTestProject(t, client)

	// Create task first
	respCreate := req(t, app, http.MethodPost, "/api/tasks", domain.CreateTaskRequest{
		ProjectID: pID,
		Title:     "Write PRD mission control",
	})
	if respCreate.StatusCode != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", respCreate.StatusCode)
	}
	created := decode[httpx.Response[domain.Task]](t, respCreate)

	resp := req(t, app, http.MethodGet, "/api/tasks/"+created.Data.ID, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	envelope := decode[httpx.Response[domain.Task]](t, resp)
	if envelope.Data.ID != created.Data.ID {
		t.Fatalf("expected task ID %s, got %s", created.Data.ID, envelope.Data.ID)
	}
	if envelope.Data.ProjectID != pID {
		t.Fatalf("expected project ID %s, got %s", pID, envelope.Data.ProjectID)
	}
	if envelope.Data.Title != "Write PRD mission control" {
		t.Fatalf("unexpected title: %s", envelope.Data.Title)
	}
}

func TestGetTaskNotFound(t *testing.T) {
	app, _ := setupApp(t)
	resp := req(t, app, http.MethodGet, "/api/tasks/nonexistent", nil)

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", resp.StatusCode)
	}

	var errResp httpx.ErrorResponse
	if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
		t.Fatalf("decode error response: %v", err)
	}

	if errResp.Error != "TASK_NOT_FOUND" {
		t.Fatalf("expected error TASK_NOT_FOUND, got %s", errResp.Error)
	}
	if errResp.Data != nil {
		t.Fatalf("expected nil data in error response, got %v", errResp.Data)
	}
}

func TestCreateTask(t *testing.T) {
	app, client := setupApp(t)
	pID := createTestProject(t, client)

	// 400 Bad Request if projectId missing (AC2)
	respNoProj := req(t, app, http.MethodPost, "/api/tasks", domain.CreateTaskRequest{
		Title: "Task without project",
	})
	if respNoProj.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400 for missing projectId, got %d", respNoProj.StatusCode)
	}
	errRespNoProj := decode[httpx.ErrorResponse](t, respNoProj)
	if errRespNoProj.Error != "PROJECT_ID_REQUIRED" {
		t.Fatalf("expected PROJECT_ID_REQUIRED, got %s", errRespNoProj.Error)
	}

	// 400 Bad Request if projectId invalid UUID (AC2)
	respInvalidProj := req(t, app, http.MethodPost, "/api/tasks", domain.CreateTaskRequest{
		ProjectID: "invalid-uuid",
		Title:     "Task with invalid project",
	})
	if respInvalidProj.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400 for invalid projectId, got %d", respInvalidProj.StatusCode)
	}

	// 404 Not Found if projectId unknown (AC2)
	respUnknownProj := req(t, app, http.MethodPost, "/api/tasks", domain.CreateTaskRequest{
		ProjectID: uuid.New().String(),
		Title:     "Task with unknown project",
	})
	if respUnknownProj.StatusCode != http.StatusNotFound {
		t.Fatalf("expected status 404 for unknown project, got %d", respUnknownProj.StatusCode)
	}

	// 400 Bad Request if title empty
	resp := req(t, app, http.MethodPost, "/api/tasks", domain.CreateTaskRequest{
		ProjectID: pID,
		Title:     "   ",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", resp.StatusCode)
	}
	errResp := decode[httpx.ErrorResponse](t, resp)
	if errResp.Error != "TASK_TITLE_REQUIRED" {
		t.Fatalf("expected error TASK_TITLE_REQUIRED, got %s", errResp.Error)
	}

	// 201 Created on valid task
	title := "New test task"
	status := domain.StatusTodo
	resp2 := req(t, app, http.MethodPost, "/api/tasks", domain.CreateTaskRequest{
		ProjectID: pID,
		Title:     title,
		Status:    &status,
	})
	if resp2.StatusCode != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", resp2.StatusCode)
	}
	created := decode[httpx.Response[domain.Task]](t, resp2)
	if created.Data.Key != "OP-1" {
		t.Fatalf("expected task ID OP-1, got %s", created.Data.Key)
	}
	if created.Data.ProjectID != pID {
		t.Fatalf("expected project ID %s, got %s", pID, created.Data.ProjectID)
	}
	if created.Data.Title != title {
		t.Fatalf("expected title %s, got %s", title, created.Data.Title)
	}

	// 400 Bad Request on invalid status
	badStatus := domain.TaskStatus("bogus")
	respBadStatus := req(t, app, http.MethodPost, "/api/tasks", domain.CreateTaskRequest{
		ProjectID: pID,
		Title:     "Bad status task",
		Status:    &badStatus,
	})
	if respBadStatus.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400 for invalid status, got %d", respBadStatus.StatusCode)
	}
	errBadStatus := decode[httpx.ErrorResponse](t, respBadStatus)
	if errBadStatus.Error != "TASK_INVALID_STATUS" {
		t.Fatalf("expected TASK_INVALID_STATUS, got %s", errBadStatus.Error)
	}

	// 201 Created with blocked status
	blockedStatus := domain.StatusBlocked
	respBlocked := req(t, app, http.MethodPost, "/api/tasks", domain.CreateTaskRequest{
		ProjectID: pID,
		Title:     "Blocked task",
		Status:    &blockedStatus,
	})
	if respBlocked.StatusCode != http.StatusCreated {
		t.Fatalf("expected status 201 for blocked status, got %d", respBlocked.StatusCode)
	}
	createdBlocked := decode[httpx.Response[domain.Task]](t, respBlocked)
	if createdBlocked.Data.Status != domain.StatusBlocked {
		t.Fatalf("expected status %s, got %s", domain.StatusBlocked, createdBlocked.Data.Status)
	}
}

func TestUpdateTask(t *testing.T) {
	app, client := setupApp(t)
	pID := createTestProject(t, client)

	// Create task first
	initStatus := domain.StatusTodo
	respCreate := req(t, app, http.MethodPost, "/api/tasks", domain.CreateTaskRequest{
		ProjectID: pID,
		Title:     "Task for update",
		Status:    &initStatus,
	})
	if respCreate.StatusCode != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", respCreate.StatusCode)
	}
	created := decode[httpx.Response[domain.Task]](t, respCreate)

	// 400 Bad Request if body has projectId (AC2)
	respPatchProj := req(t, app, http.MethodPatch, "/api/tasks/"+created.Data.ID, map[string]any{
		"projectId": pID,
	})
	if respPatchProj.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400 for patch containing projectId, got %d", respPatchProj.StatusCode)
	}

	// 400 Bad Request if body has projectId null (AC2)
	respPatchProjNull := req(t, app, http.MethodPatch, "/api/tasks/"+created.Data.ID, map[string]any{
		"projectId": nil,
	})
	if respPatchProjNull.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400 for patch containing projectId null, got %d", respPatchProjNull.StatusCode)
	}

	// 400 on empty title update
	emptyTitle := ""
	resp := req(t, app, http.MethodPatch, "/api/tasks/"+created.Data.ID, domain.UpdateTaskRequest{
		Title: &emptyTitle,
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", resp.StatusCode)
	}

	// 200 on stage update + appends activity
	newStatus := domain.StatusInProgress
	resp2 := req(t, app, http.MethodPatch, "/api/tasks/"+created.Data.ID, domain.UpdateTaskRequest{
		Status: &newStatus,
	})
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp2.StatusCode)
	}
	updated := decode[httpx.Response[domain.Task]](t, resp2)
	if updated.Data.Status != newStatus {
		t.Fatalf("expected status %s, got %s", newStatus, updated.Data.Status)
	}

	// Transition to blocked + invalid status 400
	blockedStatus := domain.StatusBlocked
	respBlocked := req(t, app, http.MethodPatch, "/api/tasks/"+created.Data.ID, domain.UpdateTaskRequest{
		Status: &blockedStatus,
	})
	if respBlocked.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 for blocked transition, got %d", respBlocked.StatusCode)
	}
	updatedBlocked := decode[httpx.Response[domain.Task]](t, respBlocked)
	if updatedBlocked.Data.Status != domain.StatusBlocked {
		t.Fatalf("expected status %s, got %s", domain.StatusBlocked, updatedBlocked.Data.Status)
	}

	badStatus := domain.TaskStatus("nope")
	respBad := req(t, app, http.MethodPatch, "/api/tasks/"+created.Data.ID, domain.UpdateTaskRequest{
		Status: &badStatus,
	})
	if respBad.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400 for invalid status update, got %d", respBad.StatusCode)
	}

	// Verify activity list has stage transitions appended
	resp3 := req(t, app, http.MethodGet, "/api/tasks/"+created.Data.ID+"/activity", nil)
	if resp3.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp3.StatusCode)
	}
	actEnvelope := decode[httpx.Response[[]domain.Activity]](t, resp3)
	if len(actEnvelope.Data) != 3 {
		t.Fatalf("expected 3 activities after stage updates, got %d", len(actEnvelope.Data))
	}
	lastAct := actEnvelope.Data[1]
	if lastAct.Kind != domain.ActivityKindStatus {
		t.Fatalf("expected activity kind status, got %s", lastAct.Kind)
	}
	expectedBody := "moved stage todo → in-progress"
	if lastAct.Body != expectedBody {
		t.Fatalf("expected body %q, got %q", expectedBody, lastAct.Body)
	}
	if actEnvelope.Data[2].Body != "moved stage in-progress → blocked" {
		t.Fatalf("expected body %q, got %q", "moved stage in-progress → blocked", actEnvelope.Data[2].Body)
	}
}

func TestListTaskActivity(t *testing.T) {
	app, client := setupApp(t)
	pID := createTestProject(t, client)

	// 404 for missing task
	resp := req(t, app, http.MethodGet, "/api/tasks/nonexistent/activity", nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", resp.StatusCode)
	}

	// 200 for existing task with activity
	respCreate := req(t, app, http.MethodPost, "/api/tasks", domain.CreateTaskRequest{
		ProjectID: pID,
		Title:     "Task with activity",
	})
	if respCreate.StatusCode != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", respCreate.StatusCode)
	}
	created := decode[httpx.Response[domain.Task]](t, respCreate)

	resp2 := req(t, app, http.MethodGet, "/api/tasks/"+created.Data.ID+"/activity", nil)
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp2.StatusCode)
	}
	envelope := decode[httpx.Response[[]domain.Activity]](t, resp2)
	if len(envelope.Data) != 1 {
		t.Fatalf("expected 1 activity for created task, got %d", len(envelope.Data))
	}
}

func TestCreateTaskActivity(t *testing.T) {
	app, client := setupApp(t)
	pID := createTestProject(t, client)

	// Create task first
	respCreate := req(t, app, http.MethodPost, "/api/tasks", domain.CreateTaskRequest{
		ProjectID: pID,
		Title:     "Task for activity",
	})
	if respCreate.StatusCode != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", respCreate.StatusCode)
	}
	createdTask := decode[httpx.Response[domain.Task]](t, respCreate)

	// 400 on empty body
	resp := req(t, app, http.MethodPost, "/api/tasks/"+createdTask.Data.ID+"/activity", domain.CreateActivityRequest{
		Body: "   ",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", resp.StatusCode)
	}
	errResp := decode[httpx.ErrorResponse](t, resp)
	if errResp.Error != "TASK_ACTIVITY_BODY_REQUIRED" {
		t.Fatalf("expected error TASK_ACTIVITY_BODY_REQUIRED, got %s", errResp.Error)
	}

	// 404 on missing task
	resp2 := req(t, app, http.MethodPost, "/api/tasks/nonexistent/activity", domain.CreateActivityRequest{
		Body: "test comment",
	})
	if resp2.StatusCode != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", resp2.StatusCode)
	}

	// 400 when author is supplied in HTTP request body (AC4)
	respAuthor := req(t, app, http.MethodPost, "/api/tasks/"+createdTask.Data.ID+"/activity", map[string]any{
		"body":   "Spoofed comment",
		"author": "spoofed-user",
	})
	if respAuthor.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400 when author supplied over HTTP, got %d", respAuthor.StatusCode)
	}
	errAuthorResp := decode[httpx.ErrorResponse](t, respAuthor)
	if errAuthorResp.Error != "TASK_ACTIVITY_AUTHOR_FORBIDDEN" {
		t.Fatalf("expected error TASK_ACTIVITY_AUTHOR_FORBIDDEN, got %s", errAuthorResp.Error)
	}

	// 201 on valid activity
	resp3 := req(t, app, http.MethodPost, "/api/tasks/"+createdTask.Data.ID+"/activity", domain.CreateActivityRequest{
		Body: "New review comment",
	})
	if resp3.StatusCode != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", resp3.StatusCode)
	}
	created := decode[httpx.Response[domain.Activity]](t, resp3)
	if created.Data.Body != "New review comment" {
		t.Fatalf("expected body 'New review comment', got %s", created.Data.Body)
	}
	if created.Data.Author != "" {
		t.Fatalf("expected empty author, got %s", created.Data.Author)
	}
	if created.Data.Kind != domain.ActivityKindComment {
		t.Fatalf("expected default kind 'comment', got %s", created.Data.Kind)
	}
}
