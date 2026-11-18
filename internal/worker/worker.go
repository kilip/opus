package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fasthttp/websocket"
	"github.com/kilip/opus/internal/node"
	"github.com/kilip/opus/internal/worker/domain"
	"github.com/kilip/opus/internal/worker/provider/agy"
)

// Re-exported command and event constants from domain.
const (
	CommandStartRun        = domain.CommandStartRun
	CommandCancelRun       = domain.CommandCancelRun
	CommandCleanupWorktree = domain.CommandCleanupWorktree
	EventRunCancelled      = domain.EventRunCancelled
	EventRunError          = domain.EventRunError
	EventCapabilities      = domain.EventCapabilities
	EventSessionCompleted  = domain.EventSessionCompleted
	EventSessionFailed     = domain.EventSessionFailed
	EventSessionUpdate     = domain.EventSessionUpdate

	ToolStatusReady           = node.ToolStatusReady
	ToolStatusUnauthenticated = node.ToolStatusUnauthenticated
	ToolStatusMissing         = node.ToolStatusMissing
)

// Re-exported payload types and envelope constructors from domain.
type (
	Policy          = domain.Policy
	StartRun        = domain.StartRun
	CancelRun       = domain.CancelRun
	CleanupWorktree = domain.CleanupWorktree
	Event           = domain.Event
	Capabilities    = domain.Capabilities
	HelloPayload    = domain.HelloPayload
	ToolStatus      = domain.ToolStatus
)

var (
	NewStartRunEnvelope        = domain.NewStartRunEnvelope
	NewCancelRunEnvelope       = domain.NewCancelRunEnvelope
	NewCleanupWorktreeEnvelope = domain.NewCleanupWorktreeEnvelope
)

// RunFunc is the pluggable runner callback invoked for each start_run command.
// The provided ctx is cancelled when cancel_run is received for cmd.RunID or when the worker stops.
type RunFunc func(ctx context.Context, cmd StartRun) error

// ponytail: exponential backoff capped at 30s; add jitter if worker cluster exceeds 1k nodes
const (
	DefaultHeartbeatInterval = 15 * time.Second
	DefaultInitialBackoff    = 1 * time.Second
	DefaultMaxBackoff        = 30 * time.Second
	backoffFactor            = 2.0
)

// Config holds configuration parameters for a Worker instance.
type Config struct {
	// MasterURL is the base URL of the master node (e.g. "http://localhost:7880").
	MasterURL string

	// NodeID is the unique identifier of this worker node.
	NodeID string

	// Credential is the worker authentication secret.
	Credential string

	// HeartbeatInterval is the duration between heartbeat envelopes.
	// Defaults to 15s if <= 0.
	HeartbeatInterval time.Duration

	// Logger is the slog logger instance.
	// Defaults to slog.Default() if nil.
	Logger *slog.Logger

	// InProcess connects the worker directly to the master via Go channels
	// without network sockets. If set, MasterURL is not dialed.
	InProcess *InProcessTransport

	// InitialBackoff specifies the initial backoff duration for reconnections.
	// Defaults to 1s if <= 0.
	InitialBackoff time.Duration

	// MaxBackoff specifies the maximum backoff duration for reconnections.
	// Defaults to 30s if <= 0.
	MaxBackoff time.Duration

	// Dialer allows overriding the WebSocket dialer (e.g. in tests).
	// Defaults to websocket.DefaultDialer if nil.
	Dialer *websocket.Dialer

	// OnCommand is an optional callback invoked when a command envelope is received.
	OnCommand func(env node.Envelope)

	// OnConnect is an optional callback invoked when WebSocket connects and hello is sent.
	OnConnect func()

	// HomeDir is the worker data root directory (<APP_HOME>/{repos,wt,logs}).
	// Defaults to $APP_HOME if set, otherwise ~/.opus.
	HomeDir string

	// CapabilityProbe detects local tool availability.
	// Defaults to NewCapabilityProbe(AgyBin) if nil.
	CapabilityProbe CapabilityProbe

	// AgyBin is the agy binary path or name (defaults to WORKER_AGY_BIN, then "agy").
	AgyBin string

	// OnStartRun is the pluggable runner callback invoked for start_run commands.
	OnStartRun RunFunc
}

