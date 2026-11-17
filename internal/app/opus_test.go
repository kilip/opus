package app_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/gofiber/fiber/v3"
	"github.com/kilip/opus/ent"
	"github.com/kilip/opus/ent/enttest"
	"github.com/kilip/opus/internal/app"
	"github.com/kilip/opus/internal/node"
	"github.com/kilip/opus/internal/platform/eventbus"
	"github.com/kilip/opus/internal/tracker"
	"github.com/kilip/opus/internal/worker"
	_ "modernc.org/sqlite"
)

func TestNewMailer_FallbackToConsole(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn}))

	cfg := app.Config{
		SMTPHost: "",
	}

	m := app.NewMailer(cfg, logger)
	if m == nil {
		t.Fatal("expected non-nil mailer")
	}

	typeName := reflect.TypeOf(m).String()
	if !strings.Contains(typeName, "consoleMailer") {
		t.Errorf("expected consoleMailer, got %s", typeName)
	}

	logOutput := buf.String()
	if !strings.Contains(logOutput, "APP_SMTP_HOST not set, falling back to console mailer") {
		t.Errorf("expected startup warning in log, got: %q", logOutput)
	}
}

func TestNewMailer_FallbackWithNilLogger(t *testing.T) {
	cfg := app.Config{
		SMTPHost: "",
	}
	m := app.NewMailer(cfg, nil)
	if m == nil {
		t.Fatal("expected non-nil mailer even with nil logger")
	}
}

func TestNewMailer_SMTPConfigured(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn}))

	cfg := app.Config{
		SMTPHost: "smtp.example.com",
		SMTPPort: 587,
	}

	m := app.NewMailer(cfg, logger)
	if m == nil {
		t.Fatal("expected non-nil mailer")
	}

	typeName := reflect.TypeOf(m).String()
	if !strings.Contains(typeName, "smtpMailer") {
		t.Errorf("expected smtpMailer, got %s", typeName)
	}

	if buf.Len() > 0 {
		t.Errorf("expected no warning logs when SMTP is configured, got: %q", buf.String())
	}
}

func TestConfig_SMTPDefaults(t *testing.T) {
	cfg, err := app.GetConfig(context.Background())
	if err != nil {
		t.Fatalf("GetConfig returned unexpected error: %v", err)
	}

	if cfg.SMTPPort != 587 {
		t.Errorf("expected default SMTPPort 587, got %d", cfg.SMTPPort)
	}
	if cfg.SMTPHost != "" {
		t.Errorf("expected default SMTPHost to be empty, got %q", cfg.SMTPHost)
	}
	if cfg.SMTPUsername != "" {
		t.Errorf("expected default SMTPUsername to be empty, got %q", cfg.SMTPUsername)
	}
	if cfg.SMTPPassword != "" {
		t.Errorf("expected default SMTPPassword to be empty, got %q", cfg.SMTPPassword)
	}
	if cfg.SMTPFrom != "" {
		t.Errorf("expected default SMTPFrom to be empty, got %q", cfg.SMTPFrom)
	}
}

