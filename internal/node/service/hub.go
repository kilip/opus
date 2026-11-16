package service

import (
	"fmt"
	"sync"

	"github.com/kilip/opus/internal/node/domain"
)

// Conn abstracts a live WebSocket connection.
type Conn interface {
	WriteJSON(v any) error
	Close() error
}

type nodeConn struct {
	conn Conn
	mu   sync.Mutex
}

// EventHandler processes an incoming event envelope from a worker node.
type EventHandler func(nodeID string, env domain.Envelope)

// DisconnectHandler processes a disconnect notification for a worker node.
type DisconnectHandler func(nodeID string)

// HeartbeatHandler processes a hello or heartbeat envelope from a worker node.
type HeartbeatHandler func(nodeID string, env domain.Envelope)

// Hub manages live worker node WebSocket connections.
type Hub struct {
	mu                 sync.RWMutex
	conns              map[string]*nodeConn
	eventHandlers      []EventHandler
	heartbeatHandlers  []HeartbeatHandler
	disconnectHandlers []DisconnectHandler
	masterID           string
}

// NewHub creates a new Hub. Optional masterID can be passed to prevent
// master node from registering live connections.
func NewHub(masterID ...string) *Hub {
	var mid string
	if len(masterID) > 0 {
		mid = masterID[0]
	}
	return &Hub{
		conns:    make(map[string]*nodeConn),
		masterID: mid,
	}
}

// SetMasterID updates the master node ID for rejection on connect.
func (h *Hub) SetMasterID(id string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.masterID = id
}

// Register records a live connection for nodeID.
// If an existing connection exists for nodeID, the previous connection is closed
// and replaced with the new connection.
// Returns an unregister cleanup function to be called when the connection closes.
func (h *Hub) Register(nodeID string, conn Conn) (func(), error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.masterID != "" && nodeID == h.masterID {
		return nil, domain.ErrMasterConnectForbidden
	}

	if old, ok := h.conns[nodeID]; ok {
		_ = old.conn.Close()
	}

	nc := &nodeConn{conn: conn}
	h.conns[nodeID] = nc

	cleanup := func() {
		var disconnected bool
		h.mu.Lock()
		if cur, ok := h.conns[nodeID]; ok && cur == nc {
			delete(h.conns, nodeID)
			disconnected = true
		}
		handlers := make([]DisconnectHandler, len(h.disconnectHandlers))
		copy(handlers, h.disconnectHandlers)
		h.mu.Unlock()

		if disconnected {
			for _, fn := range handlers {
				fn(nodeID)
			}
		}
	}

	return cleanup, nil
}

// Connected returns true if nodeID currently has an active connection.
func (h *Hub) Connected(nodeID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, ok := h.conns[nodeID]
	return ok
}

// Send delivers an envelope to the specified nodeID.
// Returns ErrNodeNotConnected if the node does not have an active connection.
func (h *Hub) Send(nodeID string, env domain.Envelope) error {
	if !domain.IsValidEnvelopeType(env.Type) {
		return fmt.Errorf("%w: %s", domain.ErrUnknownEnvelopeType, env.Type)
	}

	h.mu.RLock()
	nc, ok := h.conns[nodeID]
	h.mu.RUnlock()

	if !ok {
		return domain.ErrNodeNotConnected
	}

	nc.mu.Lock()
	defer nc.mu.Unlock()
	return nc.conn.WriteJSON(env)
}

// OnEvent registers a callback function for received worker events.
func (h *Hub) OnEvent(handler EventHandler) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.eventHandlers = append(h.eventHandlers, handler)
}

// OnHeartbeat registers a callback function for received worker hello/heartbeat envelopes.
func (h *Hub) OnHeartbeat(handler HeartbeatHandler) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.heartbeatHandlers = append(h.heartbeatHandlers, handler)
}

// OnDisconnect registers a callback function for worker node disconnects.
func (h *Hub) OnDisconnect(handler DisconnectHandler) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.disconnectHandlers = append(h.disconnectHandlers, handler)
}

// HandleMessage parses a raw JSON payload and dispatches it.
// Unknown envelope types are rejected without closing the hub.
func (h *Hub) HandleMessage(nodeID string, msg []byte) error {
	env, err := domain.DecodeEnvelope(msg)
	if err != nil {
		return err
	}
	return h.Dispatch(nodeID, env)
}

// Dispatch dispatches a decoded envelope to registered handlers.
func (h *Hub) Dispatch(nodeID string, env domain.Envelope) error {
	if !domain.IsValidEnvelopeType(env.Type) {
		return fmt.Errorf("%w: %s", domain.ErrUnknownEnvelopeType, env.Type)
	}

	switch env.Type {
	case domain.EnvelopeTypeEvent:
		h.mu.RLock()
		handlers := make([]EventHandler, len(h.eventHandlers))
		copy(handlers, h.eventHandlers)
		h.mu.RUnlock()

		for _, fn := range handlers {
			fn(nodeID, env)
		}
	case domain.EnvelopeTypeHello, domain.EnvelopeTypeHeartbeat:
		h.mu.RLock()
		handlers := make([]HeartbeatHandler, len(h.heartbeatHandlers))
		copy(handlers, h.heartbeatHandlers)
		h.mu.RUnlock()

		for _, fn := range handlers {
			fn(nodeID, env)
		}
	}
	return nil
}

// Disconnect closes the connection for the specified nodeID.
func (h *Hub) Disconnect(nodeID string) error {
	h.mu.RLock()
	nc, ok := h.conns[nodeID]
	h.mu.RUnlock()

	if !ok {
		return domain.ErrNodeNotConnected
	}

	nc.mu.Lock()
	defer nc.mu.Unlock()
	return nc.conn.Close()
}

// Close closes all active connections in the hub.
func (h *Hub) Close() error {
	h.mu.Lock()
	conns := make([]*nodeConn, 0, len(h.conns))
	for _, nc := range h.conns {
		conns = append(conns, nc)
	}
	h.mu.Unlock()

	for _, nc := range conns {
		nc.mu.Lock()
		_ = nc.conn.Close()
		nc.mu.Unlock()
	}
	return nil
}
