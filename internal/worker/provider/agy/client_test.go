package agy_test

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kilip/opus/internal/worker/provider/agy"
)

// helperConfig returns an agy.Config that spawns the test binary itself as a fake ACP server.
func helperConfig(t *testing.T, mode string) agy.Config {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}
	env := append(os.Environ(),
		"GO_WANT_HELPER_PROCESS=1",
		"AGY_FAKE_MODE="+mode,
		"AGY_TESTDATA_DIR="+filepath.Join("testdata"),
	)
	return agy.Config{
		Bin:  exe,
		Args: []string{"-test.run=TestHelperProcess", "--", "acp"},
		Env:  env,
	}
}

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	mode := os.Getenv("AGY_FAKE_MODE")
	testdataDir := os.Getenv("AGY_TESTDATA_DIR")

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)

	writeLine := func(v any) {
		raw, _ := json.Marshal(v)
		_, _ = os.Stdout.Write(append(raw, '\n'))
	}

	var activePromptID json.RawMessage
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var req struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
		}
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			continue
		}

		switch req.Method {
		case "initialize":
			raw, err := os.ReadFile(filepath.Join(testdataDir, "initialize.json"))
			if err != nil {
				os.Exit(2)
			}
			var fixture struct {
				Response json.RawMessage `json:"response"`
			}
			_ = json.Unmarshal(raw, &fixture)
			var resp map[string]any
			_ = json.Unmarshal(fixture.Response, &resp)
			resp["id"] = json.RawMessage(req.ID)
			writeLine(resp)

		case "session/new":
			raw, err := os.ReadFile(filepath.Join(testdataDir, "session_new.json"))
			if err != nil {
				os.Exit(2)
			}
			var fixture struct {
				Response json.RawMessage `json:"response"`
			}
			_ = json.Unmarshal(raw, &fixture)
			var resp map[string]any
			_ = json.Unmarshal(fixture.Response, &resp)
			resp["id"] = json.RawMessage(req.ID)
			writeLine(resp)

		case "session/resume":
			raw, err := os.ReadFile(filepath.Join(testdataDir, "session_cancel_resume.json"))
			if err != nil {
				os.Exit(2)
			}
			var fixture struct {
				ResumeResponse json.RawMessage `json:"resumeResponse"`
			}
			_ = json.Unmarshal(raw, &fixture)
			var resp map[string]any
			_ = json.Unmarshal(fixture.ResumeResponse, &resp)
			resp["id"] = json.RawMessage(req.ID)
			writeLine(resp)

		case "session/prompt":
			switch mode {
			case "stream":
				raw, err := os.ReadFile(filepath.Join(testdataDir, "session_prompt_updates.jsonl"))
				if err != nil {
					os.Exit(2)
				}
				for _, l := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
					if strings.Contains(l, `"stopReason"`) {
						var resp map[string]any
						_ = json.Unmarshal([]byte(l), &resp)
						resp["id"] = json.RawMessage(req.ID)
						writeLine(resp)
					} else {
						_, _ = os.Stdout.WriteString(l + "\n")
					}
				}

			case "permission_and_question":
				raw, err := os.ReadFile(filepath.Join(testdataDir, "permission_and_question.json"))
				if err != nil {
					os.Exit(2)
				}
				var fixture struct {
					PermissionRequest         map[string]any `json:"permissionRequest"`
					AskQuestionToolCallUpdate map[string]any `json:"askQuestionToolCallUpdate"`
				}
				_ = json.Unmarshal(raw, &fixture)
				// Emit permission request AND ask_question update as single-line NDJSON.
				writeLine(fixture.PermissionRequest)
				writeLine(fixture.AskQuestionToolCallUpdate)
				activePromptID = req.ID

			case "cancel_clean":
				// Wait for session/cancel notification.
				activePromptID = req.ID

			case "cancel_hang":
				// Ignore session/cancel and hang until SIGKILL.
				activePromptID = req.ID

			case "unauthenticated":
				writeLine(map[string]any{
					"jsonrpc": "2.0",
					"id":      json.RawMessage(req.ID),
					"error": map[string]any{
						"code":    -32000,
						"message": "Authentication required. Please visit the URL to log in:",
					},
				})

			case "quota":
				writeLine(map[string]any{
					"jsonrpc": "2.0",
					"id":      json.RawMessage(req.ID),
					"error": map[string]any{
						"code":    -32000,
						"message": "RESOURCE_EXHAUSTED: You have exhausted your capacity on this model.",
					},
				})
			}

		case "session/cancel":
			if mode == "cancel_clean" {
				writeLine(map[string]any{
					"jsonrpc": "2.0",
					"id":      json.RawMessage(activePromptID),
					"result":  map[string]any{"stopReason": "cancelled"},
				})
				os.Exit(0)
			}
			// In cancel_hang mode, intentionally do not exit so client must SIGKILL.

		case "":
			// JSON-RPC response from client (e.g. for session/request_permission).
			if mode == "permission_and_question" && len(activePromptID) > 0 {
				var clientResp agy.PermissionResponse
				var frame struct {
					Result json.RawMessage `json:"result"`
				}
				_ = json.Unmarshal([]byte(line), &frame)
				_ = json.Unmarshal(frame.Result, &clientResp)
				if clientResp.Outcome.OptionID == "allow_once" {
					writeLine(map[string]any{
						"jsonrpc": "2.0",
						"id":      json.RawMessage(activePromptID),
						"result":  map[string]any{"stopReason": "end_turn"},
					})
				}
			}
		}
	}
	os.Exit(0)
}

