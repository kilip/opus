package notify_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kilip/opus/internal/node"
	"github.com/kilip/opus/internal/notify"
	"github.com/kilip/opus/internal/platform/eventbus"
	"github.com/kilip/opus/internal/run"
)

type capturedRequest struct {
	Path   string
	ChatID string
	Text   string
}

func TestTelegram_RunFinishedAndNodeOffline(t *testing.T) {
	ctx := context.Background()
	var (
		mu       sync.Mutex
		requests []capturedRequest
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body struct {
			ChatID string `json:"chat_id"`
			Text   string `json:"text"`
		}
		_ = json.Unmarshal(raw, &body)
		mu.Lock()
		requests = append(requests, capturedRequest{
			Path:   r.URL.Path,
			ChatID: body.ChatID,
			Text:   body.Text,
		})
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	bus := eventbus.New()
	t.Cleanup(func() { _ = bus.Close(ctx) })

	n := notify.New(notify.Config{
		BotToken: "test-token",
		ChatID:   "chat-123",
		BaseURL:  srv.URL,
	})
	if err := n.Register(bus); err != nil {
		t.Fatalf("Register: %v", err)
	}

	runID := uuid.New()
	taskID := uuid.New()
	finishedPayload, _ := json.Marshal(run.RunFinishedPayload{
		RunID:   runID,
		TaskID:  taskID,
		Mode:    run.ModeVibe,
		Outcome: run.StateFailed,
		Reason:  "build failed",
	})
	if err := bus.Publish(ctx, eventbus.Event{
		ID:      uuid.NewString(),
		Type:    run.EventFinished,
		Subject: runID.String(),
		At:      time.Now(),
		Payload: finishedPayload,
	}); err != nil {
		t.Fatalf("Publish run.finished: %v", err)
	}

	offlinePayload, _ := json.Marshal(node.NodeEventPayload{
		ID:   "node-1",
		Name: "worker-alpha",
	})
	if err := bus.Publish(ctx, eventbus.Event{
		ID:      uuid.NewString(),
		Type:    node.EventOffline,
		Subject: "node-1",
		At:      time.Now(),
		Payload: offlinePayload,
	}); err != nil {
		t.Fatalf("Publish node.offline: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		count := len(requests)
		mu.Unlock()
		if count >= 2 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(requests) != 2 {
		t.Fatalf("expected 2 telegram requests, got %d", len(requests))
	}

	if requests[0].Path != "/bottest-token/sendMessage" || requests[0].ChatID != "chat-123" {
		t.Fatalf("unexpected request[0] metadata: %+v", requests[0])
	}
	if !strings.Contains(requests[0].Text, runID.String()) ||
		!strings.Contains(requests[0].Text, string(run.StateFailed)) ||
		!strings.Contains(requests[0].Text, "build failed") {
		t.Fatalf("unexpected run.finished text: %q", requests[0].Text)
	}

	if requests[1].Path != "/bottest-token/sendMessage" || requests[1].ChatID != "chat-123" {
		t.Fatalf("unexpected request[1] metadata: %+v", requests[1])
	}
	if !strings.Contains(requests[1].Text, "worker-alpha") || !strings.Contains(requests[1].Text, "node-1") {
		t.Fatalf("unexpected node.offline text: %q", requests[1].Text)
	}
}

func TestTelegram_DisabledWhenEnvEmpty(t *testing.T) {
	ctx := context.Background()
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cases := []struct {
		name  string
		token string
		chat  string
	}{
		{name: "both empty", token: "", chat: ""},
		{name: "missing token", token: "", chat: "123"},
		{name: "missing chat", token: "tok", chat: ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bus := eventbus.New()
			t.Cleanup(func() { _ = bus.Close(ctx) })

			n := notify.New(notify.Config{
				BotToken: tc.token,
				ChatID:   tc.chat,
				BaseURL:  srv.URL,
			})
			if n.Enabled() {
				t.Fatalf("expected Enabled() = false")
			}
			if err := n.Register(bus); err != nil {
				t.Fatalf("Register: %v", err)
			}

			payload, _ := json.Marshal(node.NodeEventPayload{ID: "n1", Name: "alpha"})
			if err := n.HandleEvent(ctx, eventbus.Event{
				Type:    node.EventOffline,
				Payload: payload,
			}); err != nil {
				t.Fatalf("HandleEvent: %v", err)
			}
			if called {
				t.Fatalf("expected no HTTP calls when disabled")
			}
		})
	}
}

func TestTelegram_DeliveryErrorLogsAndDoesNotBlock(t *testing.T) {
	ctx := context.Background()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, nil))

	n := notify.New(notify.Config{
		BotToken: "tok",
		ChatID:   "chat",
		BaseURL:  srv.URL,
		Logger:   logger,
	})

	payload, _ := json.Marshal(node.NodeEventPayload{ID: "n1", Name: "alpha"})
	err := n.HandleEvent(ctx, eventbus.Event{
		Type:    node.EventOffline,
		Payload: payload,
	})
	if err != nil {
		t.Fatalf("expected HandleEvent to swallow delivery error, got %v", err)
	}
	if !strings.Contains(logBuf.String(), "telegram notification failed") {
		t.Fatalf("expected error log, got %q", logBuf.String())
	}
}
