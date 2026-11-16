package worker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/fasthttp/websocket"
	"github.com/kilip/opus/internal/node"
)

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

	mu     sync.Mutex
	wsConn *websocket.Conn
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

	return &Worker{
		cfg:               cfg,
		logger:            logger,
		heartbeatInterval: hbInterval,
		initialBackoff:    initBackoff,
		maxBackoff:        maxBack,
		dialer:            dialer,
		inProc:            cfg.InProcess,
	}
}

// Run starts the worker control channel client loop.
// It connects to the master, sends hello, periodically heartbeats, and processes commands.
// Run returns nil when ctx is cancelled.
func (w *Worker) Run(ctx context.Context) error {
	if ctx.Err() != nil {
		return nil
	}

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

func (w *Worker) runInProcess(ctx context.Context) error {
	helloEnv := node.Envelope{
		Type: node.EnvelopeTypeHello,
		ID:   fmt.Sprintf("hello-%s", w.cfg.NodeID),
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

		helloEnv := node.Envelope{
			Type: node.EnvelopeTypeHello,
			ID:   fmt.Sprintf("hello-%s", w.cfg.NodeID),
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
	default:
		w.logger.Debug("received envelope", "type", env.Type, "id", env.ID)
	}
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