func TestClient_InitializeSessionPromptAndUpdates(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var (
		mu      sync.Mutex
		updates []agy.SessionUpdateParams
	)
	cfg := helperConfig(t, "stream")
	cfg.OnUpdate = func(u agy.SessionUpdateParams) {
		mu.Lock()
		updates = append(updates, u)
		mu.Unlock()
	}

	c, err := agy.Start(ctx, cfg)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() { _ = c.Close() }()

	initRes, err := c.Initialize(ctx, agy.InitializeParams{
		ClientInfo: agy.ClientInfo{Name: "opus-test", Version: "0.1.0"},
	})
	if err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	if initRes.AgentInfo.Name != "Antigravity ACP Server" {
		t.Errorf("unexpected agentInfo: %+v", initRes.AgentInfo)
	}

	sessRes, err := c.NewSession(ctx, agy.SessionNewParams{
		Cwd: t.TempDir(),
		Meta: &agy.SessionMeta{
			Antigravity: &agy.AntigravityMeta{
				SystemPrompt: &agy.SystemPromptMeta{Append: "Be concise."},
			},
		},
	})
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	if sessRes.SessionID == "" {
		t.Fatal("expected non-empty sessionId")
	}

	promptRes, err := c.Prompt(ctx, agy.SessionPromptParams{
		SessionID: sessRes.SessionID,
		Prompt:    []agy.PromptContent{{Type: "text", Text: "Say hi."}},
	})
	if err != nil {
		t.Fatalf("Prompt: %v", err)
	}
	if promptRes.StopReason != "end_turn" {
		t.Errorf("expected stopReason end_turn, got %q", promptRes.StopReason)
	}
	if promptRes.Usage == nil || promptRes.Usage.TotalTokens != 8773 {
		t.Errorf("unexpected usage: %+v", promptRes.Usage)
	}

	mu.Lock()
	gotUpdates := len(updates)
	mu.Unlock()
	if gotUpdates != 7 {
		t.Errorf("expected 7 streamed session/update notifications, got %d", gotUpdates)
	}
}

