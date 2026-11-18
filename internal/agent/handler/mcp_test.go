package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/kilip/opus/internal/agent/service"
	"github.com/kilip/opus/internal/auth"
	"github.com/kilip/opus/internal/node"
	"github.com/kilip/opus/internal/platform/httpx"
	"github.com/kilip/opus/internal/run"
	"github.com/kilip/opus/internal/session"
	"github.com/kilip/opus/internal/task"
)

type stubNodes struct {
	tokens map[string]*node.NodeRecord
}

func (s *stubNodes) AuthenticateWorker(_ context.Context, cred string) (*node.NodeRecord, error) {
	if n, ok := s.tokens[cred]; ok {
		return n, nil
	}
	return nil, node.ErrUnauthorized
}

type stubAuth struct {
	sessions map[string]uuid.UUID
}

func (s *stubAuth) ValidateSession(_ context.Context, token string) (uuid.UUID, error) {
	if uid, ok := s.sessions[token]; ok {
		return uid, nil
	}
	return uuid.Nil, auth.ErrRateLimited
}

type stubRuns struct {
	runs map[uuid.UUID]run.Run
}

func (s *stubRuns) Get(_ context.Context, id uuid.UUID) (run.Run, error) {
	if r, ok := s.runs[id]; ok {
		return r, nil
	}
	return run.Run{}, run.ErrRunNotFound
}

func (s *stubRuns) SetReport(_ context.Context, id uuid.UUID, report json.RawMessage) (run.Run, error) {
	r, ok := s.runs[id]
	if !ok {
		return run.Run{}, run.ErrRunNotFound
	}
	r.Report = report
	s.runs[id] = r
	return r, nil
}

type stubTasks struct {
	tasks map[string]task.Record
}

func (s *stubTasks) Get(_ context.Context, id string) (task.Record, error) {
	if t, ok := s.tasks[id]; ok {
		return t, nil
	}
	return task.Record{}, task.ErrTaskNotFound
}

func (s *stubTasks) GetByKey(_ context.Context, key string) (task.Record, error) {
	for _, t := range s.tasks {
		if strings.EqualFold(t.Key, key) {
			return t, nil
		}
	}
	return task.Record{}, task.ErrTaskNotFound
}

func (s *stubTasks) List(_ context.Context, filters ...task.TaskFilter) ([]task.Record, error) {
	var pid string
	if len(filters) > 0 {
		pid = filters[0].ProjectID
	}
	var out []task.Record
	for _, t := range s.tasks {
		if pid != "" && t.ProjectID != pid {
			continue
		}
		out = append(out, t)
	}
	return out, nil
}

func (s *stubTasks) Update(_ context.Context, id string, req task.UpdateTaskRequest) (task.Record, error) {
	t, ok := s.tasks[id]
	if !ok {
		return task.Record{}, task.ErrTaskNotFound
	}
	if req.Status != nil {
		t.Status = *req.Status
	}
	s.tasks[id] = t
	return t, nil
}

func (s *stubTasks) ListActivity(_ context.Context, _ string) ([]task.Activity, error) {
	return nil, nil
}

func (s *stubTasks) AddActivity(_ context.Context, _ string, kind task.ActivityKind, author, body string) (task.Activity, error) {
	return task.Activity{
		ID:     uuid.NewString(),
		Kind:   kind,
		Author: author,
		Body:   body,
	}, nil
}

type stubSessions struct{}