// Worker is the control channel client for a worker node.
type Worker struct {
	cfg               Config
	logger            *slog.Logger
	heartbeatInterval time.Duration
	initialBackoff    time.Duration
	maxBackoff        time.Duration
	dialer            *websocket.Dialer
	inProc            *InProcessTransport
	homeDir           string
	probe             CapabilityProbe

	mu       sync.Mutex
	wsConn   *websocket.Conn
	lastCaps domain.Capabilities

	runsMu sync.Mutex
	runs   map[string]activeRun
	runCtx context.Context
	runsWg sync.WaitGroup
}

type activeRun struct {
	cancel    context.CancelFunc
	sessionID string
}

// New creates a new Worker instance with the given configuration.
func New(cfg Config) *Worker {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	hbInterval := cfg.HeartbeatInterval
	if hbInterval <= 0 {
		hbInterval = DefaultHeartbeatInterval
	}

	initBackoff := cfg.InitialBackoff
	if initBackoff <= 0 {
		initBackoff = DefaultInitialBackoff
	}

	maxBack := cfg.MaxBackoff
	if maxBack <= 0 {
		maxBack = DefaultMaxBackoff
	}

	dialer := cfg.Dialer
	if dialer == nil {
		dialer = websocket.DefaultDialer
	}

	probe := cfg.CapabilityProbe
	if probe == nil {
		probe = NewCapabilityProbe(cfg.AgyBin)
	}

	w := &Worker{
		cfg:               cfg,
		logger:            logger,
		heartbeatInterval: hbInterval,
		initialBackoff:    initBackoff,
		maxBackoff:        maxBack,
		dialer:            dialer,
		inProc:            cfg.InProcess,
		homeDir:           resolveHomeDir(cfg.HomeDir),
		probe:             probe,
		runs:              make(map[string]activeRun),
	}
	if w.cfg.OnStartRun == nil {
		w.cfg.OnStartRun = w.runStart
	}
	return w
}

func resolveHomeDir(explicit string) string {
	if explicit != "" {
		return explicit
	}
	if env := os.Getenv("APP_HOME"); env != "" {
		return env
	}
	if home := os.Getenv("HOME"); home != "" {
		return filepath.Join(home, ".opus")
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return filepath.Join(home, ".opus")
	}
	return ".opus"
}

// HomeDir returns the resolved worker home directory.
func (w *Worker) HomeDir() string {
	return w.homeDir
}

func (w *Worker) bootstrapDirs() error {
	for _, sub := range []string{"repos", "wt", "logs"} {
		dir := filepath.Join(w.homeDir, sub)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("bootstrap worker dir %s: %w", dir, err)
		}
	}
	return nil
}

// Run starts the worker control channel client loop.
// It bootstraps local directories, connects to the master, sends hello,
// periodically heartbeats, and processes commands.
// Run returns nil when ctx is cancelled.
func (w *Worker) Run(ctx context.Context) error {
	if ctx.Err() != nil {
		return nil
	}

	if err := w.bootstrapDirs(); err != nil {
		return err
	}

	w.runsMu.Lock()
	w.runCtx = ctx
	w.runsMu.Unlock()

	defer w.runsWg.Wait()

	if w.inProc != nil {
		return w.runInProcess(ctx)
	}

	return w.runWebSocket(ctx)
}

// Send delivers an envelope (typically an event) to the master.
func (w *Worker) Send(env node.Envelope) error {
	if w.inProc != nil {
		return w.inProc.SendEvent(context.Background(), env)
	}

	w.mu.Lock()
	defer w.mu.Unlock()
	if w.wsConn == nil {
		return node.ErrNodeNotConnected
	}
	return w.wsConn.WriteJSON(env)
}

