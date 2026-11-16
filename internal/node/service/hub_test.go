package service_test

import (
	"errors"
	"sync"
	"testing"

	"github.com/kilip/opus/internal/node/domain"
	"github.com/kilip/opus/internal/node/service"
)

type mockConn struct {
	mu      sync.Mutex
	written []any
	closed  bool
}

func (m *mockConn) WriteJSON(v any) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.written = append(m.written, v)
	return nil
}

func (m *mockConn) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

func (m *mockConn) isClosed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.closed
}

func (m *mockConn) getWritten() []any {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]any, len(m.written))
	copy(out, m.written)
	return out
}

func TestHub_Connected(t *testing.T) {
	hub := service.NewHub()
	nodeID := "node-1"

	if hub.Connected(nodeID) {
		t.Fatal("expected Connected to be false initially")
	}

	conn := &mockConn{}
	cleanup, err := hub.Register(nodeID, conn)
	if err != nil {
		t.Fatalf("unexpected register error: %v", err)
	}

	if !hub.Connected(nodeID) {
		t.Fatal("expected Connected to be true after register")
	}

	cleanup()

	if hub.Connected(nodeID) {
		t.Fatal("expected Connected to be false after cleanup")
	}
}

func TestHub_ConnectionReplacement(t *testing.T) {
	hub := service.NewHub()
	nodeID := "node-1"

	conn1 := &mockConn{}
	cleanup1, err := hub.Register(nodeID, conn1)
	if err != nil {
		t.Fatalf("register conn1 failed: %v", err)
	}

	conn2 := &mockConn{}
	cleanup2, err := hub.Register(nodeID, conn2)
	if err != nil {
		t.Fatalf("register conn2 failed: %v", err)
	}

	// conn1 must be closed upon replacement
	if !conn1.isClosed() {
		t.Fatal("expected conn1 to be closed after replacement")
	}
	// conn2 must remain open
	if conn2.isClosed() {
		t.Fatal("expected conn2 to be open")
	}

	// Calling cleanup1 (from the old conn) should NOT remove conn2
	cleanup1()
	if !hub.Connected(nodeID) {
		t.Fatal("expected node to still be connected after old cleanup1 called")
	}

	// Calling cleanup2 removes the active connection
	cleanup2()
	if hub.Connected(nodeID) {
		t.Fatal("expected node to be disconnected after cleanup2")
	}
}

func TestHub_Send(t *testing.T) {
	hub := service.NewHub()
	nodeID := "worker-1"

	cmd := domain.Envelope{
		Type: domain.EnvelopeTypeCommand,
		ID:   "cmd-1",
	}

	// Send to unconnected node returns ErrNodeNotConnected
	err := hub.Send(nodeID, cmd)
	if !errors.Is(err, domain.ErrNodeNotConnected) {
		t.Fatalf("expected ErrNodeNotConnected, got %v", err)
	}

	conn := &mockConn{}
	cleanup, err := hub.Register(nodeID, conn)
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}
	defer cleanup()

	// Send to connected node succeeds
	err = hub.Send(nodeID, cmd)
	if err != nil {
		t.Fatalf("unexpected send error: %v", err)
	}

	written := conn.getWritten()
	if len(written) != 1 {
		t.Fatalf("expected 1 written envelope, got %d", len(written))
	}
	env, ok := written[0].(domain.Envelope)
	if !ok || env.ID != "cmd-1" {
		t.Fatalf("expected envelope with ID cmd-1, got %+v", written[0])
	}

	// Send unknown envelope type returns ErrUnknownEnvelopeType
	err = hub.Send(nodeID, domain.Envelope{Type: "invalid"})
	if !errors.Is(err, domain.ErrUnknownEnvelopeType) {
		t.Fatalf("expected ErrUnknownEnvelopeType, got %v", err)
	}
}

func TestHub_EventsAndUnknownTypeHandling(t *testing.T) {
	hub := service.NewHub()
	nodeID := "worker-1"

	conn := &mockConn{}
	cleanup, err := hub.Register(nodeID, conn)
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}
	defer cleanup()

	var receivedEvents []domain.Envelope
	hub.OnEvent(func(nID string, env domain.Envelope) {
		if nID == nodeID {
			receivedEvents = append(receivedEvents, env)
		}
	})

	// 1. Valid event envelope reaches listener
	eventMsg := []byte(`{"type":"event","id":"evt-1","payload":{"result":"success"}}`)
	err = hub.HandleMessage(nodeID, eventMsg)
	if err != nil {
		t.Fatalf("handle event message failed: %v", err)
	}
	if len(receivedEvents) != 1 || receivedEvents[0].ID != "evt-1" {
		t.Fatalf("expected 1 event with ID evt-1, got %+v", receivedEvents)
	}

	// 2. Hello message handled cleanly without triggering event listener
	helloMsg := []byte(`{"type":"hello"}`)
	err = hub.HandleMessage(nodeID, helloMsg)
	if err != nil {
		t.Fatalf("handle hello message failed: %v", err)
	}
	if len(receivedEvents) != 1 {
		t.Fatalf("expected still 1 event, got %d", len(receivedEvents))
	}

	// 3. Unknown envelope type is rejected, but hub and connection remain alive
	unknownMsg := []byte(`{"type":"unknown_custom_type"}`)
	err = hub.HandleMessage(nodeID, unknownMsg)
	if !errors.Is(err, domain.ErrUnknownEnvelopeType) {
		t.Fatalf("expected ErrUnknownEnvelopeType, got %v", err)
	}

	// Hub connection is STILL active
	if !hub.Connected(nodeID) {
		t.Fatal("expected hub connection to remain alive after unknown envelope type")
	}
}

