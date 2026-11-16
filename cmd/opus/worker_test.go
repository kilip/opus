package main

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fasthttp/websocket"
)

// AC4: opus worker starts worker mode from node.json; missing file exits non-zero telling
// the user to run opus join.
func TestWorker_AC4_MissingConfigPromptsOpusJoin(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	// Ensure no node.json exists
	err := runWorker(context.Background())
	if err == nil {
		t.Fatal("expected error when node.json is missing, got nil")
	}

	if !strings.Contains(err.Error(), "opus join") {
		t.Errorf("expected error message to prompt 'opus join', got: %v", err)
	}

	// Also verify via Cobra command execution
	rootCmd := newRootCmd()
	rootCmd.SetArgs([]string{"worker"})
	cmdErr := rootCmd.ExecuteContext(context.Background())
	if cmdErr == nil {
		t.Fatal("expected CLI error for worker without config, got nil")
	}
	if !strings.Contains(cmdErr.Error(), "opus join") {
		t.Errorf("expected CLI error to prompt 'opus join', got: %v", cmdErr)
	}
}

// AC5: Worker mode starts no HTTP listener and serves no UI (test asserts no port is bound).
func TestWorker_AC5_NoHTTPListenerBound(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	wsConnected := make(chan struct{}, 1)
	upgrader := websocket.Upgrader{
		CheckOrigin: func(*http.Request) bool { return true },
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/nodes/connect" {
			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				return
			}
			defer func() { _ = conn.Close() }()
			select {
			case wsConnected <- struct{}{}:
			default:
			}
			for {
				if _, _, err := conn.ReadMessage(); err != nil {
					break
				}
			}
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	// Write valid node config
	cfgPath := filepath.Join(tempHome, ".opus", "node.json")
	err := writeNodeConfig(cfgPath, &NodeConfig{
		NodeID:     "worker-ac5-id",
		Credential: "secret-cred",
		MasterURL:  ts.URL,
	})
	if err != nil {
		t.Fatalf("write node config: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	workerErrCh := make(chan error, 1)
	go func() {
		workerErrCh <- runWorker(ctx)
	}()

	select {
	case <-wsConnected:
	case err := <-workerErrCh:
		t.Fatalf("runWorker failed unexpectedly: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for worker connection")
	}

	// AC5 assertion: Assert that worker has not bound an HTTP listener on 127.0.0.1:7880
	// (the default Opus HTTP serve port).
	conn, dialErr := net.DialTimeout("tcp", "127.0.0.1:7880", 50*time.Millisecond)
	if dialErr == nil {
		_ = conn.Close()
		t.Fatal("expected no HTTP server listening on port 7880, but connection succeeded")
	}

	// Cancel context to stop worker cleanly
	cancel()

	select {
	case err := <-workerErrCh:
		if err != nil {
			t.Fatalf("expected nil error on cancellation (exit 0), got: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for worker to shut down")
	}
}

// AC6: The home directory is overridable for tests (HOME env or a function parameter);
// tests never touch the real ~/.opus.
func TestWorker_AC6_HomeDirectoryOverridable(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	homeDir, err := opusHomeDir()
	if err != nil {
		t.Fatalf("opusHomeDir failed: %v", err)
	}

	expectedDir := filepath.Join(tempHome, ".opus")
	if homeDir != expectedDir {
		t.Errorf("expected opus home dir %q, got %q", expectedDir, homeDir)
	}

	configPath, err := defaultNodeConfigPath()
	if err != nil {
		t.Fatalf("defaultNodeConfigPath failed: %v", err)
	}

	expectedPath := filepath.Join(tempHome, ".opus", "node.json")
	if configPath != expectedPath {
		t.Errorf("expected node config path %q, got %q", expectedPath, configPath)
	}

	// Ensure writing config puts it inside the temp directory
	cfg := &NodeConfig{
		NodeID:     "ac6-node",
		Credential: "ac6-cred",
		MasterURL:  "http://localhost:7880",
	}
	if err := writeNodeConfig(configPath, cfg); err != nil {
		t.Fatalf("writeNodeConfig failed: %v", err)
	}

	if _, err := os.Stat(expectedPath); err != nil {
		t.Fatalf("expected file at %s, stat failed: %v", expectedPath, err)
	}

	// Ensure reading back retrieves the config from temp directory
	loaded, err := readNodeConfig(configPath)
	if err != nil {
		t.Fatalf("readNodeConfig failed: %v", err)
	}
	if loaded.NodeID != "ac6-node" || loaded.Credential != "ac6-cred" || loaded.MasterURL != "http://localhost:7880" {
		t.Errorf("unexpected loaded config: %+v", loaded)
	}
}

func TestRootCmd_Subcommands(t *testing.T) {
	rootCmd := newRootCmd()
	if rootCmd.RunE == nil {
		t.Fatal("expected rootCmd to have RunE pointing to serve by default")
	}

	foundServe := false
	foundJoin := false
	foundWorker := false
	for _, c := range rootCmd.Commands() {
		switch c.Name() {
		case "serve":
			foundServe = true
		case "join":
			foundJoin = true
		case "worker":
			foundWorker = true
		}
	}

	if !foundServe {
		t.Error("expected 'serve' subcommand registered")
	}
	if !foundJoin {
		t.Error("expected 'join' subcommand registered")
	}
	if !foundWorker {
		t.Error("expected 'worker' subcommand registered")
	}
}