// Connected returns true if the worker is currently connected to master.
func (w *Worker) Connected() bool {
	if w.inProc != nil {
		return !w.inProc.Closed()
	}

	w.mu.Lock()
	defer w.mu.Unlock()
	return w.wsConn != nil
}

func (w *Worker) buildHelloEnvelope(ctx context.Context) (node.Envelope, error) {
	caps := w.probe(ctx)
	w.mu.Lock()
	w.lastCaps = caps
	w.mu.Unlock()

	netInfo := DetectNetInfo(w.cfg.MasterURL)
	raw, err := json.Marshal(domain.HelloPayload{
		NodeID:       w.cfg.NodeID,
		Capabilities: caps,
		Hostname:     netInfo.Hostname,
		IPAddress:    netInfo.IPAddress,
		MACAddress:   netInfo.MACAddress,
	})
	if err != nil {
		return node.Envelope{}, err
	}
	return node.Envelope{
		Type:    node.EnvelopeTypeHello,
		ID:      fmt.Sprintf("hello-%s", w.cfg.NodeID),
		Payload: raw,
	}, nil
}

// CheckCapabilities probes local tool capabilities and emits a capability update
// event upward if the state changed since the last report.
func (w *Worker) CheckCapabilities(ctx context.Context) error {
	caps := w.probe(ctx)
	w.mu.Lock()
	changed := caps != w.lastCaps
	if changed {
		w.lastCaps = caps
	}
	w.mu.Unlock()

	if !changed {
		return nil
	}
	return w.sendWorkerEvent(ctx, domain.Event{
		Type:         domain.EventCapabilities,
		Capabilities: &caps,
	})
}

// ReportAgyUnauthenticated updates the worker's agy capability to unauthenticated
// and emits a capabilities_updated event if the state changed.
func (w *Worker) ReportAgyUnauthenticated(ctx context.Context) error {
	w.mu.Lock()
	caps := w.lastCaps
	changed := caps.Agy != node.ToolStatusUnauthenticated
	if changed {
		caps.Agy = node.ToolStatusUnauthenticated
		w.lastCaps = caps
	}
	w.mu.Unlock()

	if !changed {
		return nil
	}
	return w.sendWorkerEvent(ctx, domain.Event{
		Type:         domain.EventCapabilities,
		Capabilities: &caps,
	})
}

func (w *Worker) sendWorkerEvent(ctx context.Context, evt domain.Event) error {
	raw, err := json.Marshal(evt)
	if err != nil {
		return err
	}
	env := node.Envelope{
		Type:    node.EnvelopeTypeEvent,
		ID:      fmt.Sprintf("evt-%s-%d", w.cfg.NodeID, time.Now().UnixNano()),
		Payload: raw,
	}
	if w.inProc != nil {
		return w.inProc.SendEvent(ctx, env)
	}
	return w.Send(env)
}

func (w *Worker) runInProcess(ctx context.Context) error {
	helloEnv, err := w.buildHelloEnvelope(ctx)
	if err != nil {
		return err
	}
	if err := w.inProc.SendEvent(ctx, helloEnv); err != nil {
		if ctx.Err() != nil {
			return nil
		}
		return err
	}

	if w.cfg.OnConnect != nil {
		w.cfg.OnConnect()
	}

	stopHeartbeat := make(chan struct{})
	defer close(stopHeartbeat)

	go func() {
		ticker := time.NewTicker(w.heartbeatInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-stopHeartbeat:
				return
			case <-ticker.C:
				hbEnv := node.Envelope{
					Type: node.EnvelopeTypeHeartbeat,
					ID:   fmt.Sprintf("hb-%s-%d", w.cfg.NodeID, time.Now().UnixNano()),
				}
				if err := w.inProc.SendEvent(ctx, hbEnv); err != nil {
					return
				}
				_ = w.CheckCapabilities(ctx)
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-w.inProc.done:
			return nil
		case cmd, ok := <-w.inProc.Commands:
			if !ok {
				return nil
			}
			w.handleEnvelope(cmd)
		}
	}
}

