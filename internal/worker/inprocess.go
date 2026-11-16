package worker

import (
	"context"
	"errors"
	"sync"

	"github.com/kilip/opus/internal/node"
)

// ErrTransportClosed indicates an attempt to send on a closed transport.
var ErrTransportClosed = errors.New("in-process transport is closed")

// InProcessTransport carries control channel envelopes between master and worker
// in-process via Go channels without network sockets.
type InProcessTransport struct {
	Commands chan node.Envelope
	Events   chan node.Envelope

	mu     sync.Mutex
	closed bool
	done   chan struct{}
}

// NewInProcess creates a new InProcessTransport channel pair with the given buffer size.
// If bufferSize is omitted or <= 0, a default buffer size of 64 is used.
func NewInProcess(bufferSize ...int) *InProcessTransport {
	size := 64
	if len(bufferSize) > 0 && bufferSize[0] > 0 {
		size = bufferSize[0]
	}
	return &InProcessTransport{
		Commands: make(chan node.Envelope, size),
		Events:   make(chan node.Envelope, size),
		done:     make(chan struct{}),
	}
}

// SendCommand sends a command envelope from master to worker.
func (t *InProcessTransport) SendCommand(ctx context.Context, env node.Envelope) error {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return ErrTransportClosed
	}
	t.mu.Unlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.done:
		return ErrTransportClosed
	case t.Commands <- env:
		return nil
	}
}

// SendEvent sends an event envelope from worker to master.
func (t *InProcessTransport) SendEvent(ctx context.Context, env node.Envelope) error {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return ErrTransportClosed
	}
	t.mu.Unlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.done:
		return ErrTransportClosed
	case t.Events <- env:
		return nil
	}
}

// Close closes the in-process transport and unblocks pending operations.
func (t *InProcessTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return nil
	}
	t.closed = true
	close(t.done)
	return nil
}

// Closed returns true if the transport has been closed.
func (t *InProcessTransport) Closed() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.closed
}