func (s *stubSessions) RecordEvent(_ context.Context, sessionID uuid.UUID, kind string, payload json.RawMessage) (session.SessionEvent, error) {
	return session.SessionEvent{
		ID:        uuid.New(),
		SessionID: sessionID,
		Kind:      kind,
		Payload:   payload,
	}, nil
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func mcpCall(t *testing.T, app *fiber.App, headers map[string]string, cookies []*http.Cookie, body any) (*http.Response, rpcResponse) {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	for _, ck := range cookies {
		req.AddCookie(ck)
	}
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	var rpc rpcResponse
	if resp.StatusCode == http.StatusOK {
		if err := json.NewDecoder(resp.Body).Decode(&rpc); err != nil {
			t.Fatalf("decode rpc response: %v", err)
		}
	}
	return resp, rpc
}

func TestMCPEndpointAuthAndTools(t *testing.T) {
	t.Parallel()

	nodeA := uuid.New()
	nodeB := uuid.New()
	projA := uuid.New()
	projB := uuid.New()
	taskA := uuid.New()
	taskB := uuid.New()
	runA := uuid.New()
	runB := uuid.New()
	sessA := uuid.New()
	userID := uuid.New()

	nodes := &stubNodes{
		tokens: map[string]*node.NodeRecord{
			"worker-secret-a": {ID: nodeA.String(), Role: node.RoleWorker},
		},
	}
	authStore := &stubAuth{
		sessions: map[string]uuid.UUID{
			"valid-session-cookie": userID,
		},
	}
	runs := &stubRuns{
		runs: map[uuid.UUID]run.Run{
			runA: {
				ID:        runA,
				TaskID:    taskA,
				ProjectID: projA,
				NodeID:    &nodeA,
				SessionID: &sessA,
			},
			runB: {
				ID:        runB,
				TaskID:    taskB,
				ProjectID: projB,
				NodeID:    &nodeB,
			},
		},
	}
	tasks := &stubTasks{
		tasks: map[string]task.Record{
			taskA.String(): {
				ID:        taskA.String(),
				Key:       "OP-134",
				ProjectID: projA.String(),
				Title:     "Streamable HTTP MCP",
				Status:    task.StatusInProgress,
				Priority:  task.PriorityP1,
				Type:      task.TypeFeat,
			},
			taskB.String(): {
				ID:        taskB.String(),
				Key:       "OP-999",
				ProjectID: projB.String(),
				Title:     "Foreign task",
				Status:    task.StatusTodo,
				Priority:  task.PriorityP2,
				Type:      task.TypeChore,
			},
		},
	}

	svc := service.New(service.Config{
		Tasks:    tasks,
		Runs:     runs,
		Sessions: &stubSessions{},
	})
	h := New(Config{
		Service: svc,
		Nodes:   nodes,
		Auth:    authStore,
		Runs:    runs,
	})

	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(slog.Default())})
	h.Register(app)

	t.Run("rejects unauthenticated requests with 401", func(t *testing.T) {
		resp, _ := mcpCall(t, app, nil, nil, map[string]any{
			"jsonrpc": "2.0",
			"id":      1,
			"method":  "tools/list",
		})
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", resp.StatusCode)
		}

		respBad, _ := mcpCall(t, app, map[string]string{
			"Authorization": "Bearer wrong-token",
		}, nil, map[string]any{
			"jsonrpc": "2.0",
			"id":      2,
			"method":  "tools/list",
		})
		if respBad.StatusCode != http.StatusUnauthorized {
			t.Fatalf("bad bearer status = %d, want 401", respBad.StatusCode)
		}
	})

	t.Run("rejects foreign run ownership with 403", func(t *testing.T) {
		resp, _ := mcpCall(t, app, map[string]string{
			"Authorization": "Bearer worker-secret-a",
			"X-Opus-Run-ID": runB.String(),
		}, nil, map[string]any{
			"jsonrpc": "2.0",
			"id":      3,
			"method":  "tools/list",
		})
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("foreign run status = %d, want 403", resp.StatusCode)
		}
	})

	t.Run("lists all 5 tools and calls get_task with Bearer node auth", func(t *testing.T) {
		headers := map[string]string{
			"Authorization":     "Bearer worker-secret-a",
			"X-Opus-Run-ID":     runA.String(),
			"X-Opus-Session-ID": sessA.String(),
			"X-Opus-Project-ID": projA.String(),
		}

		resp, rpc := mcpCall(t, app, headers, nil, map[string]any{
			"jsonrpc": "2.0",
			"id":      10,
			"method":  "tools/list",
		})
		if resp.StatusCode != http.StatusOK || rpc.Error != nil {
			t.Fatalf("tools/list failed: status=%d err=%+v", resp.StatusCode, rpc.Error)
		}
		var listRes struct {
			Tools []struct {
				Name string `json:"name"`
			} `json:"tools"`
		}
		if err := json.Unmarshal(rpc.Result, &listRes); err != nil {
			t.Fatalf("unmarshal tools/list: %v", err)
		}
		if len(listRes.Tools) != 5 {
			t.Fatalf("expected 5 tools, got %d", len(listRes.Tools))
		}

		// Call get_task defaulting to bound run task.
		respCall, rpcCall := mcpCall(t, app, headers, nil, map[string]any{
			"jsonrpc": "2.0",
			"id":      11,
			"method":  "tools/call",
			"params": map[string]any{
				"name":      "get_task",
				"arguments": map[string]any{},
			},
		})
		if respCall.StatusCode != http.StatusOK || rpcCall.Error != nil {
			t.Fatalf("get_task failed: status=%d err=%+v", respCall.StatusCode, rpcCall.Error)
		}
		var toolRes struct {
			IsError           bool `json:"isError"`
			StructuredContent struct {
				Key string `json:"key"`
			} `json:"structuredContent"`
		}
		if err := json.Unmarshal(rpcCall.Result, &toolRes); err != nil {
			t.Fatalf("unmarshal get_task result: %v", err)
		}
		if toolRes.IsError || toolRes.StructuredContent.Key != "OP-134" {
			t.Fatalf("unexpected get_task result: %+v", toolRes)
		}

		// Cross-project read returns isError: true with forbidden_scope.
		_, rpcForbidden := mcpCall(t, app, headers, nil, map[string]any{
			"jsonrpc": "2.0",
			"id":      12,
			"method":  "tools/call",
			"params": map[string]any{
				"name":      "get_task",
				"arguments": map[string]any{"ref": "OP-999"},
			},
		})
		var errRes struct {
			IsError bool `json:"isError"`
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
		}
		if err := json.Unmarshal(rpcForbidden.Result, &errRes); err != nil {
			t.Fatalf("unmarshal forbidden result: %v", err)
		}
		if !errRes.IsError || len(errRes.Content) == 0 || !strings.Contains(errRes.Content[0].Text, "forbidden_scope") {
			t.Fatalf("expected isError with forbidden_scope, got %+v", errRes)
		}
	})

	t.Run("accepts valid user session cookie auth", func(t *testing.T) {
		resp, rpc := mcpCall(t, app, map[string]string{
			"X-Opus-Project-ID": projA.String(),
		}, []*http.Cookie{
			{Name: auth.SessionCookie, Value: "valid-session-cookie"},
		}, map[string]any{
			"jsonrpc": "2.0",
			"id":      20,
			"method":  "tools/call",
			"params": map[string]any{
				"name":      "list_tasks",
				"arguments": map[string]any{},
			},
		})
		if resp.StatusCode != http.StatusOK || rpc.Error != nil {
			t.Fatalf("cookie list_tasks failed: status=%d err=%+v", resp.StatusCode, rpc.Error)
		}
		var res struct {
			IsError           bool `json:"isError"`
			StructuredContent struct {
				Total int `json:"total"`
			} `json:"structuredContent"`
		}
		if err := json.Unmarshal(rpc.Result, &res); err != nil {
			t.Fatalf("unmarshal cookie result: %v", err)
		}
		if res.IsError || res.StructuredContent.Total != 1 {
			t.Fatalf("unexpected cookie list_tasks result: %+v", res)
		}
	})
}