func TestClient_PermissionAndQuestionCallbacks(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	permCalled := make(chan agy.PermissionRequest, 1)
	questionCalled := make(chan agy.QuestionRequest, 1)

	cfg := helperConfig(t, "permission_and_question")
	cfg.OnPermission = func(_ context.Context, req agy.PermissionRequest) (agy.PermissionResponse, error) {
		permCalled <- req
		return agy.PermissionResponse{
			Outcome: agy.PermissionOutcome{
				Outcome:  "selected",
				OptionID: "allow_once",
			},
		}, nil
	}
	cfg.OnQuestion = func(_ context.Context, req agy.QuestionRequest) (agy.QuestionResponse, error) {
		questionCalled <- req
		return agy.QuestionResponse{Skipped: true}, nil
	}

	c, err := agy.Start(ctx, cfg)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() { _ = c.Close() }()

	if _, err := c.Initialize(ctx, agy.InitializeParams{}); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	sess, err := c.NewSession(ctx, agy.SessionNewParams{Cwd: t.TempDir()})
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}

	res, err := c.Prompt(ctx, agy.SessionPromptParams{
		SessionID: sess.SessionID,
		Prompt:    []agy.PromptContent{{Type: "text", Text: "run command"}},
	})
	if err != nil {
		t.Fatalf("Prompt: %v", err)
	}
	if res.StopReason != "end_turn" {
		t.Errorf("expected end_turn after permission approval, got %q", res.StopReason)
	}

	select {
	case req := <-permCalled:
		if len(req.Options) != 3 {
			t.Errorf("expected 3 permission options, got %d", len(req.Options))
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for OnPermission callback")
	}

	select {
	case q := <-questionCalled:
		if len(q.Questions) != 1 || q.Questions[0].Question != "Which option?" {
			t.Errorf("unexpected question request: %+v", q)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for OnQuestion callback")
	}
}

func TestClient_Cancel_CleanAndForceKill(t *testing.T) {
	t.Run("clean exit on cancel", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		cfg := helperConfig(t, "cancel_clean")
		cfg.CancelGrace = 2 * time.Second
		c, err := agy.Start(ctx, cfg)
		if err != nil {
			t.Fatalf("Start: %v", err)
		}
		defer func() { _ = c.Close() }()

		sess, err := c.NewSession(ctx, agy.SessionNewParams{Cwd: t.TempDir()})
		if err != nil {
			t.Fatalf("NewSession: %v", err)
		}

		promptErr := make(chan error, 1)
		go func() {
			res, err := c.Prompt(ctx, agy.SessionPromptParams{
				SessionID: sess.SessionID,
				Prompt:    []agy.PromptContent{{Type: "text", Text: "long task"}},
			})
			if err == nil && res.StopReason != "cancelled" {
				err = errors.New("expected stopReason cancelled")
			}
			promptErr <- err
		}()

		time.Sleep(50 * time.Millisecond)
		if err := c.Cancel(ctx, sess.SessionID); err != nil {
			t.Fatalf("Cancel: %v", err)
		}
		if err := <-promptErr; err != nil {
			t.Fatalf("Prompt after cancel: %v", err)
		}
	})

	t.Run("SIGKILL after grace period when process hangs", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		cfg := helperConfig(t, "cancel_hang")
		cfg.CancelGrace = 100 * time.Millisecond
		c, err := agy.Start(ctx, cfg)
		if err != nil {
			t.Fatalf("Start: %v", err)
		}
		defer func() { _ = c.Close() }()

		sess, err := c.NewSession(ctx, agy.SessionNewParams{Cwd: t.TempDir()})
		if err != nil {
			t.Fatalf("NewSession: %v", err)
		}

		go func() {
			_, _ = c.Prompt(ctx, agy.SessionPromptParams{
				SessionID: sess.SessionID,
				Prompt:    []agy.PromptContent{{Type: "text", Text: "hang"}},
			})
		}()

		time.Sleep(50 * time.Millisecond)
		start := time.Now()
		if err := c.Cancel(ctx, sess.SessionID); err != nil {
			t.Fatalf("Cancel: %v", err)
		}
		elapsed := time.Since(start)
		if elapsed < 80*time.Millisecond || elapsed > 2*time.Second {
			t.Errorf("expected Cancel to wait ~100ms grace before SIGKILL, took %v", elapsed)
		}
	})
}