func (w *Worker) runWebSocket(ctx context.Context) error {
	wsURL, err := buildWSURL(w.cfg.MasterURL)
	if err != nil {
		return err
	}

	header := make(http.Header)
	header.Set("Authorization", "Bearer "+w.cfg.Credential)

	backoff := w.initialBackoff

	for {
		if ctx.Err() != nil {
			return nil
		}

		conn, resp, err := w.dialer.Dial(wsURL, header)
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusUnauthorized {
				return fmt.Errorf("%w: invalid credential (status 401)", node.ErrUnauthorized)
			}
			if resp != nil && resp.StatusCode == http.StatusForbidden {
				return fmt.Errorf("master connect forbidden (status 403)")
			}

			w.logger.Warn("dial failed, reconnecting", "err", err, "backoff", backoff)
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(backoff):
			}
			backoff = NextBackoff(backoff, w.maxBackoff)
			continue
		}

		w.setConn(conn)

		helloEnv, err := w.buildHelloEnvelope(ctx)
		if err != nil {
			_ = conn.Close()
			w.clearConn()
			return err
		}
		if err := w.writeEnvelope(conn, helloEnv); err != nil {
			w.logger.Warn("failed to send hello", "err", err)
			_ = conn.Close()
			w.clearConn()

			select {
			case <-ctx.Done():
				return nil
			case <-time.After(backoff):
			}
			backoff = NextBackoff(backoff, w.maxBackoff)
			continue
		}

		backoff = w.initialBackoff

		if w.cfg.OnConnect != nil {
			w.cfg.OnConnect()
		}

		var wg sync.WaitGroup
		stopHeartbeat := make(chan struct{})

		wg.Add(1)
		go func() {
			defer wg.Done()
			w.heartbeatLoop(ctx, conn, stopHeartbeat)
		}()

		w.readLoop(conn)

		close(stopHeartbeat)
		_ = conn.Close()
		wg.Wait()
		w.clearConn()

		if ctx.Err() != nil {
			return nil
		}

		w.logger.Warn("connection dropped, reconnecting", "backoff", backoff)
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(backoff):
		}
		backoff = NextBackoff(backoff, w.maxBackoff)
	}
}

func (w *Worker) setConn(c *websocket.Conn) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.wsConn = c
}

func (w *Worker) clearConn() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.wsConn = nil
}

func (w *Worker) writeEnvelope(conn *websocket.Conn, env node.Envelope) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return conn.WriteJSON(env)
}

func (w *Worker) heartbeatLoop(ctx context.Context, conn *websocket.Conn, stop <-chan struct{}) {
	ticker := time.NewTicker(w.heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			_ = conn.Close()
			return
		case <-stop:
			return
		case <-ticker.C:
			hbEnv := node.Envelope{
				Type: node.EnvelopeTypeHeartbeat,
				ID:   fmt.Sprintf("hb-%s-%d", w.cfg.NodeID, time.Now().UnixNano()),
			}
			if err := w.writeEnvelope(conn, hbEnv); err != nil {
				w.logger.Debug("heartbeat write failed", "err", err)
				_ = conn.Close()
				return
			}
			_ = w.CheckCapabilities(ctx)
		}
	}
}

func (w *Worker) readLoop(conn *websocket.Conn) {
	for {
		var env node.Envelope
		if err := conn.ReadJSON(&env); err != nil {
			return
		}
		w.handleEnvelope(env)
	}
}

func (w *Worker) handleEnvelope(env node.Envelope) {
	switch env.Type {
	case node.EnvelopeTypeCommand:
		w.logger.Info("received command", "id", env.ID, "payload", string(env.Payload))
		if w.cfg.OnCommand != nil {
			w.cfg.OnCommand(env)
		}
		w.dispatchCommand(env)
	default:
		w.logger.Debug("received envelope", "type", env.Type, "id", env.ID)
	}
}

