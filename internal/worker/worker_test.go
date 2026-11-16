package worker_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"io"
	"log/slog"
	"net"
	"strings"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/gofiber/fiber/v3"
	"github.com/kilip/opus/ent"
	"github.com/kilip/opus/ent/enttest"
	entnode "github.com/kilip/opus/ent/node"
	"github.com/kilip/opus/internal/node"
	"github.com/kilip/opus/internal/platform/httpx"
	"github.com/kilip/opus/internal/worker"
	_ "modernc.org/sqlite"
)

func setupClient(t *testing.T) *ent.Client {
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

func setupTestServer(t *testing.T, client *ent.Client) (*fiber.App, *node.Node) {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	app := fiber.New(fiber.Config{
		ErrorHandler: httpx.ErrorHandler(logger),
	})

	nodeFeature := node.New(client, node.Config{
		BaseURL: "http://localhost:7880",
	})
	nodeFeature.RegisterPublic(app.Group("/api"))
	return app, nodeFeature
}

func seedWorker(t *testing.T, client *ent.Client, name, cred string) *ent.Node {
	t.Helper()
	h := sha256.Sum256([]byte(cred))
	n, err := client.Node.Create().
		SetName(name).
		SetRole(entnode.RoleWorker).
		SetStatus(entnode.StatusOffline).
		SetCredentialHash(hex.EncodeToString(h[:])).
		Save(context.Background())
	if err != nil {
		t.Fatalf("seed worker %s: %v", name, err)
	}
	return n
}

// AC1: worker.New(cfg) and Run(ctx) exist; Run returns nil when ctx is cancelled.
func TestWorker_Run_CancelReturnsNil(t *testing.T) {
	w := worker.New(worker.Config{
		MasterURL:  "http://127.0.0.1:54321",
		NodeID:     "test-node-cancel",
		Credential: "some-cred",
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := w.Run(ctx)
	if err != nil {
		t.Errorf("expected nil error on canceled context, got: %v", err)
	}
}

// AC2: Authenticates (Authorization: Bearer <credential>), sends hello then heartbeat,
// and logs received command messages.
func TestWorker_RealServer_HelloHeartbeatCommand(t *testing.T) {
	client := setupClient(t)
	app, nodeFeature := setupTestServer(t, client)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}
	defer func() { _ = ln.Close() }()

	go func() {
		_ = app.Listener(ln)
	}()

	workerCred := "secret-worker-key"
	workerNode := seedWorker(t, client, "worker-real-1", workerCred)

	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, nil))

	commandCh := make(chan node.Envelope, 1)
	connectedCh := make(chan struct{}, 1)

	w := worker.New(worker.Config{
		MasterURL:         "http://" + ln.Addr().String(),
		NodeID:            workerNode.ID.String(),
		Credential:        workerCred,
		HeartbeatInterval: 30 * time.Millisecond,
		Logger:            logger,
		InitialBackoff:    30 * time.Millisecond,
		OnCommand: func(env node.Envelope) {
			commandCh <- env
		},
		OnConnect: func() {
			select {
			case connectedCh <- struct{}{}:
			default:
			}
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runErrCh := make(chan error, 1)
	go func() {
		runErrCh <- w.Run(ctx)
	}()

	// Wait for connected
	select {
	case <-connectedCh:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for worker OnConnect")
	}

	// Wait for hub connection tracking
	deadline := time.Now().Add(2 * time.Second)
	for !nodeFeature.Hub().Connected(workerNode.ID.String()) {
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for nodeFeature.Hub().Connected")
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Verify worker is online in DB (heartbeat/hello recorded)
	time.Sleep(80 * time.Millisecond)
	wn, err := client.Node.Get(context.Background(), workerNode.ID)
	if err != nil {
		t.Fatalf("get node failed: %v", err)
	}
	if wn.Status != entnode.StatusOnline {
		t.Errorf("expected node status online, got %s", wn.Status)
	}

	// Dispatch command from master server
	cmdEnv := node.Envelope{
		Type:    node.EnvelopeTypeCommand,
		ID:      "cmd-step-94",
		Payload: []byte(`{"action":"sync"}`),
	}
	if err := nodeFeature.Hub().Send(workerNode.ID.String(), cmdEnv); err != nil {
		t.Fatalf("send command failed: %v", err)
	}

	// Verify worker received and logged command
	select {
	case received := <-commandCh:
		if received.ID != "cmd-step-94" {
			t.Errorf("expected command id cmd-step-94, got %s", received.ID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for command on worker")
	}

	if !strings.Contains(logBuf.String(), "received command") {
		t.Errorf("expected logs to contain 'received command', got: %s", logBuf.String())
	}

	// Worker sends event back to master
	eventReceived := make(chan node.Envelope, 1)
	nodeFeature.Hub().OnEvent(func(nodeID string, env node.Envelope) {
		if nodeID == workerNode.ID.String() {
			select {
			case eventReceived <- env:
			default:
			}
		}
	})

	eventEnv := node.Envelope{
		Type:    node.EnvelopeTypeEvent,
		ID:      "event-101",
		Payload: []byte(`{"status":"running"}`),
	}
	if err := w.Send(eventEnv); err != nil {
		t.Fatalf("worker send event failed: %v", err)
	}

	select {
	case received := <-eventReceived:
		if received.ID != "event-101" {
			t.Errorf("expected event id event-101, got %s", received.ID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for event on hub")
	}

	// Cancel context and assert clean exit
	cancel()
	select {
	case err := <-runErrCh:
		if err != nil {
			t.Errorf("expected nil error on cancel, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for w.Run to exit")
	}
}

// AC3: When connection drops, client reconnects with exponential backoff capped at 30s;
// test restarts the server and asserts reconnection.
func TestWorker_ReconnectOnServerRestart(t *testing.T) {
	client := setupClient(t)
	app1, nodeFeature1 := setupTestServer(t, client)

	ln1, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}
	serverAddr := ln1.Addr().String()

	go func() {
		_ = app1.Listener(ln1)
	}()

	workerCred := "reconnect-secret"
	workerNode := seedWorker(t, client, "worker-reconnect", workerCred)

	reconnectedCh := make(chan struct{}, 5)

	w := worker.New(worker.Config{
		MasterURL:         "http://" + serverAddr,
		NodeID:            workerNode.ID.String(),
		Credential:        workerCred,
		HeartbeatInterval: 30 * time.Millisecond,
		InitialBackoff:    30 * time.Millisecond,
		MaxBackoff:        100 * time.Millisecond,
		OnConnect: func() {
			reconnectedCh <- struct{}{}
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runErrCh := make(chan error, 1)
	go func() {
		runErrCh <- w.Run(ctx)
	}()

	// 1. Wait for initial connection
	select {
	case <-reconnectedCh:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for initial connection")
	}

	for !nodeFeature1.Hub().Connected(workerNode.ID.String()) {
		time.Sleep(10 * time.Millisecond)
	}

	// 2. Shut down first server
	nodeFeature1.Stop()
	_ = app1.Shutdown()
	_ = ln1.Close()

	// Wait for disconnect in first hub
	deadline := time.Now().Add(2 * time.Second)
	for nodeFeature1.Hub().Connected(workerNode.ID.String()) {
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for hub1 to disconnect")
		}
		time.Sleep(10 * time.Millisecond)
	}

	// 3. Restart server on same address
	time.Sleep(50 * time.Millisecond)
	var ln2 net.Listener
	for attempt := 0; attempt < 20; attempt++ {
		ln2, err = net.Listen("tcp", serverAddr)
		if err == nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("re-listen on %s failed: %v", serverAddr, err)
	}
	defer func() { _ = ln2.Close() }()

	app2, nodeFeature2 := setupTestServer(t, client)
	go func() {
		_ = app2.Listener(ln2)
	}()

	// 4. Assert worker automatically reconnects to server 2
	select {
	case <-reconnectedCh:
	case <-time.After(4 * time.Second):
		t.Fatal("timed out waiting for worker reconnection to server 2")
	}

	deadline = time.Now().Add(3 * time.Second)
	for !nodeFeature2.Hub().Connected(workerNode.ID.String()) {
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for hub2 to record connected worker")
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Cancel and assert clean exit
	cancel()
	select {
	case err := <-runErrCh:
		if err != nil {
			t.Errorf("expected nil error on cancel, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for Run to exit")
	}
}

// AC3: Backoff values are constants with a ponytail: comment.
func TestBackoff_ConstantsAndCalculation(t *testing.T) {
	if worker.DefaultHeartbeatInterval != 15*time.Second {
		t.Errorf("expected 15s heartbeat interval, got %v", worker.DefaultHeartbeatInterval)
	}
	if worker.DefaultInitialBackoff != 1*time.Second {
		t.Errorf("expected 1s initial backoff, got %v", worker.DefaultInitialBackoff)
	}
	if worker.DefaultMaxBackoff != 30*time.Second {
		t.Errorf("expected 30s max backoff, got %v", worker.DefaultMaxBackoff)
	}

	// Test backoff progression
	current := worker.DefaultInitialBackoff // 1s
	expected := []time.Duration{
		2 * time.Second,
		4 * time.Second,
		8 * time.Second,
		16 * time.Second,
		30 * time.Second, // 32s capped at 30s
		30 * time.Second, // capped at 30s
	}

	for i, exp := range expected {
		current = worker.NextBackoff(current, worker.DefaultMaxBackoff)
		if current != exp {
			t.Errorf("step %d: expected %v, got %v", i, exp, current)
		}
	}
}

// AC4: Wrong credential stops reconnecting and Run returns a clear error (no infinite retry on 401).
func TestWorker_WrongCredential_StopsAndReturns401(t *testing.T) {
	client := setupClient(t)
	app, _ := setupTestServer(t, client)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}
	defer func() { _ = ln.Close() }()

	go func() {
		_ = app.Listener(ln)
	}()

	workerNode := seedWorker(t, client, "worker-wrong-cred", "correct-credential")

	w := worker.New(worker.Config{
		MasterURL:  "http://" + ln.Addr().String(),
		NodeID:     workerNode.ID.String(),
		Credential: "invalid-credential-here",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err = w.Run(ctx)
	if err == nil {
		t.Fatal("expected error with wrong credential, got nil")
	}

	if !errors.Is(err, node.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}

	if errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected immediate stop on 401, but timed out (infinite retry)")
	}
}

// AC5: Master node's in-process path carries the same envelope types through Go channels,
// with no socket; a test sends a command and an event through it.
func TestInProcessTransport_CommandAndEvent(t *testing.T) {
	transport := worker.NewInProcess()
	defer func() { _ = transport.Close() }()

	ctx := context.Background()

	// 1. Send command through transport
	cmdEnv := node.Envelope{
		Type:    node.EnvelopeTypeCommand,
		ID:      "cmd-test-1",
		Payload: []byte(`{"action":"ping"}`),
	}
	if err := transport.SendCommand(ctx, cmdEnv); err != nil {
		t.Fatalf("send command failed: %v", err)
	}

	select {
	case received := <-transport.Commands:
		if received.Type != node.EnvelopeTypeCommand || received.ID != "cmd-test-1" {
			t.Errorf("unexpected command: %+v", received)
		}
	default:
		t.Fatal("expected command in transport.Commands")
	}

	// 2. Send event through transport
	evtEnv := node.Envelope{
		Type:    node.EnvelopeTypeEvent,
		ID:      "evt-test-1",
		Payload: []byte(`{"status":"pong"}`),
	}
	if err := transport.SendEvent(ctx, evtEnv); err != nil {
		t.Fatalf("send event failed: %v", err)
	}

	select {
	case received := <-transport.Events:
		if received.Type != node.EnvelopeTypeEvent || received.ID != "evt-test-1" {
			t.Errorf("unexpected event: %+v", received)
		}
	default:
		t.Fatal("expected event in transport.Events")
	}

	// 3. Send hello and heartbeat
	helloEnv := node.Envelope{Type: node.EnvelopeTypeHello, ID: "hello-inproc"}
	if err := transport.SendEvent(ctx, helloEnv); err != nil {
		t.Fatalf("send hello failed: %v", err)
	}
	<-transport.Events

	hbEnv := node.Envelope{Type: node.EnvelopeTypeHeartbeat, ID: "hb-inproc"}
	if err := transport.SendEvent(ctx, hbEnv); err != nil {
		t.Fatalf("send heartbeat failed: %v", err)
	}
	<-transport.Events

	// 4. Closed transport rejects sends
	_ = transport.Close()
	if !transport.Closed() {
		t.Error("expected transport.Closed() to be true")
	}

	if err := transport.SendCommand(ctx, cmdEnv); !errors.Is(err, worker.ErrTransportClosed) {
		t.Errorf("expected ErrTransportClosed on SendCommand, got %v", err)
	}
	if err := transport.SendEvent(ctx, evtEnv); !errors.Is(err, worker.ErrTransportClosed) {
		t.Errorf("expected ErrTransportClosed on SendEvent, got %v", err)
	}
}

// AC5: Worker running with InProcessTransport carries the same envelope types.
func TestWorker_InProcess_Lifecycle(t *testing.T) {
	transport := worker.NewInProcess()
	defer func() { _ = transport.Close() }()

	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, nil))

	commandCh := make(chan node.Envelope, 1)

	w := worker.New(worker.Config{
		NodeID:            "master-node-01",
		InProcess:         transport,
		HeartbeatInterval: 30 * time.Millisecond,
		Logger:            logger,
		OnCommand: func(env node.Envelope) {
			commandCh <- env
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runErrCh := make(chan error, 1)
	go func() {
		runErrCh <- w.Run(ctx)
	}()

	// 1. Worker automatically sends hello to master on transport.Events
	select {
	case hello := <-transport.Events:
		if hello.Type != node.EnvelopeTypeHello {
			t.Errorf("expected hello envelope, got %s", hello.Type)
		}
		if hello.ID != "hello-master-node-01" {
			t.Errorf("expected hello id hello-master-node-01, got %s", hello.ID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for in-process hello envelope")
	}

	// 2. Worker sends heartbeat periodically on transport.Events
	select {
	case hb := <-transport.Events:
		if hb.Type != node.EnvelopeTypeHeartbeat {
			t.Errorf("expected heartbeat envelope, got %s", hb.Type)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for in-process heartbeat envelope")
	}

	// 3. Master sends command to worker on transport.Commands
	cmd := node.Envelope{
		Type:    node.EnvelopeTypeCommand,
		ID:      "cmd-inproc-42",
		Payload: []byte(`{"action":"run"}`),
	}
	if err := transport.SendCommand(ctx, cmd); err != nil {
		t.Fatalf("send inproc command failed: %v", err)
	}

	select {
	case received := <-commandCh:
		if received.ID != "cmd-inproc-42" {
			t.Errorf("expected command id cmd-inproc-42, got %s", received.ID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for worker to receive inproc command")
	}

	if !strings.Contains(logBuf.String(), "received command") {
		t.Errorf("expected logs to contain 'received command', got: %s", logBuf.String())
	}

	// 4. Worker sends event using w.Send()
	event := node.Envelope{
		Type:    node.EnvelopeTypeEvent,
		ID:      "evt-inproc-99",
		Payload: []byte(`{"state":"done"}`),
	}
	if err := w.Send(event); err != nil {
		t.Fatalf("worker send inproc event failed: %v", err)
	}

	select {
	case received := <-transport.Events:
		for received.Type == node.EnvelopeTypeHeartbeat {
			received = <-transport.Events
		}
		if received.ID != "evt-inproc-99" {
			t.Errorf("expected event id evt-inproc-99, got %s", received.ID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for event on transport.Events")
	}

	// 5. Cancel context and assert clean exit
	cancel()
	select {
	case err := <-runErrCh:
		if err != nil {
			t.Errorf("expected nil error on cancel, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for Run to exit")
	}
}