func TestConfig_OAuthEnv(t *testing.T) {
	t.Run("legacy env vars ignored", func(t *testing.T) {
		legacyVars := []string{
			"SESSION" + "_SECRET",
			"SESSION" + "_TTL",
			"GITHUB" + "_CLIENT_ID",
			"GITHUB" + "_CLIENT_SECRET",
			"GOOGLE" + "_CLIENT_ID",
			"GOOGLE" + "_CLIENT_SECRET",
			"FACEBOOK" + "_CLIENT_ID",
			"FACEBOOK" + "_CLIENT_SECRET",
		}
		for _, env := range legacyVars {
			t.Setenv(env, "legacy-value")
		}

		cfg, err := app.GetConfig(context.Background())
		if err != nil {
			t.Fatalf("GetConfig returned unexpected error: %v", err)
		}

		if cfg.SessionSecret != "" {
			t.Errorf("expected empty SessionSecret, got %q", cfg.SessionSecret)
		}
		if cfg.SessionTTL != 720*time.Hour {
			t.Errorf("expected default SessionTTL 720h, got %v", cfg.SessionTTL)
		}
		if cfg.GitHubClientID != "" || cfg.GitHubClientSecret != "" {
			t.Errorf("expected empty github oauth config, got id=%q secret=%q", cfg.GitHubClientID, cfg.GitHubClientSecret)
		}
		if cfg.GoogleClientID != "" || cfg.GoogleClientSecret != "" {
			t.Errorf("expected empty google oauth config, got id=%q secret=%q", cfg.GoogleClientID, cfg.GoogleClientSecret)
		}
		if cfg.FacebookClientID != "" || cfg.FacebookClientSecret != "" {
			t.Errorf("expected empty facebook oauth config, got id=%q secret=%q", cfg.FacebookClientID, cfg.FacebookClientSecret)
		}
	})

	t.Run("auth env vars populated", func(t *testing.T) {
		t.Setenv("AUTH_SECRET", "test-auth-secret")
		t.Setenv("AUTH_SESSION_TTL", "12h")
		t.Setenv("AUTH_GITHUB_ID", "test-gh-id")
		t.Setenv("AUTH_GITHUB_SECRET", "test-gh-secret")
		t.Setenv("AUTH_GOOGLE_ID", "test-g-id")
		t.Setenv("AUTH_GOOGLE_SECRET", "test-g-secret")
		t.Setenv("AUTH_FACEBOOK_ID", "test-fb-id")
		t.Setenv("AUTH_FACEBOOK_SECRET", "test-fb-secret")

		cfg, err := app.GetConfig(context.Background())
		if err != nil {
			t.Fatalf("GetConfig returned unexpected error: %v", err)
		}

		if cfg.SessionSecret != "test-auth-secret" {
			t.Errorf("expected SessionSecret %q, got %q", "test-auth-secret", cfg.SessionSecret)
		}
		if cfg.SessionTTL != 12*time.Hour {
			t.Errorf("expected SessionTTL %v, got %v", 12*time.Hour, cfg.SessionTTL)
		}
		if cfg.GitHubClientID != "test-gh-id" || cfg.GitHubClientSecret != "test-gh-secret" {
			t.Errorf("unexpected github oauth config: id=%q secret=%q", cfg.GitHubClientID, cfg.GitHubClientSecret)
		}
		if cfg.GoogleClientID != "test-g-id" || cfg.GoogleClientSecret != "test-g-secret" {
			t.Errorf("unexpected google oauth config: id=%q secret=%q", cfg.GoogleClientID, cfg.GoogleClientSecret)
		}
		if cfg.FacebookClientID != "test-fb-id" || cfg.FacebookClientSecret != "test-fb-secret" {
			t.Errorf("unexpected facebook oauth config: id=%q secret=%q", cfg.FacebookClientID, cfg.FacebookClientSecret)
		}
	})
}

func TestConfig_TrustedOrigins(t *testing.T) {
	t.Setenv("APP_TRUSTED_ORIGINS", "http://localhost:5173,http://localhost:3000")
	cfg, err := app.GetConfig(context.Background())
	if err != nil {
		t.Fatalf("GetConfig returned unexpected error: %v", err)
	}
	if len(cfg.TrustedOrigins) != 2 {
		t.Fatalf("expected 2 trusted origins, got %d", len(cfg.TrustedOrigins))
	}
	if cfg.TrustedOrigins[0] != "http://localhost:5173" || cfg.TrustedOrigins[1] != "http://localhost:3000" {
		t.Errorf("unexpected trusted origins: %v", cfg.TrustedOrigins)
	}
}

type shutdownTrackingDriver struct {
	dialect.Driver
	onClose func()
}

func (d *shutdownTrackingDriver) Close() error {
	if d.onClose != nil {
		d.onClose()
	}
	return d.Driver.Close()
}