func (w *Worker) dispatchCommand(env node.Envelope) {
	if len(env.Payload) == 0 {
		return
	}
	var hdr struct {
		Type   string `json:"type"`
		Action string `json:"action"`
	}
	if err := json.Unmarshal(env.Payload, &hdr); err != nil {
		w.logger.Warn("invalid command payload", "id", env.ID, "err", err)
		return
	}
	cmdType := hdr.Type
	if cmdType == "" {
		cmdType = hdr.Action
	}

	switch cmdType {
	case domain.CommandStartRun:
		var cmd domain.StartRun
		if err := json.Unmarshal(env.Payload, &cmd); err != nil {
			_ = w.sendWorkerEvent(w.baseContext(), domain.Event{
				Type:  domain.EventRunError,
				Error: fmt.Sprintf("invalid start_run payload: %v", err),
			})
			return
		}
		w.handleStartRun(cmd)
	case domain.CommandCancelRun:
		var cmd domain.CancelRun
		if err := json.Unmarshal(env.Payload, &cmd); err != nil {
			_ = w.sendWorkerEvent(w.baseContext(), domain.Event{
				Type:  domain.EventRunError,
				Error: fmt.Sprintf("invalid cancel_run payload: %v", err),
			})
			return
		}
		w.handleCancelRun(cmd)
	case domain.CommandCleanupWorktree:
		var cmd domain.CleanupWorktree
		if err := json.Unmarshal(env.Payload, &cmd); err != nil {
			w.logger.Warn("invalid cleanup_worktree payload", "id", env.ID, "err", err)
			return
		}
		gitMgr := w.Git()
		baseCtx := w.baseContext()
		repoDir, err := gitMgr.EnsureRepo(baseCtx, cmd.ProjectSlug, "", cmd.MappedPath)
		if err != nil {
			w.logger.Warn("cleanup_worktree ensure repo failed", "slug", cmd.ProjectSlug, "err", err)
			return
		}
		if err := gitMgr.CleanupWorktree(baseCtx, repoDir, cmd.ProjectSlug, cmd.TaskKey, cmd.Branch); err != nil {
			w.logger.Warn("cleanup_worktree failed", "slug", cmd.ProjectSlug, "taskKey", cmd.TaskKey, "err", err)
		}
	}
}

func (w *Worker) baseContext() context.Context {
	w.runsMu.Lock()
	ctx := w.runCtx
	w.runsMu.Unlock()
	if ctx != nil {
		return ctx
	}
	return context.Background()
}

func (w *Worker) handleStartRun(cmd domain.StartRun) {
	baseCtx := w.baseContext()
	if cmd.RunID == "" {
		_ = w.sendWorkerEvent(baseCtx, domain.Event{
			Type:  domain.EventRunError,
			Error: "runId is required",
		})
		return
	}

	w.runsMu.Lock()
	if _, exists := w.runs[cmd.RunID]; exists {
		w.runsMu.Unlock()
		_ = w.sendWorkerEvent(baseCtx, domain.Event{
			Type:      domain.EventRunError,
			RunID:     cmd.RunID,
			SessionID: cmd.SessionID,
			Error:     "run already active",
		})
		return
	}

	runCtx, cancel := context.WithCancel(baseCtx)
	w.runs[cmd.RunID] = activeRun{cancel: cancel, sessionID: cmd.SessionID}
	w.runsWg.Add(1)
	w.runsMu.Unlock()

	go func() {
		defer func() {
			cancel()
			w.runsMu.Lock()
			delete(w.runs, cmd.RunID)
			w.runsMu.Unlock()
			w.runsWg.Done()
		}()

		if w.cfg.OnStartRun == nil {
			<-runCtx.Done()
			return
		}
		if err := w.cfg.OnStartRun(runCtx, cmd); err != nil && runCtx.Err() == nil {
			_ = w.sendWorkerEvent(baseCtx, domain.Event{
				Type:      domain.EventRunError,
				RunID:     cmd.RunID,
				SessionID: cmd.SessionID,
				Error:     w.scrubSecret(err.Error()),
			})
		}
	}()
}

