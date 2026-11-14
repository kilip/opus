package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/opus/internal/task/handler"
)

func get(t *testing.T, path string) *http.Response {
	t.Helper()

	app := fiber.New()
	handler.New().Register(app.Group("/api"))

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, path, nil))
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	return resp
}

func decode(t *testing.T, resp *http.Response, v any) {
	t.Helper()
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		t.Fatalf("decode body: %v", err)
	}
}

func TestList(t *testing.T) {
	resp := get(t, "/api/tasks")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var list handler.TaskList
	decode(t, resp, &list)
	if list.Total != len(list.Items) || list.Total == 0 {
		t.Fatalf("total = %d, items = %d", list.Total, len(list.Items))
	}
	for _, task := range list.Items {
		if task.Labels == nil {
			t.Errorf("%s: labels must serialise as [], not null", task.ID)
		}
	}
}

func TestGet(t *testing.T) {
	resp := get(t, "/api/tasks/t3")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var task handler.Task
	decode(t, resp, &task)
	if task.ID != "t3" || task.Key != "OPUS-3" || task.Status != handler.TaskStatusInProgress {
		t.Fatalf("unexpected task: %+v", task)
	}
}

func TestGetNotFound(t *testing.T) {
	resp := get(t, "/api/tasks/nope")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}

	var body handler.ErrorResponse
	decode(t, resp, &body)
	if body.Message == "" {
		t.Fatal("error message must not be empty")
	}
}

func TestActivity(t *testing.T) {
	resp := get(t, "/api/tasks/t2/activity")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var list handler.ActivityList
	decode(t, resp, &list)
	if len(list.Items) != 2 {
		t.Fatalf("items = %d, want 2", len(list.Items))
	}
}

func TestActivityEmptyIsArray(t *testing.T) {
	resp := get(t, "/api/tasks/t9/activity")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var list handler.ActivityList
	decode(t, resp, &list)
	if list.Items == nil {
		t.Fatal("items must serialise as [], not null")
	}
}

func TestActivityNotFound(t *testing.T) {
	if got := get(t, "/api/tasks/nope/activity").StatusCode; got != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", got)
	}
}