func TestClient_UnauthenticatedAndQuotaErrors(t *testing.T) {
	for _, tc := range []struct {
		mode    string
		wantErr error
	}{
		{mode: "unauthenticated", wantErr: agy.ErrUnauthenticated},
		{mode: "quota", wantErr: agy.ErrQuotaExhausted},
	} {
		t.Run(tc.mode, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			c, err := agy.Start(ctx, helperConfig(t, tc.mode))
			if err != nil {
				t.Fatalf("Start: %v", err)
			}
			defer func() { _ = c.Close() }()

			_, err = c.Prompt(ctx, agy.SessionPromptParams{
				SessionID: "s1",
				Prompt:    []agy.PromptContent{{Type: "text", Text: "hi"}},
			})
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected error %v, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestClient_ResumeSession(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	c, err := agy.Start(ctx, helperConfig(t, "stream"))
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() { _ = c.Close() }()

	res, err := c.ResumeSession(ctx, agy.SessionResumeParams{
		SessionID: "00000000-0000-4000-8000-000000000001",
		Cwd:       t.TempDir(),
	})
	if err != nil {
		t.Fatalf("ResumeSession: %v", err)
	}
	if res.SessionID != "00000000-0000-4000-8000-000000000001" {
		t.Errorf("unexpected resumed sessionId: %q", res.SessionID)
	}
}

func TestProbeAuth_FakeBinary(t *testing.T) {
	ctx := context.Background()

	t.Run("missing binary", func(t *testing.T) {
		installed, auth := agy.ProbeAuth(ctx, filepath.Join(t.TempDir(), "no-such-agy"))
		if installed || auth {
			t.Errorf("expected (false, false), got (%v, %v)", installed, auth)
		}
	})

	t.Run("unauthenticated binary", func(t *testing.T) {
		bin := filepath.Join(t.TempDir(), "agy")
		if err := os.WriteFile(bin, []byte("#!/bin/sh\necho 'Authentication required' >&2\nexit 1\n"), 0o755); err != nil {
			t.Fatalf("write fake agy: %v", err)
		}
		installed, auth := agy.ProbeAuth(ctx, bin)
		if !installed || auth {
			t.Errorf("expected (true, false), got (%v, %v)", installed, auth)
		}
	})

	t.Run("ready binary", func(t *testing.T) {
		bin := filepath.Join(t.TempDir(), "agy")
		if err := os.WriteFile(bin, []byte("#!/bin/sh\necho 'gemini-4-argon-medium'\nexit 0\n"), 0o755); err != nil {
			t.Fatalf("write fake agy: %v", err)
		}
		installed, auth := agy.ProbeAuth(ctx, bin)
		if !installed || !auth {
			t.Errorf("expected (true, true), got (%v, %v)", installed, auth)
		}
	})
}

func TestClient_InjectsHTTPMCPServers(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dir := t.TempDir()
	capturedPath := filepath.Join(dir, "requests.jsonl")
	bin := filepath.Join(dir, "fake-agy")
	script := `#!/bin/sh
while IFS= read -r line; do
  printf '%s\n' "$line" >> "` + capturedPath + `"
  case "$line" in
    *'"initialize"'*)
      printf '{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":1,"agentInfo":{"name":"fake","version":"1.0"}}}\n'
      ;;
    *'"session/new"'*)
      printf '{"jsonrpc":"2.0","id":2,"result":{"sessionId":"sess-new-1"}}\n'
      ;;
    *'"session/resume"'*)
      printf '{"jsonrpc":"2.0","id":3,"result":{"sessionId":"sess-new-1"}}\n'
      ;;
  esac
done
`
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake agy: %v", err)
	}

	c, err := agy.Start(ctx, agy.Config{Bin: bin})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() { _ = c.Close() }()

	if _, err := c.Initialize(ctx, agy.InitializeParams{}); err != nil {
		t.Fatalf("Initialize: %v", err)
	}

	servers := []agy.MCPServer{
		{
			Name: "opus",
			Type: "http",
			URL:  "http://127.0.0.1:7880/mcp",
			Headers: []agy.MCPHeader{
				{Name: "Authorization", Value: "Bearer secret-cred"},
				{Name: "X-Opus-Run-ID", Value: "run-1"},
			},
		},
	}

	if _, err := c.NewSession(ctx, agy.SessionNewParams{Cwd: dir, MCPServers: servers}); err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	if _, err := c.ResumeSession(ctx, agy.SessionResumeParams{SessionID: "sess-new-1", Cwd: dir, MCPServers: servers}); err != nil {
		t.Fatalf("ResumeSession: %v", err)
	}

	raw, err := os.ReadFile(capturedPath)
	if err != nil {
		t.Fatalf("read captured requests: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`"type":"http"`,
		`"url":"http://127.0.0.1:7880/mcp"`,
		`"name":"Authorization"`,
		`"value":"Bearer secret-cred"`,
		`"name":"X-Opus-Run-ID"`,
		`"value":"run-1"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("captured ACP requests missing %s:\n%s", want, text)
		}
	}
}