func TestOpus_Serve_ShutdownSequence(t *testing.T) {
	var (
		orderMu sync.Mutex
		order   []string
	)
	record := func(action string) {
		orderMu.Lock()
		order = append(order, action)
		orderMu.Unlock()
	}

	sqlDB, err := sql.Open("sqlite", "file:opus_shutdown_test?mode=memory&cache=shared&_pragma=foreign_keys(ON)")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	baseDrv := entsql.OpenDB(dialect.SQLite, sqlDB)
	trackingDrv := &shutdownTrackingDriver{
		Driver: baseDrv,
		onClose: func() {
			record("db.Close")
		},
	}
	client := enttest.NewClient(t, enttest.WithOptions(ent.Driver(trackingDrv)))

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	bus := eventbus.New(eventbus.WithLogger(logger))

	// Find free port
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	port := strconv.Itoa(ln.Addr().(*net.TCPAddr).Port)
	_ = ln.Close()

	cfg := app.Config{
		Host:            "127.0.0.1",
		Port:            port,
		ShutdownTimeout: 2 * time.Second,
	}

	fiberApp := app.GetFiber(cfg, logger, client, bus)

	httpListening := make(chan struct{})
	fiberApp.Hooks().OnListen(func(fiber.ListenData) error {
		close(httpListening)
		return nil
	})
	fiberApp.Hooks().OnPostShutdown(func(error) error {
		record("http.Shutdown")
		return nil
	})

	// Setup a subscriber that processes an event and records when bus.Close drains it
	drainBlocker := make(chan struct{})
	busDraining := make(chan struct{})
	err = bus.Subscribe("shutdown-sub", func(ctx context.Context, e eventbus.Event) error {
		close(busDraining)
		<-drainBlocker
		record("bus.Close")
		return nil
	})
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	dummyEvt, err := eventbus.NewEvent("test.shutdown", "test", nil)
	if err != nil {
		t.Fatalf("new event: %v", err)
	}
	if err := bus.Publish(context.Background(), dummyEvt); err != nil {
		t.Fatalf("publish: %v", err)
	}

	// Wait until subscriber starts processing the event
	select {
	case <-busDraining:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for subscriber to receive event")
	}

	opusApp := app.New(cfg, fiberApp, logger, client, bus)

	ctx, cancel := context.WithCancel(context.Background())
	serveDone := make(chan error, 1)
	go func() {
		serveDone <- opusApp.Serve(ctx)
	}()

	// Wait until HTTP is listening
	select {
	case <-httpListening:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for HTTP to listen")
	}

	// Trigger shutdown
	cancel()

	// Wait for http.Shutdown to be recorded before unblocking the bus drain
	deadline := time.Now().Add(2 * time.Second)
	for {
		orderMu.Lock()
		hasHTTP := len(order) > 0 && order[0] == "http.Shutdown"
		orderMu.Unlock()
		if hasHTTP || time.Now().After(deadline) {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	close(drainBlocker)

	select {
	case err := <-serveDone:
		if err != nil {
			t.Fatalf("unexpected Serve error: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for Serve to return")
	}

	orderMu.Lock()
	defer orderMu.Unlock()
	expectedOrder := []string{"http.Shutdown", "bus.Close", "db.Close"}
	if !reflect.DeepEqual(order, expectedOrder) {
		t.Fatalf("expected shutdown order %v, got %v", expectedOrder, order)
	}

	// AC2: after Serve returns, Publish on the bus returns ErrClosed
	if err := bus.Publish(context.Background(), dummyEvt); !errors.Is(err, eventbus.ErrClosed) {
		t.Fatalf("expected eventbus.ErrClosed after Serve returns, got %v", err)
	}
}

// AC3: The detector goroutine stops on app shutdown before db.Close in the shutdown sequence.
func TestOpus_Serve_ShutdownSequence_WithNode(t *testing.T) {
	var (
		orderMu sync.Mutex
		order   []string
	)
	record := func(action string) {
		orderMu.Lock()
		order = append(order, action)
		orderMu.Unlock()
	}

	var nodeFeature *node.Node

	sqlDB, err := sql.Open("sqlite", "file:opus_shutdown_node_test?mode=memory&cache=shared&_pragma=foreign_keys(ON)")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	baseDrv := entsql.OpenDB(dialect.SQLite, sqlDB)
	trackingDrv := &shutdownTrackingDriver{
		Driver: baseDrv,
		onClose: func() {
			if nodeFeature != nil && nodeFeature.Running() {
				t.Error("detector was still running when db.Close was called")
			}
			record("db.Close")
		},
	}
	client := enttest.NewClient(t, enttest.WithOptions(ent.Driver(trackingDrv)))

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	bus := eventbus.New(eventbus.WithLogger(logger))

	// Find free port
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	port := strconv.Itoa(ln.Addr().(*net.TCPAddr).Port)
	_ = ln.Close()

	cfg := app.Config{
		Host:            "127.0.0.1",
		Port:            port,
		ShutdownTimeout: 2 * time.Second,
	}

	nodeFeature = node.New(client, node.Config{
		BaseURL:          "http://localhost:7880",
		Bus:              bus,
		DetectorInterval: 10 * time.Millisecond,
		DetectorOnStop: func() {
			record("detector.Stop")
		},
	})

	fiberApp := app.GetFiber(cfg, logger, client, bus, app.WithNode(nodeFeature))

	httpListening := make(chan struct{})
	fiberApp.Hooks().OnListen(func(fiber.ListenData) error {
		close(httpListening)
		return nil
	})
	fiberApp.Hooks().OnPostShutdown(func(error) error {
		record("http.Shutdown")
		return nil
	})

	drainBlocker := make(chan struct{})
	busDraining := make(chan struct{})
	err = bus.Subscribe("shutdown-sub-node", func(ctx context.Context, e eventbus.Event) error {
		close(busDraining)
		<-drainBlocker
		record("bus.Close")
		return nil
	})
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	dummyEvt, err := eventbus.NewEvent("test.shutdown", "test", nil)
	if err != nil {
		t.Fatalf("new event: %v", err)
	}
	if err := bus.Publish(context.Background(), dummyEvt); err != nil {
		t.Fatalf("publish: %v", err)
	}

	select {
	case <-busDraining:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for subscriber to receive event")
	}

	opusApp := app.New(cfg, fiberApp, logger, client, bus, app.WithNode(nodeFeature))

	ctx, cancel := context.WithCancel(context.Background())
	serveDone := make(chan error, 1)
	go func() {
		serveDone <- opusApp.Serve(ctx)
	}()

	select {
	case <-httpListening:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for HTTP to listen")
	}

	// Verify detector is running
	if !nodeFeature.Running() {
		t.Fatal("expected detector to be running while server is listening")
	}

	// Trigger shutdown
	cancel()

	// Wait for http.Shutdown to be recorded before unblocking the bus drain
	deadline := time.Now().Add(2 * time.Second)
	for {
		orderMu.Lock()
		hasHTTP := len(order) > 0 && order[0] == "http.Shutdown"
		orderMu.Unlock()
		if hasHTTP || time.Now().After(deadline) {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	close(drainBlocker)

	select {
	case err := <-serveDone:
		if err != nil {
			t.Fatalf("unexpected Serve error: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for Serve to return")
	}

	// Verify detector is no longer running
	if nodeFeature.Running() {
		t.Fatal("expected detector to be stopped after Serve returned")
	}

	orderMu.Lock()
	defer orderMu.Unlock()
	expectedOrder := []string{"http.Shutdown", "detector.Stop", "bus.Close", "db.Close"}
	if !reflect.DeepEqual(order, expectedOrder) {
		t.Fatalf("expected shutdown order %v, got %v", expectedOrder, order)
	}
}

func TestOpus_Serve_ShutdownSequence_WithTracker(t *testing.T) {
	var (
		orderMu sync.Mutex
		order   []string
	)
	record := func(action string) {
		orderMu.Lock()
		order = append(order, action)
		orderMu.Unlock()
	}

	var trackerFeature *tracker.Tracker

	sqlDB, err := sql.Open("sqlite", "file:opus_shutdown_tracker_test?mode=memory&cache=shared&_pragma=foreign_keys(ON)")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	baseDrv := entsql.OpenDB(dialect.SQLite, sqlDB)
	trackingDrv := &shutdownTrackingDriver{
		Driver: baseDrv,
		onClose: func() {
			if trackerFeature != nil && trackerFeature.Running() {
				t.Error("tracker poller was still running when db.Close was called")
			}
			record("db.Close")
		},
	}
	client := enttest.NewClient(t, enttest.WithOptions(ent.Driver(trackingDrv)))

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	bus := eventbus.New(eventbus.WithLogger(logger))

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	port := strconv.Itoa(ln.Addr().(*net.TCPAddr).Port)
	_ = ln.Close()

	cfg := app.Config{
		Host:            "127.0.0.1",
		Port:            port,
		ShutdownTimeout: 2 * time.Second,
	}

	trackerFeature = tracker.New(client, tracker.Config{
		PollInterval: 10 * time.Millisecond,
		Logger:       logger,
		Runner: func(_ context.Context, _ ...string) ([]byte, error) {
			return []byte("[]"), nil
		},
		OnStop: func() {
			record("tracker.Stop")
		},
	})

	fiberApp := app.GetFiber(cfg, logger, client, bus, app.WithTracker(trackerFeature))

	httpListening := make(chan struct{})
	fiberApp.Hooks().OnListen(func(fiber.ListenData) error {
		close(httpListening)
		return nil
	})
	fiberApp.Hooks().OnPostShutdown(func(error) error {
		record("http.Shutdown")
		return nil
	})

	opusApp := app.New(cfg, fiberApp, logger, client, bus, app.WithTracker(trackerFeature))

	ctx, cancel := context.WithCancel(context.Background())
	serveDone := make(chan error, 1)
	go func() {
		serveDone <- opusApp.Serve(ctx)
	}()

	select {
	case <-httpListening:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for HTTP to listen")
	}

	if !trackerFeature.Running() {
		t.Fatal("expected tracker poller to be running while server is listening")
	}

	cancel()

	select {
	case err := <-serveDone:
		if err != nil {
			t.Fatalf("unexpected Serve error: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for Serve to return")
	}

	if trackerFeature.Running() {
		t.Fatal("expected tracker poller to be stopped after Serve returned")
	}

	orderMu.Lock()
	expectedOrder := []string{"http.Shutdown", "tracker.Stop", "db.Close"}
	gotOrder := append([]string(nil), order...)
	orderMu.Unlock()
	if !reflect.DeepEqual(gotOrder, expectedOrder) {
		t.Fatalf("expected shutdown order %v, got %v", expectedOrder, gotOrder)
	}

	buf := make([]byte, 1<<20)
	n := runtime.Stack(buf, true)
	if strings.Contains(string(buf[:n]), "tracker/service.(*Poller)") {
		t.Fatalf("goroutine leak: tracker poller goroutine still active after Serve returned:\n%s", buf[:n])
	}
}

func TestOpus_MasterWorker_InProcessRoundTrip(t *testing.T) {
	db, err := sql.Open("sqlite", "file:"+t.Name()+"?mode=memory&cache=shared&_pragma=foreign_keys(ON)&_pragma=busy_timeout(10000)")
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	db.SetMaxOpenConns(1)
	drv := entsql.OpenDB(dialect.SQLite, db)
	client := enttest.NewClient(t, enttest.WithOptions(ent.Driver(drv)))

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	bus := eventbus.New(eventbus.WithLogger(logger))
	nodeFeature := node.New(client, node.Config{BaseURL: "http://127.0.0.1:0", Bus: bus})
	if err := nodeFeature.Bootstrap(context.Background()); err != nil {
		t.Fatalf("bootstrap node: %v", err)
	}
	masterID, err := nodeFeature.MasterID(context.Background())
	if err != nil || masterID == "" {
		t.Fatalf("resolve master ID: id=%q err=%v", masterID, err)
	}

	cfg := app.Config{
		AppName:         "opus",
		Host:            "127.0.0.1",
		Port:            "0",
		ShutdownTimeout: 2 * time.Second,
		AppHome:         t.TempDir(),
	}

	fiberApp := fiber.New()
	started := make(chan string, 1)
	events := make(chan worker.Event, 4)

	nodeFeature.Hub().OnEvent(func(nodeID string, env node.Envelope) {
		if nodeID != masterID {
			return
		}
		var evt worker.Event
		if err := json.Unmarshal(env.Payload, &evt); err == nil {
			events <- evt
		}
	})

	inProc := worker.NewInProcess()
	masterWorker := worker.New(worker.Config{
		NodeID:            masterID,
		HomeDir:           cfg.AppHome,
		InProcess:         inProc,
		HeartbeatInterval: time.Hour,
		Logger:            logger,
		CapabilityProbe: func(_ context.Context) worker.Capabilities {
			return worker.Capabilities{
				Agy: worker.ToolStatusReady,
				Gh:  worker.ToolStatusReady,
			}
		},
		OnStartRun: func(ctx context.Context, cmd worker.StartRun) error {
			started <- cmd.RunID
			<-ctx.Done()
			return nil
		},
	})

	opusApp := app.New(cfg, fiberApp, logger, client, bus,
		app.WithNode(nodeFeature),
		app.WithWorker(masterWorker, inProc),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	serveDone := make(chan error, 1)
	go func() {
		serveDone <- opusApp.Serve(ctx)
	}()

	// Wait until master in-process worker is registered on Hub.
	deadline := time.Now().Add(2 * time.Second)
	for !nodeFeature.Hub().Connected(masterID) {
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for master worker registration on Hub")
		}
		time.Sleep(10 * time.Millisecond)
	}

	startEnv, err := worker.NewStartRunEnvelope("cmd-master-1", worker.StartRun{RunID: "master-run-1"})
	if err != nil {
		t.Fatalf("encode start_run: %v", err)
	}
	if err := nodeFeature.Hub().Send(masterID, startEnv); err != nil {
		t.Fatalf("hub send start_run: %v", err)
	}

	select {
	case id := <-started:
		if id != "master-run-1" {
			t.Fatalf("expected master-run-1 started, got %s", id)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for master OnStartRun")
	}

	cancelEnv, err := worker.NewCancelRunEnvelope("cmd-master-cancel-1", worker.CancelRun{
		RunID:  "master-run-1",
		Reason: "stop",
	})
	if err != nil {
		t.Fatalf("encode cancel_run: %v", err)
	}
	if err := nodeFeature.Hub().Send(masterID, cancelEnv); err != nil {
		t.Fatalf("hub send cancel_run: %v", err)
	}

	select {
	case evt := <-events:
		if evt.Type != worker.EventRunCancelled || evt.RunID != "master-run-1" {
			t.Fatalf("expected run_cancelled for master-run-1, got %+v", evt)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for master worker event")
	}

	cancel()
	select {
	case err := <-serveDone:
		if err != nil {
			t.Fatalf("unexpected Serve error: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for Serve to exit")
	}
}
