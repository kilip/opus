package handler_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/gofiber/fiber/v3"
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
	app, _ := setupApp(t)
	resp := req(t, app, http.MethodGet, "/api/tasks", nil)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	envelope := decode[httpx.Response[[]domain.Task]](t, resp)
	if envelope.Meta == nil || envelope.Meta.Total == nil {
		t.Fatal("expected meta.total to be present")
	}
	if *envelope.Meta.Total != 0 {
		t.Fatalf("expected total 0, got %d", *envelope.Meta.Total)
	}
	if len(envelope.Data) != 0 {
		t.Fatalf("expected 0 tasks in data, got %d", len(envelope.Data))
	}
}

func TestGetTask(t *testing.T) {
	app, _ := setupApp(t)

	// Create task first
	respCreate := req(t, app, http.MethodPost, "/api/tasks", domain.CreateTaskRequest{
		Title: "Write PRD mission control",
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
	app, _ := setupApp(t)

	// 400 Bad Request if title empty
	resp := req(t, app, http.MethodPost, "/api/tasks", domain.CreateTaskRequest{
		Title: "   ",
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
	status := domain.TaskStatusTodo
	resp2 := req(t, app, http.MethodPost, "/api/tasks", domain.CreateTaskRequest{
		Title:  title,
		Status: &status,
	})
	if resp2.StatusCode != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", resp2.StatusCode)
	}
	created := decode[httpx.Response[domain.Task]](t, resp2)
	if created.Data.Key != "OP-1" {
		t.Fatalf("expected task ID OP-1, got %s", created.Data.Key)
	}
	if created.Data.Title != title {
		t.Fatalf("expected title %s, got %s", title, created.Data.Title)
	}
}

func TestUpdateTask(t *testing.T) {
	app, _ := setupApp(t)

	// Create task first
	initStatus := domain.TaskStatusTodo
	respCreate := req(t, app, http.MethodPost, "/api/tasks", domain.CreateTaskRequest{
		Title:  "Task for update",
		Status: &initStatus,
	})
	if respCreate.StatusCode != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", respCreate.StatusCode)
	}
	created := decode[httpx.Response[domain.Task]](t, respCreate)

	// 400 on empty title update
	emptyTitle := ""
	resp := req(t, app, http.MethodPatch, "/api/tasks/"+created.Data.ID, domain.UpdateTaskRequest{
		Title: &emptyTitle,
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", resp.StatusCode)
	}

	// 200 on stage update + appends activity
	newStatus := domain.TaskStatusInProgress
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

	// Verify activity list has stage transition appended
	resp3 := req(t, app, http.MethodGet, "/api/tasks/"+created.Data.ID+"/activity", nil)
	if resp3.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp3.StatusCode)
	}
	actEnvelope := decode[httpx.Response[[]domain.Activity]](t, resp3)
	if len(actEnvelope.Data) != 2 {
		t.Fatalf("expected 2 activities after stage update, got %d", len(actEnvelope.Data))
	}
	lastAct := actEnvelope.Data[1]
	if lastAct.Kind != domain.ActivityKindStatus {
		t.Fatalf("expected activity kind status, got %s", lastAct.Kind)
	}
	expectedBody := "moved stage todo → in-progress"
	if lastAct.Body != expectedBody {
		t.Fatalf("expected body %q, got %q", expectedBody, lastAct.Body)
	}
}

func TestListTaskActivity(t *testing.T) {
	app, _ := setupApp(t)

	// 404 for missing task
	resp := req(t, app, http.MethodGet, "/api/tasks/nonexistent/activity", nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", resp.StatusCode)
	}

	// 200 for existing task with activity
	respCreate := req(t, app, http.MethodPost, "/api/tasks", domain.CreateTaskRequest{
		Title: "Task with activity",
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
	app, _ := setupApp(t)

	// Create task first
	respCreate := req(t, app, http.MethodPost, "/api/tasks", domain.CreateTaskRequest{
		Title: "Task for activity",
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