// OpusMCPServers builds the session-scoped Opus HTTP MCP server definition
// for an agy ACP session/new or session/resume call.
func (w *Worker) OpusMCPServers(cmd StartRun) []agy.MCPServer {
	base := strings.TrimRight(strings.TrimSpace(w.cfg.MasterURL), "/")
	if base == "" {
		return nil
	}
	var headers []agy.MCPHeader
	if w.cfg.Credential != "" {
		headers = append(headers, agy.MCPHeader{
			Name:  "Authorization",
			Value: "Bearer " + w.cfg.Credential,
		})
	}
	if cmd.RunID != "" {
		headers = append(headers, agy.MCPHeader{Name: "X-Opus-Run-ID", Value: cmd.RunID})
	}
	if cmd.SessionID != "" {
		headers = append(headers, agy.MCPHeader{Name: "X-Opus-Session-ID", Value: cmd.SessionID})
	}
	if cmd.ProjectID != "" {
		headers = append(headers, agy.MCPHeader{Name: "X-Opus-Project-ID", Value: cmd.ProjectID})
	}
	return []agy.MCPServer{
		{
			Name:    "opus",
			Type:    "http",
			URL:     base + "/mcp",
			Headers: headers,
		},
	}
}

// StartAgySession creates or resumes an agy ACP session with Opus MCP server config injected.
func (w *Worker) StartAgySession(
	ctx context.Context,
	client *agy.Client,
	cwd string,
	cmd StartRun,
	resumeSessionID string,
	meta *agy.SessionMeta,
) (agy.SessionNewResult, error) {
	servers := w.OpusMCPServers(cmd)
	if resumeSessionID != "" {
		return client.ResumeSession(ctx, agy.SessionResumeParams{
			SessionID:  resumeSessionID,
			Cwd:        cwd,
			MCPServers: servers,
		})
	}
	return client.NewSession(ctx, agy.SessionNewParams{
		Cwd:        cwd,
		MCPServers: servers,
		Meta:       meta,
	})
}

func (w *Worker) scrubSecret(s string) string {
	if w.cfg.Credential == "" {
		return s
	}
	s = strings.ReplaceAll(s, "Bearer "+w.cfg.Credential, "[REDACTED]")
	return strings.ReplaceAll(s, w.cfg.Credential, "[REDACTED]")
}

func (w *Worker) handleCancelRun(cmd domain.CancelRun) {
	if cmd.RunID == "" {
		return
	}
	w.runsMu.Lock()
	entry, ok := w.runs[cmd.RunID]
	w.runsMu.Unlock()
	if !ok {
		w.logger.Debug("cancel_run for unknown or finished run", "runId", cmd.RunID)
		return
	}
	entry.cancel()
	_ = w.sendWorkerEvent(w.baseContext(), domain.Event{
		Type:      domain.EventRunCancelled,
		RunID:     cmd.RunID,
		SessionID: entry.sessionID,
		Reason:    cmd.Reason,
	})
}

func buildWSURL(rawURL string) (string, error) {
	if rawURL == "" {
		return "", errors.New("master URL is required")
	}
	if !strings.Contains(rawURL, "://") {
		rawURL = "http://" + rawURL
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid master URL: %w", err)
	}
	switch u.Scheme {
	case "https":
		u.Scheme = "wss"
	case "http":
		u.Scheme = "ws"
	case "ws", "wss":
		// already websocket scheme
	default:
		return "", fmt.Errorf("unsupported URL scheme: %s", u.Scheme)
	}

	p := strings.TrimSuffix(u.Path, "/")
	if !strings.HasSuffix(p, "/api/nodes/connect") {
		if strings.HasSuffix(p, "/api") {
			u.Path = p + "/nodes/connect"
		} else {
			u.Path = p + "/api/nodes/connect"
		}
	}
	return u.String(), nil
}

// NextBackoff calculates the next backoff duration by multiplying current by backoffFactor,
// capped at max.
func NextBackoff(current, max time.Duration) time.Duration {
	next := time.Duration(float64(current) * backoffFactor)
	if next > max {
		return max
	}
	return next
}
