package eventbus

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
)

var (
	// ErrClosed is returned when an operation is attempted on a closed bus.
	ErrClosed = errors.New("eventbus: bus is closed")
	// ErrDuplicateSubscriber is returned when subscribing with an already registered name.
	ErrDuplicateSubscriber = errors.New("eventbus: duplicate subscriber")
)

const defaultBufferSize = 256

// Handler processes an event delivered by the event bus.
type Handler func(ctx context.Context, e Event) error

// Option configures a Bus.
type Option func(*Bus)

// WithBuffer sets the per-subscriber channel buffer size (default is 256).
func WithBuffer(n int) Option {
	return func(b *Bus) {
		if n > 0 {
			b.bufferSize = n
		}
	}
}

// WithLogger sets the logger for the Bus (default is slog.Default()).
func WithLogger(logger *slog.Logger) Option {
	return func(b *Bus) {
		if logger != nil {
			b.logger = logger
		}
	}
}

// Bus is an in-process, at-most-once, best effort event bus.
// Delivery is asynchronous; late subscribers miss earlier events.
// If a subscriber queue overflows, newly published events are dropped.
type Bus struct {
	bufferSize int
	logger     *slog.Logger

	mu     sync.RWMutex
	closed bool
	subs   map[string]*subscriber
	wg     sync.WaitGroup

	ctx    context.Context
	cancel context.CancelFunc
}

type message struct {
	ctx   context.Context
	event Event
}

type subscriber struct {
	bus     *Bus
	name    string
	handler Handler
	ch      chan message
	types   map[string]struct{}
	dropped atomic.Uint64
}

func (s *subscriber) matches(eventType string) bool {
	if len(s.types) == 0 {
		return true
	}
	_, ok := s.types[eventType]
	return ok
}

// ponytail: one goroutine per subscriber; shard by hash(Subject) if a handler becomes a throughput bottleneck
func (s *subscriber) run() {
	defer s.bus.wg.Done()
	for msg := range s.ch {
		s.handleMessage(msg)
	}
}

func (s *subscriber) handleMessage(msg message) {
	defer func() {
		if r := recover(); r != nil {
			s.bus.logger.Error("event handler panic recovered",
				slog.String("subscriber", s.name),
				slog.String("event_id", msg.event.ID),
				slog.String("event_type", msg.event.Type),
				slog.Any("panic", r),
			)
		}
	}()

	hCtx, cancel := context.WithCancel(msg.ctx)
	stop := context.AfterFunc(s.bus.ctx, cancel)
	defer stop()
	defer cancel()

	// ponytail: no per-handler timeout; add one if a handler can hang while the bus is open
	if err := s.handler(hCtx, msg.event); err != nil {
		s.bus.logger.Error("event handler error",
			slog.String("subscriber", s.name),
			slog.String("event_id", msg.event.ID),
			slog.String("event_type", msg.event.Type),
			slog.Any("error", err),
		)
	}
}

// New creates a new in-process event bus.
func New(opts ...Option) *Bus {
	ctx, cancel := context.WithCancel(context.Background())
	b := &Bus{
		bufferSize: defaultBufferSize,
		logger:     slog.Default(),
		subs:       make(map[string]*subscriber),
		ctx:        ctx,
		cancel:     cancel,
	}
	for _, opt := range opts {
		opt(b)
	}
	return b
}

// Subscribe registers a subscriber with the given name and handler.
// If types are provided, only events matching one of the types are delivered.
// If types is empty, all events are delivered.
// Returns ErrClosed if the bus is closed, or ErrDuplicateSubscriber if name is already registered.
func (b *Bus) Subscribe(name string, h Handler, types ...string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return ErrClosed
	}
	if _, exists := b.subs[name]; exists {
		return ErrDuplicateSubscriber
	}

	sub := &subscriber{
		bus:     b,
		name:    name,
		handler: h,
		ch:      make(chan message, b.bufferSize),
		types:   make(map[string]struct{}, len(types)),
	}
	for _, t := range types {
		sub.types[t] = struct{}{}
	}

	b.subs[name] = sub
	b.wg.Add(1)
	go sub.run()

	return nil
}

// Publish sends an event to all matching subscribers.
// It is non-blocking: if a subscriber's queue is full, the event is dropped for that subscriber.
// Per-subscriber ordering is preserved by holding a read lock around non-blocking sends.
// Returns ErrClosed if the bus is closed.
func (b *Bus) Publish(ctx context.Context, e Event) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.closed {
		return ErrClosed
	}

	if ctx == nil {
		ctx = context.Background()
	}
	pubCtx := context.WithoutCancel(ctx)
	msg := message{
		ctx:   pubCtx,
		event: e,
	}

	for _, sub := range b.subs {
		if !sub.matches(e.Type) {
			continue
		}
		select {
		case sub.ch <- msg:
		default:
			// ponytail: drop-newest on overflow; task store is still in-memory so a dropped event is lost for good. Buffer 256 + error log until the outbox lands
			sub.dropped.Add(1)
			b.logger.Error("event dropped: subscriber queue full",
				slog.String("subscriber", sub.name),
				slog.String("event_type", e.Type),
			)
		}
	}
	return nil
}

// Dropped returns the number of events dropped for the named subscriber due to queue overflow.
// Returns 0 if subscriber is not found.
func (b *Bus) Dropped(name string) uint64 {
	b.mu.RLock()
	sub, ok := b.subs[name]
	b.mu.RUnlock()
	if !ok {
		return 0
	}
	return sub.dropped.Load()
}

// Close stops accepting new events, drains all pending events in subscriber queues,
// and waits for active handlers to complete or ctx to expire.
// If ctx expires before workers finish, Close cancels handler contexts and returns ctx.Err().
func (b *Bus) Close(ctx context.Context) error {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return ErrClosed
	}
	b.closed = true
	for _, sub := range b.subs {
		close(sub.ch)
	}
	b.mu.Unlock()

	if ctx == nil {
		ctx = context.Background()
	}

	done := make(chan struct{})
	go func() {
		b.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		b.cancel()
		return nil
	default:
	}

	select {
	case <-done:
		b.cancel()
		return nil
	case <-ctx.Done():
		b.cancel()
		return ctx.Err()
	}
}