func TestHub_MasterConnectForbidden(t *testing.T) {
	masterID := "master-node-uuid"
	hub := service.NewHub(masterID)

	conn := &mockConn{}
	_, err := hub.Register(masterID, conn)
	if !errors.Is(err, domain.ErrMasterConnectForbidden) {
		t.Fatalf("expected ErrMasterConnectForbidden, got %v", err)
	}
}

func TestHub_HeartbeatDispatch(t *testing.T) {
	hub := service.NewHub()
	nodeID := "worker-1"

	conn := &mockConn{}
	cleanup, err := hub.Register(nodeID, conn)
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}
	defer cleanup()

	var (
		mu                 sync.Mutex
		heartbeatEnvelopes []domain.Envelope
	)
	hub.OnHeartbeat(func(nID string, env domain.Envelope) {
		if nID == nodeID {
			mu.Lock()
			heartbeatEnvelopes = append(heartbeatEnvelopes, env)
			mu.Unlock()
		}
	})

	// Dispatch hello
	helloMsg := []byte(`{"type":"hello"}`)
	if err := hub.HandleMessage(nodeID, helloMsg); err != nil {
		t.Fatalf("handle hello failed: %v", err)
	}

	// Dispatch heartbeat
	heartbeatMsg := []byte(`{"type":"heartbeat"}`)
	if err := hub.HandleMessage(nodeID, heartbeatMsg); err != nil {
		t.Fatalf("handle heartbeat failed: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(heartbeatEnvelopes) != 2 {
		t.Fatalf("expected 2 heartbeat envelopes, got %d", len(heartbeatEnvelopes))
	}
	if heartbeatEnvelopes[0].Type != domain.EnvelopeTypeHello {
		t.Errorf("expected first envelope type hello, got %s", heartbeatEnvelopes[0].Type)
	}
	if heartbeatEnvelopes[1].Type != domain.EnvelopeTypeHeartbeat {
		t.Errorf("expected second envelope type heartbeat, got %s", heartbeatEnvelopes[1].Type)
	}
}

func TestHub_OnDisconnect_Callback(t *testing.T) {
	hub := service.NewHub()
	nodeID := "worker-1"

	var (
		mu           sync.Mutex
		disconnected []string
	)
	hub.OnDisconnect(func(nID string) {
		mu.Lock()
		disconnected = append(disconnected, nID)
		mu.Unlock()
	})

	conn1 := &mockConn{}
	cleanup1, err := hub.Register(nodeID, conn1)
	if err != nil {
		t.Fatalf("register conn1 failed: %v", err)
	}

	// Replace conn1 with conn2
	conn2 := &mockConn{}
	cleanup2, err := hub.Register(nodeID, conn2)
	if err != nil {
		t.Fatalf("register conn2 failed: %v", err)
	}

	// Calling cleanup1 from older connection should NOT fire disconnect
	cleanup1()

	mu.Lock()
	if len(disconnected) != 0 {
		t.Fatalf("expected 0 disconnect calls after old cleanup1, got %d", len(disconnected))
	}
	mu.Unlock()

	// Calling cleanup2 for the active connection fires disconnect once
	cleanup2()

	mu.Lock()
	if len(disconnected) != 1 || disconnected[0] != nodeID {
		t.Fatalf("expected 1 disconnect call for %s, got %v", nodeID, disconnected)
	}
	mu.Unlock()

	// Calling cleanup2 again does not fire disconnect again
	cleanup2()
	mu.Lock()
	if len(disconnected) != 1 {
		t.Fatalf("expected still 1 disconnect call, got %d", len(disconnected))
	}
	mu.Unlock()
}

func TestHub_Disconnect(t *testing.T) {
	hub := service.NewHub()
	nodeID := "node-disc-test"
	conn := &mockConn{}

	_, err := hub.Register(nodeID, conn)
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	if err := hub.Disconnect(nodeID); err != nil {
		t.Fatalf("disconnect failed: %v", err)
	}
	if !conn.isClosed() {
		t.Error("expected conn to be closed")
	}

	// Disconnecting unmapped node returns ErrNodeNotConnected
	if err := hub.Disconnect("unknown-node"); !errors.Is(err, domain.ErrNodeNotConnected) {
		t.Errorf("expected ErrNodeNotConnected, got %v", err)
	}
}

func TestHub_Close(t *testing.T) {
	hub := service.NewHub()
	conn1 := &mockConn{}
	conn2 := &mockConn{}

	_, _ = hub.Register("node-c1", conn1)
	_, _ = hub.Register("node-c2", conn2)

	if err := hub.Close(); err != nil {
		t.Fatalf("hub close failed: %v", err)
	}

	if !conn1.isClosed() {
		t.Error("expected conn1 to be closed")
	}
	if !conn2.isClosed() {
		t.Error("expected conn2 to be closed")
	}
}
