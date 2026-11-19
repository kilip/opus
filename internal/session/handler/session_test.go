package handler_test

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/kilip/opus/ent"
	"github.com/kilip/opus/ent/enttest"
	"github.com/kilip/opus/internal/platform/httpx"
	"github.com/kilip/opus/internal/session/domain"
	"github.com/kilip/opus/internal/session/handler"
	"github.com/kilip/opus/internal/session/repo"
	"github.com/kilip/opus/internal/session/service"
	_ "modernc.org/sqlite"
)

func setupHandlerTest(t *testing.T) (*fiber.App, *service.Service) {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+t.Name()+"?mode=memory&cache=shared&_pragma=foreign_keys(ON)&_pragma=busy_timeout(10000)")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db.SetMaxOpenConns(1)
	drv := entsql.OpenDB(dialect.SQLite, db)
	client := enttest.NewClient(t, enttest.WithOptions(ent.Driver(drv)))
	t.Cleanup(func() {
		_ = client.Close()
	})

	svc := service.New(repo.New(client), service.Config{})
	app := fiber.New(fiber.Config{
		ErrorHandler: httpx.ErrorHandler(slog.New(slog.NewTextHandler(io.Discard, nil))),
	})
	handler.New(svc).Register(app.Group("/api"))
	return app, svc
}

func TestHandler_ListAndGet(t *testing.T) {
	ctx := context.Background()
	app, svc := setupHandlerTest(t)

	task1 := uuid.New()
	task2 := uuid.New()
	node1 := uuid.New()

	s1, _ := svc.Enqueue(ctx, domain.CreateInput{TaskID: task1, NodeID: node1, Provider: "agy"})
	s2, _ := svc.Enqueue(ctx, domain.CreateInput{TaskID: task2, NodeID: node1, Provider: "agy"})

	// Filter by taskId & status.
	req := httptest.NewRequest(http.MethodGet, "/api/sessions?taskId="+task1.String()+"&status=running", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test list: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var listEnv struct {
		Data []domain.SessionResponse `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&listEnv); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(listEnv.Data) != 1 || listEnv.Data[0].ID != s1.ID {
		t.Fatalf("expected only s1, got %+v", listEnv.Data)
	}

	// Get s2.
	req2 := httptest.NewRequest(http.MethodGet, "/api/sessions/"+s2.ID.String(), nil)
	resp2, err := app.Test(req2)
	if err != nil {
		t.Fatalf("app.Test get: %v", err)
	}
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp2.StatusCode)
	}
}

func TestHandler_SSE_HistoryLiveAndDisconnectSafe(t *testing.T) {
	ctx := context.Background()
	app, svc := setupHandlerTest(t)

	sess, err := svc.Enqueue(ctx, domain.CreateInput{
		TaskID:   uuid.New(),
		NodeID:   uuid.New(),
		Provider: "agy",
	})
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	// Record 2 historical events before SSE connects.
	_, _ = svc.RecordEvent(ctx, sess.ID, "chunk", json.RawMessage(`{"text":"one"}`))
	_, _ = svc.RecordEvent(ctx, sess.ID, "chunk", json.RawMessage(`{"text":"two"}`))

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer func() { _ = ln.Close() }()
	go func() {
		_ = app.Listener(ln)
	}()
	defer func() {
		_ = app.Shutdown()
	}()

	baseURL := "http://" + ln.Addr().String()

	// 1. Connect with after_seq=1: should receive seq 2 from history, then live seq 3,
	// and closing client mid-stream must NOT cancel/stop the session.
	reqCtx, reqCancel := context.WithCancel(ctx)
	req, _ := http.NewRequestWithContext(reqCtx, http.MethodGet, baseURL+"/api/sessions/"+sess.ID.String()+"/events?after_seq=1", nil)
	httpResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("SSE GET: %v", err)
	}
	if httpResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", httpResp.StatusCode)
	}

	reader := bufio.NewReader(httpResp.Body)
	readEvent := func() (string, string) {
		var idLine, dataLine string
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				t.Fatalf("read SSE line: %v", err)
			}
			line = strings.TrimSpace(line)
			if line == "" {
				if idLine != "" || dataLine != "" {
					return idLine, dataLine
				}
				continue
			}
			if strings.HasPrefix(line, "id: ") {
				idLine = strings.TrimPrefix(line, "id: ")
			} else if strings.HasPrefix(line, "data: ") {
				dataLine = strings.TrimPrefix(line, "data: ")
			}
		}
	}

	id2, _ := readEvent()
	if id2 != "2" {
		t.Fatalf("expected historical event id 2 after after_seq=1, got %q", id2)
	}

	// Emit live event seq 3.
	_, _ = svc.RecordEvent(ctx, sess.ID, "chunk", json.RawMessage(`{"text":"three"}`))
	id3, _ := readEvent()
	if id3 != "3" {
		t.Fatalf("expected live event id 3, got %q", id3)
	}

	// Disconnect client mid-stream.
	reqCancel()
	_ = httpResp.Body.Close()

	// Give server a moment to process disconnect, then verify session is still running.
	time.Sleep(50 * time.Millisecond)
	gotSess, err := svc.Get(ctx, sess.ID)
	if err != nil {
		t.Fatalf("Get after SSE disconnect: %v", err)
	}
	if gotSess.Status != domain.StatusRunning {
		t.Fatalf("expected session to remain running after SSE client disconnect, got %s", gotSess.Status)
	}

	// 2. Connect a second SSE client and complete the session; stream should emit terminal event and close.
	req2, _ := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/api/sessions/"+sess.ID.String()+"/events?after_seq=2", nil)
	httpResp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatalf("SSE GET 2: %v", err)
	}
	defer func() { _ = httpResp2.Body.Close() }()

	go func() {
		time.Sleep(30 * time.Millisecond)
		_, _, _ = svc.Complete(ctx, sess.ID)
	}()

	bodyBytes, err := io.ReadAll(httpResp2.Body)
	if err != nil {
		t.Fatalf("ReadAll SSE 2: %v", err)
	}
	bodyStr := string(bodyBytes)
	if !strings.Contains(bodyStr, "id: 3") || !strings.Contains(bodyStr, "event: session.completed") {
		t.Fatalf("expected seq 3 and terminal session.completed event, got:\n%s", bodyStr)
	}
}

func TestSessionHandler_SSEKeepaliveAndShutdownClose(t *testing.T) {
	ctx := context.Background()
	_, svc := setupHandlerTest(t)
	h := handler.New(svc)
	h.SetKeepaliveInterval(20 * time.Millisecond)
	app := fiber.New(fiber.Config{
		ErrorHandler: httpx.ErrorHandler(slog.New(slog.NewTextHandler(io.Discard, nil))),
	})
	h.Register(app.Group("/api"))

	sess, err := svc.Enqueue(ctx, domain.CreateInput{
		TaskID:   uuid.New(),
		NodeID:   uuid.New(),
		Provider: "agy",
	})
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}
	go func() { _ = app.Listener(ln) }()
	t.Cleanup(func() { _ = app.Shutdown() })

	baseURL := "http://" + ln.Addr().String()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/api/sessions/"+sess.ID.String()+"/events", nil)
	httpResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("SSE GET: %v", err)
	}
	defer func() { _ = httpResp.Body.Close() }()

	go func() {
		time.Sleep(80 * time.Millisecond)
		svc.Close()
	}()

	bodyBytes, err := io.ReadAll(httpResp.Body)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if !strings.Contains(string(bodyBytes), ": keepalive\n\n") {
		t.Fatalf("expected SSE keepalive comment, got:\n%s", string(bodyBytes))
	}
}
