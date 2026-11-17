package agy

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"
)

// DefaultCancelGrace is the default grace period after session/cancel before SIGKILL.
const DefaultCancelGrace = 5 * time.Second

// Config configures an ACP stdio Client subprocess.
type Config struct {
	// Bin is the agy binary path or name. Defaults to WORKER_AGY_BIN, then "agy".
	Bin string

	// Args overrides the CLI arguments passed to Bin. Defaults to ["acp"]
	// (plus "--dangerously-skip-permissions" if DangerouslySkipPermissions is true).
	Args []string

	// Cwd is the working directory for the child process.
	Cwd string

	// Env optionally overrides or appends to the child process environment.
	Env []string

	// DangerouslySkipPermissions passes --dangerously-skip-permissions when Args is empty.
	DangerouslySkipPermissions bool

	// CancelGrace is the wait duration after session/cancel before sending SIGKILL.
	// Defaults to DefaultCancelGrace (5s) if <= 0.
	CancelGrace time.Duration

	// OnUpdate is invoked for each incoming session/update notification.
	OnUpdate func(SessionUpdateParams)

	// OnPermission is invoked for incoming session/request_permission requests.
	OnPermission func(context.Context, PermissionRequest) (PermissionResponse, error)

	// OnQuestion is invoked for incoming ask_question requests or tool calls.
	OnQuestion func(context.Context, QuestionRequest) (QuestionResponse, error)
}

type rpcMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int64          `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

// Client is a concrete JSON-RPC 2.0 stdio client for `agy acp`.
type Client struct {
	cfg         Config
	cmd         *exec.Cmd
	stdin       io.WriteCloser
	cancelGrace time.Duration

	nextID atomic.Int64

	writeMu sync.Mutex

	pendingMu sync.Mutex
	pending   map[int64]chan rpcMessage

	done     chan struct{}
	waitOnce sync.Once
	waitErr  error

	reqCtx    context.Context
	reqCancel context.CancelFunc
	reqWg     sync.WaitGroup
}

// Start spawns the `agy acp` subprocess and starts its stdio JSON-RPC read loop.
func Start(ctx context.Context, cfg Config) (*Client, error) {
	bin := cfg.Bin
	if bin == "" {
		bin = os.Getenv("WORKER_AGY_BIN")
	}
	if bin == "" {
		bin = "agy"
	}

	args := cfg.Args
	if len(args) == 0 {
		args = []string{"acp"}
		if cfg.DangerouslySkipPermissions {
			args = append(args, "--dangerously-skip-permissions")
		}
	}

	grace := cfg.CancelGrace
	if grace <= 0 {
		grace = DefaultCancelGrace
	}

	cmd := exec.Command(bin, args...)
	if cfg.Cwd != "" {
		cmd.Dir = cfg.Cwd
	}
	if len(cfg.Env) > 0 {
		cmd.Env = cfg.Env
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("agy stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("agy stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start %s: %w", bin, err)
	}

	reqCtx, reqCancel := context.WithCancel(ctx)
	c := &Client{
		cfg:         cfg,
		cmd:         cmd,
		stdin:       stdin,
		cancelGrace: grace,
		pending:     make(map[int64]chan rpcMessage),
		done:        make(chan struct{}),
		reqCtx:      reqCtx,
		reqCancel:   reqCancel,
	}

	go c.readLoop(stdout)
	return c, nil
}

// Initialize performs the ACP initialize handshake.
func (c *Client) Initialize(ctx context.Context, params InitializeParams) (InitializeResult, error) {
	if params.ProtocolVersion == 0 {
		params.ProtocolVersion = 1
	}
	if params.ClientCapabilities == nil {
		params.ClientCapabilities = map[string]any{}
	}
	var res InitializeResult
	if err := c.call(ctx, "initialize", params, &res); err != nil {
		return InitializeResult{}, err
	}
	return res, nil
}

// NewSession creates a new ACP session via session/new.
func (c *Client) NewSession(ctx context.Context, params SessionNewParams) (SessionNewResult, error) {
	if params.MCPServers == nil {
		params.MCPServers = []MCPServer{}
	}
	var res SessionNewResult
	if err := c.call(ctx, "session/new", params, &res); err != nil {
		return SessionNewResult{}, err
	}
	return res, nil
}

// ResumeSession restores an existing ACP session via session/resume.
func (c *Client) ResumeSession(ctx context.Context, params SessionResumeParams) (SessionNewResult, error) {
	if params.MCPServers == nil {
		params.MCPServers = []MCPServer{}
	}
	var res SessionNewResult
	if err := c.call(ctx, "session/resume", params, &res); err != nil {
		return SessionNewResult{}, err
	}
	return res, nil
}

// Prompt sends a prompt turn via session/prompt and waits for the turn result.
// Incremental updates are delivered asynchronously to Config.OnUpdate.
func (c *Client) Prompt(ctx context.Context, params SessionPromptParams) (SessionPromptResult, error) {
	var res SessionPromptResult
	if err := c.call(ctx, "session/prompt", params, &res); err != nil {
		return SessionPromptResult{}, err
	}
	return res, nil
}

// Cancel sends session/cancel, waits up to CancelGrace for the child process to exit,
// and terminates it with SIGKILL if it does not exit within the grace period.
func (c *Client) Cancel(ctx context.Context, sessionID string) error {
	_ = c.notify("session/cancel", map[string]string{"sessionId": sessionID})

	timer := time.NewTimer(c.cancelGrace)
	defer timer.Stop()

	select {
	case <-c.done:
		return nil
	case <-ctx.Done():
		c.killChild()
		<-c.done
		return ctx.Err()
	case <-timer.C:
		c.killChild()
		<-c.done
		return nil
	}
}

// Close terminates the child process if still running and waits for cleanup.
func (c *Client) Close() error {
	c.reqCancel()
	c.writeMu.Lock()
	_ = c.stdin.Close()
	c.writeMu.Unlock()

	timer := time.NewTimer(c.cancelGrace)
	defer timer.Stop()

	select {
	case <-c.done:
	case <-timer.C:
		c.killChild()
		<-c.done
	}
	c.reqWg.Wait()
	return nil
}

// Done returns a channel closed when the child process exits.
func (c *Client) Done() <-chan struct{} {
	return c.done
}

func (c *Client) killChild() {
	if c.cmd != nil && c.cmd.Process != nil {
		_ = c.cmd.Process.Kill()
	}
}

func (c *Client) call(ctx context.Context, method string, params any, out any) error {
	id := c.nextID.Add(1)
	ch := make(chan rpcMessage, 1)

	c.pendingMu.Lock()
	select {
	case <-c.done:
		c.pendingMu.Unlock()
		return ErrClientClosed
	default:
	}
	c.pending[id] = ch
	c.pendingMu.Unlock()

	defer func() {
		c.pendingMu.Lock()
		delete(c.pending, id)
		c.pendingMu.Unlock()
	}()

	rawParams, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("marshal %s params: %w", method, err)
	}
	req := rpcMessage{
		JSONRPC: "2.0",
		ID:      &id,
		Method:  method,
		Params:  rawParams,
	}
	if err := c.writeJSON(req); err != nil {
		return fmt.Errorf("write %s: %w", method, err)
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-c.done:
		return ErrClientClosed
	case resp, ok := <-ch:
		if !ok {
			return ErrClientClosed
		}
		if resp.Error != nil {
			return ClassifyRPCError(resp.Error.Code, resp.Error.Message)
		}
		if out != nil && len(resp.Result) > 0 {
			if err := json.Unmarshal(resp.Result, out); err != nil {
				return fmt.Errorf("decode %s result: %w", method, err)
			}
		}
		return nil
	}
}

func (c *Client) notify(method string, params any) error {
	rawParams, err := json.Marshal(params)
	if err != nil {
		return err
	}
	return c.writeJSON(rpcMessage{
		JSONRPC: "2.0",
		Method:  method,
		Params:  rawParams,
	})
}

func (c *Client) writeJSON(v any) error {
	raw, err := json.Marshal(v)
	if err != nil {
		return err
	}
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	_, err = c.stdin.Write(append(raw, '\n'))
	return err
}

func (c *Client) readLoop(stdout io.Reader) {
	defer func() {
		c.waitOnce.Do(func() {
			c.waitErr = c.cmd.Wait()
		})
		c.reqCancel()
		c.pendingMu.Lock()
		for id, ch := range c.pending {
			close(ch)
			delete(c.pending, id)
		}
		c.pendingMu.Unlock()
		close(c.done)
	}()

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var msg rpcMessage
		if err := json.Unmarshal(line, &msg); err != nil {
			continue
		}

		switch {
		case msg.Method != "" && msg.ID != nil:
			// Server-to-client request: handle in a goroutine so callbacks never block updates.
			c.reqWg.Add(1)
			go func(m rpcMessage) {
				defer c.reqWg.Done()
				c.handleServerRequest(m)
			}(msg)

		case msg.Method == "session/update":
			var params SessionUpdateParams
			if err := json.Unmarshal(msg.Params, &params); err == nil {
				if c.cfg.OnUpdate != nil {
					c.cfg.OnUpdate(params)
				}
				c.maybeDispatchQuestionUpdate(params)
			}

		case msg.ID != nil:
			c.pendingMu.Lock()
			ch, ok := c.pending[*msg.ID]
			c.pendingMu.Unlock()
			if ok {
				select {
				case ch <- msg:
				default:
				}
			}
		}
	}
}

func (c *Client) maybeDispatchQuestionUpdate(params SessionUpdateParams) {
	if c.cfg.OnQuestion == nil || params.Update.SessionUpdate != "tool_call" || params.Update.Name != "ask_question" {
		return
	}
	var args struct {
		Questions []QuestionItem `json:"questions"`
	}
	if err := json.Unmarshal(params.Update.Args, &args); err != nil {
		return
	}
	qReq := QuestionRequest{
		SessionID:  params.SessionID,
		ToolCallID: params.Update.ToolCallID,
		Questions:  args.Questions,
	}
	c.reqWg.Add(1)
	go func() {
		defer c.reqWg.Done()
		_, _ = c.cfg.OnQuestion(c.reqCtx, qReq)
	}()
}

func (c *Client) handleServerRequest(msg rpcMessage) {
	switch msg.Method {
	case "session/request_permission":
		var req PermissionRequest
		if err := json.Unmarshal(msg.Params, &req); err != nil {
			c.writeRPCError(*msg.ID, -32602, fmt.Sprintf("invalid permission request: %v", err))
			return
		}
		if c.cfg.OnPermission == nil {
			c.writeRPCError(*msg.ID, -32000, "no permission handler configured")
			return
		}
		resp, err := c.cfg.OnPermission(c.reqCtx, req)
		if err != nil {
			c.writeRPCError(*msg.ID, -32000, err.Error())
			return
		}
		c.writeRPCResult(*msg.ID, resp)

	case "ask_question", "session/ask_question":
		var req QuestionRequest
		if err := json.Unmarshal(msg.Params, &req); err != nil {
			c.writeRPCError(*msg.ID, -32602, fmt.Sprintf("invalid question request: %v", err))
			return
		}
		if c.cfg.OnQuestion == nil {
			c.writeRPCResult(*msg.ID, QuestionResponse{Skipped: true})
			return
		}
		resp, err := c.cfg.OnQuestion(c.reqCtx, req)
		if err != nil {
			c.writeRPCError(*msg.ID, -32000, err.Error())
			return
		}
		c.writeRPCResult(*msg.ID, resp)

	default:
		c.writeRPCError(*msg.ID, -32601, fmt.Sprintf("unsupported server request: %s", msg.Method))
	}
}

func (c *Client) writeRPCResult(id int64, res any) {
	raw, err := json.Marshal(res)
	if err != nil {
		c.writeRPCError(id, -32603, err.Error())
		return
	}
	_ = c.writeJSON(rpcMessage{
		JSONRPC: "2.0",
		ID:      &id,
		Result:  raw,
	})
}

func (c *Client) writeRPCError(id int64, code int, msg string) {
	_ = c.writeJSON(rpcMessage{
		JSONRPC: "2.0",
		ID:      &id,
		Error:   &RPCError{Code: code, Message: msg},
	})
}

// ProbeAuth checks whether bin exists on PATH and whether it is authenticated.
// Returns ("missing" / "unauthenticated" / "ready") via the provided status strings.
func ProbeAuth(ctx context.Context, bin string) (installed bool, authenticated bool) {
	if bin == "" {
		bin = os.Getenv("WORKER_AGY_BIN")
	}
	if bin == "" {
		bin = "agy"
	}
	if _, err := exec.LookPath(bin); err != nil {
		return false, false
	}
	probeCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	out, err := exec.CommandContext(probeCtx, bin, "models").CombinedOutput()
	if err != nil {
		return true, false
	}
	if errors.Is(ClassifyRPCError(0, string(out)), ErrUnauthenticated) {
		return true, false
	}
	return true, true
}
