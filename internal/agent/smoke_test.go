package agent_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/kilip/opus/ent"
	"github.com/kilip/opus/ent/enttest"
	"github.com/kilip/opus/internal/agent"
	"github.com/kilip/opus/internal/node"
	"github.com/kilip/opus/internal/platform/eventbus"
	"github.com/kilip/opus/internal/platform/httpx"
	"github.com/kilip/opus/internal/project"
	"github.com/kilip/opus/internal/run"
	"github.com/kilip/opus/internal/session"
	"github.com/kilip/opus/internal/task"
	_ "modernc.org/sqlite"
)

func setupSmokeDB(t *testing.T) *ent.Client {
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

type smokeRPCResp struct {
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func postMCP(t *testing.T, app *fiber.App, headers map[string]string, body any) smokeRPCResp {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal rpc body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST /mcp status = %d, want 200", resp.StatusCode)
	}
	var out smokeRPCResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode rpc response: %v", err)
	}
	if out.Error != nil {
		t.Fatalf("unexpected rpc error: %+v", out.Error)
	}
	return out
}

func TestAgentMCP_EndToEndSmoke(t *testing.T) {
	ctx := context.Background()
	client := setupSmokeDB(t)
	bus := eventbus.New()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	nodeFeature := node.New(client, node.Config{BaseURL: "http://127.0.0.1:7880", Bus: bus})
	projectFeature := project.New(client)
	taskFeature := task.New(client, bus, task.WithProjectChecker(projectFeature))
	sessionFeature := session.New(client, session.Config{Bus: bus})
	runFeature := run.New(client, run.Config{
		Bus:      bus,
		Tasks:    taskFeature,
		Projects: projectFeature,
		Sessions: sessionFeature,
		Logger:   logger,
	})
	agentFeature := agent.New(agent.Config{
		Tasks:    taskFeature,
		Runs:     runFeature,
		Sessions: sessionFeature,
		Nodes:    nodeFeature,
	})

	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(logger)})
	agentFeature.Register(app)

	// Register a worker node via join token + Join.
	tok, err := nodeFeature.GenerateJoinToken(ctx)
	if err != nil {
		t.Fatalf("GenerateJoinToken: %v", err)
	}
	rawToken := strings.TrimPrefix(tok.URL, "http://127.0.0.1:7880/join/")
	joined, err := nodeFeature.Join(ctx, node.JoinNodeRequest{
		Token: rawToken,
		Name:  "smoke-worker",
	})
	if err != nil {
		t.Fatalf("Join: %v", err)
	}
	nodeUUID := uuid.MustParse(joined.NodeID)

	// Create bound Project, Task, Session, and Run.
	projRow, err := client.Project.Create().
		SetName("Opus").
		SetSlug("opus").
		SetRepoProvider("github").
		SetRepoRef("kilip/opus").
		SetTrackerProvider("github").
		SetTrackerRef("kilip/opus").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	taskRec, err := taskFeature.Import(ctx, task.ImportTaskRequest{
		ProjectID:   projRow.ID.String(),
		Title:       "Smoke test task",
		Description: "Verify end-to-end MCP tool flow",
		Source:      task.SourceGitHub,
		ExternalID:  "kilip/opus#138",
		Status:      task.StatusTodo,
		Type:        task.TypeChore,
		Priority:    task.PriorityP2,
	})
	if err != nil {
		t.Fatalf("import task: %v", err)
	}
	taskUUID := uuid.MustParse(taskRec.ID)

	sessRec, err := sessionFeature.Enqueue(ctx, session.CreateInput{
		TaskID:   taskUUID,
		NodeID:   nodeUUID,
		Provider: "agy",
	})
	if err != nil {
		t.Fatalf("enqueue session: %v", err)
	}

	runRec, err := runFeature.Create(ctx, run.CreateInput{
		TaskID:    taskUUID,
		ProjectID: projRow.ID,
		Mode:      run.ModeVibe,
	})
	if err != nil {
		t.Fatalf("create run: %v", err)
	}
	if _, err := runFeature.Transition(ctx, runRec.ID, run.StatePreparing, run.TransitionInput{
		NodeID: &nodeUUID,
	}); err != nil {
		t.Fatalf("transition preparing: %v", err)
	}
	if _, err := runFeature.Transition(ctx, runRec.ID, run.StateRunning, run.TransitionInput{
		SessionID: &sessRec.ID,
	}); err != nil {
		t.Fatalf("transition running: %v", err)
	}

	headers := map[string]string{
		"Authorization":     "Bearer " + joined.Credential,
		"X-Opus-Run-ID":     runRec.ID.String(),
		"X-Opus-Session-ID": sessRec.ID.String(),
		"X-Opus-Project-ID": projRow.ID.String(),
	}

	// 1. get_task
	getResp := postMCP(t, app, headers, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "get_task",
			"arguments": map[string]any{"ref": "#138"},
		},
	})
	var getOut struct {
		IsError           bool `json:"isError"`
		StructuredContent struct {
			Key   string `json:"key"`
			Title string `json:"title"`
		} `json:"structuredContent"`
	}
	if err := json.Unmarshal(getResp.Result, &getOut); err != nil {
		t.Fatalf("unmarshal get_task: %v", err)
	}
	if getOut.IsError || getOut.StructuredContent.Key != taskRec.Key || !strings.Contains(getOut.StructuredContent.Title, "<untrusted-task-data>") {
		t.Fatalf("unexpected get_task output: %+v", getOut)
	}

	// 2. set_status
	setResp := postMCP(t, app, headers, map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "set_status",
			"arguments": map[string]any{"status": "in-progress"},
		},
	})
	var setOut struct {
		IsError bool `json:"isError"`
	}
	if err := json.Unmarshal(setResp.Result, &setOut); err != nil || setOut.IsError {
		t.Fatalf("set_status failed: %v %+v", err, setOut)
	}

	// 3. add_activity
	actResp := postMCP(t, app, headers, map[string]any{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "add_activity",
			"arguments": map[string]any{"body": "Smoke test progress <!-- stripped -->"},
		},
	})
	var actOut struct {
		IsError bool `json:"isError"`
	}
	if err := json.Unmarshal(actResp.Result, &actOut); err != nil || actOut.IsError {
		t.Fatalf("add_activity failed: %v %+v", err, actOut)
	}

	// 4. submit_report
	repResp := postMCP(t, app, headers, map[string]any{
		"jsonrpc": "2.0",
		"id":      4,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "submit_report",
			"arguments": map[string]any{
				"summary":    "End-to-end smoke test completed",
				"pr_url":     "https://github.com/kilip/opus/pull/183",
				"commit_sha": "abc1234",
				"acceptance_criteria": []map[string]any{
					{"id": "AC1", "criterion": "E2E smoke", "met": true, "evidence": "go test passed"},
				},
			},
		},
	})
	var repOut struct {
		IsError bool `json:"isError"`
	}
	if err := json.Unmarshal(repResp.Result, &repOut); err != nil || repOut.IsError {
		t.Fatalf("submit_report failed: %v %+v", err, repOut)
	}

	// Assert DB/facade state:
	updatedTask, err := taskFeature.Get(ctx, taskRec.ID)
	if err != nil {
		t.Fatalf("taskFeature.Get: %v", err)
	}
	if updatedTask.Status != task.StatusInProgress {
		t.Fatalf("task status = %q, want in-progress", updatedTask.Status)
	}

	activities, err := taskFeature.ListActivity(ctx, taskRec.ID)
	if err != nil {
		t.Fatalf("taskFeature.ListActivity: %v", err)
	}
	var vibeCount int
	for _, a := range activities {
		if a.Author == "vibe" {
			vibeCount++
		}
	}
	if vibeCount < 2 {
		t.Fatalf("expected >= 2 vibe-authored activities (add_activity + submit_report), got %d", vibeCount)
	}

	updatedRun, err := runFeature.Get(ctx, runRec.ID)
	if err != nil {
		t.Fatalf("runFeature.Get: %v", err)
	}
	var persisted agent.Report
	if err := json.Unmarshal(updatedRun.Report, &persisted); err != nil {
		t.Fatalf("unmarshal Run.Report: %v", err)
	}
	if persisted.Summary != "End-to-end smoke test completed" || persisted.CommitSHA != "abc1234" {
		t.Fatalf("unexpected persisted Run.Report: %+v", persisted)
	}

	events, err := sessionFeature.ListEventsAfter(ctx, sessRec.ID, 0)
	if err != nil {
		t.Fatalf("sessionFeature.ListEventsAfter: %v", err)
	}
	var toolCallCount int
	for _, e := range events {
		if e.Kind == "tool_call" {
			toolCallCount++
		}
	}
	if toolCallCount != 4 {
		t.Fatalf("expected 4 tool_call SessionEvents, got %d", toolCallCount)
	}
}
