package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/fasthttp/websocket"
	"github.com/kilip/opus/internal/node"
	"github.com/kilip/opus/internal/platform/httpx"
)

// AC1: opus join <url> exchanges the token, writes ~/.opus/node.json with mode 0600
// (directory 0700), then runs worker mode until SIGINT or SIGTERM, then exits 0.
// Tested against an httptest server.
func TestJoin_AC1_ExchangeWriteFileAndRunWorker(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	wsConnected := make(chan struct{}, 1)
	upgrader := websocket.Upgrader{
		CheckOrigin: func(*http.Request) bool { return true },
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/nodes/join":
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			var req node.JoinNodeRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}
			if req.Token != "valid-join-token" || req.Name != "worker-ac1" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(httpx.Response[node.JoinNodeResponse]{
				Data: node.JoinNodeResponse{
					NodeID:     "node-ac1-id",
					Credential: "cred-ac1-secret",
				},
			})

		case "/api/nodes/connect":
			auth := r.Header.Get("Authorization")
			if auth != "Bearer cred-ac1-secret" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
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

		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	joinErrCh := make(chan error, 1)
	go func() {
		joinErrCh <- runJoin(ctx, ts.URL+"/join/valid-join-token", "worker-ac1")
	}()

	select {
	case <-wsConnected:
		// WebSocket connected successfully, cancel context to simulate SIGINT/SIGTERM
		cancel()
	case err := <-joinErrCh:
		t.Fatalf("runJoin exited prematurely: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for worker WebSocket connection")
	}

	select {
	case err := <-joinErrCh:
		if err != nil {
			t.Fatalf("expected nil error on cancellation (exit 0), got: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for runJoin to exit")
	}

	// Verify ~/.opus directory mode is 0700
	dirPath := filepath.Join(tempHome, ".opus")
	dirInfo, err := os.Stat(dirPath)
	if err != nil {
		t.Fatalf("stat ~/.opus directory: %v", err)
	}
	if !dirInfo.IsDir() {
		t.Fatalf("expected %s to be directory", dirPath)
	}
	if runtime.GOOS != "windows" {
		if perm := dirInfo.Mode().Perm(); perm != 0700 {
			t.Errorf("expected ~/.opus directory mode 0700, got %04o", perm)
		}
	}

	// Verify node.json mode is 0600
	cfgPath := filepath.Join(dirPath, "node.json")
	fileInfo, err := os.Stat(cfgPath)
	if err != nil {
		t.Fatalf("stat node.json: %v", err)
	}
	if runtime.GOOS != "windows" {
		if perm := fileInfo.Mode().Perm(); perm != 0600 {
			t.Errorf("expected node.json file mode 0600, got %04o", perm)
		}
	}

	// Verify node.json contents
	cfg, err := readNodeConfig(cfgPath)
	if err != nil {
		t.Fatalf("readNodeConfig: %v", err)
	}
	if cfg.NodeID != "node-ac1-id" {
		t.Errorf("expected NodeID 'node-ac1-id', got %q", cfg.NodeID)
	}
	if cfg.Credential != "cred-ac1-secret" {
		t.Errorf("expected Credential 'cred-ac1-secret', got %q", cfg.Credential)
	}
	if cfg.MasterURL != ts.URL {
		t.Errorf("expected MasterURL %q, got %q", ts.URL, cfg.MasterURL)
	}
}

// AC2: opus join exits non-zero with a clear message if ~/.opus/node.json already exists,
// and does not call the network in that case.
func TestJoin_AC2_ExistingConfigFailsWithoutNetworkCall(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	// Pre-create ~/.opus/node.json
	opusDir := filepath.Join(tempHome, ".opus")
	if err := os.MkdirAll(opusDir, 0700); err != nil {
		t.Fatalf("mkdir .opus: %v", err)
	}
	cfgPath := filepath.Join(opusDir, "node.json")
	if err := os.WriteFile(cfgPath, []byte(`{"nodeId":"existing"}`), 0600); err != nil {
		t.Fatalf("write pre-existing node.json: %v", err)
	}

	networkCalled := false
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		networkCalled = true
		http.Error(w, "should not be called", http.StatusInternalServerError)
	}))
	defer ts.Close()

	err := runJoin(context.Background(), ts.URL+"/join/some-token", "test-node")
	if err == nil {
		t.Fatal("expected error when node.json already exists, got nil")
	}

	if networkCalled {
		t.Fatal("network call was made despite node.json already existing")
	}

	if !strings.Contains(err.Error(), "already joined") && !strings.Contains(err.Error(), "already exists") {
		t.Errorf("expected error message to mention 'already joined' or 'already exists', got: %v", err)
	}
}

// AC3: opus join exits non-zero with a clear message on a rejected token, a name conflict (409)
// or an unreachable master; node.json is not written on failure.
func TestJoin_AC3_FailuresDoNotWriteConfig(t *testing.T) {
	t.Run("rejected token (401)", func(t *testing.T) {
		tempHome := t.TempDir()
		t.Setenv("HOME", tempHome)

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(httpx.ErrorResponse{
				Error: "TOKEN_NOT_FOUND",
			})
		}))
		defer ts.Close()

		err := runJoin(context.Background(), ts.URL+"/join/bad-token", "my-node")
		if err == nil {
			t.Fatal("expected error on 401 rejected token, got nil")
		}
		if !strings.Contains(err.Error(), "401") && !strings.Contains(err.Error(), "rejected") {
			t.Errorf("expected error message to mention rejection or 401, got: %v", err)
		}

		cfgPath := filepath.Join(tempHome, ".opus", "node.json")
		if _, statErr := os.Stat(cfgPath); !os.IsNotExist(statErr) {
			t.Errorf("node.json should not exist on failure, stat err: %v", statErr)
		}
	})

	t.Run("name conflict (409)", func(t *testing.T) {
		tempHome := t.TempDir()
		t.Setenv("HOME", tempHome)

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			_ = json.NewEncoder(w).Encode(httpx.ErrorResponse{
				Error: "NODE_NAME_CONFLICT",
			})
		}))
		defer ts.Close()

		err := runJoin(context.Background(), ts.URL+"/join/token-dup", "duplicate-node")
		if err == nil {
			t.Fatal("expected error on 409 conflict, got nil")
		}
		if !strings.Contains(err.Error(), "409") && !strings.Contains(err.Error(), "duplicate-node") {
			t.Errorf("expected error message to mention 409 or node name, got: %v", err)
		}

		cfgPath := filepath.Join(tempHome, ".opus", "node.json")
		if _, statErr := os.Stat(cfgPath); !os.IsNotExist(statErr) {
			t.Errorf("node.json should not exist on failure, stat err: %v", statErr)
		}
	})

	t.Run("unreachable master", func(t *testing.T) {
		tempHome := t.TempDir()
		t.Setenv("HOME", tempHome)

		// Unreachable port
		err := runJoin(context.Background(), "http://127.0.0.1:54321/join/some-token", "my-node")
		if err == nil {
			t.Fatal("expected error on unreachable master, got nil")
		}
		if !strings.Contains(err.Error(), "unreachable") && !strings.Contains(err.Error(), "refused") {
			t.Errorf("expected error message to mention unreachable, got: %v", err)
		}

		cfgPath := filepath.Join(tempHome, ".opus", "node.json")
		if _, statErr := os.Stat(cfgPath); !os.IsNotExist(statErr) {
			t.Errorf("node.json should not exist on failure, stat err: %v", statErr)
		}
	})
}

func TestParseJoinURL(t *testing.T) {
	tests := []struct {
		name          string
		rawURL        string
		wantMasterURL string
		wantToken     string
		wantErr       bool
	}{
		{
			name:          "standard join URL",
			rawURL:        "http://localhost:7880/join/tok-123",
			wantMasterURL: "http://localhost:7880",
			wantToken:     "tok-123",
		},
		{
			name:          "trailing slash in join URL",
			rawURL:        "http://localhost:7880/join/tok-123/",
			wantMasterURL: "http://localhost:7880",
			wantToken:     "tok-123",
		},
		{
			name:          "URL with path prefix",
			rawURL:        "https://example.com/opus/join/tok-456",
			wantMasterURL: "https://example.com/opus",
			wantToken:     "tok-456",
		},
		{
			name:          "URL without scheme defaults to http",
			rawURL:        "localhost:7880/join/tok-789",
			wantMasterURL: "http://localhost:7880",
			wantToken:     "tok-789",
		},
		{
			name:    "empty URL",
			rawURL:  "",
			wantErr: true,
		},
		{
			name:    "no join path",
			rawURL:  "http://localhost:7880/foo/bar",
			wantErr: true,
		},
		{
			name:    "missing token after join",
			rawURL:  "http://localhost:7880/join/",
			wantErr: true,
		},
		{
			name:    "unsupported scheme",
			rawURL:  "ftp://localhost:7880/join/tok",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mURL, tok, err := parseJoinURL(tt.rawURL)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for URL %q, got nil", tt.rawURL)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for URL %q: %v", tt.rawURL, err)
			}
			if mURL != tt.wantMasterURL {
				t.Errorf("masterURL mismatch: got %q, want %q", mURL, tt.wantMasterURL)
			}
			if tok != tt.wantToken {
				t.Errorf("token mismatch: got %q, want %q", tok, tt.wantToken)
			}
		})
	}
}

func TestJoinCmd_CobraFlags(t *testing.T) {
	cmd := newJoinCmd()
	if cmd.Use != "join <url>" {
		t.Errorf("expected Use 'join <url>', got %q", cmd.Use)
	}
	flag := cmd.Flag("name")
	if flag == nil {
		t.Fatal("expected flag --name on join command")
	}
}
