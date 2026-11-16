package eventbus_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kilip/opus/internal/platform/eventbus"
)

type samplePayload struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

type payloadWithRaw struct {
	Raw json.RawMessage `json:"raw"`
}

func TestNewEvent_UUIDv7AndDefaults(t *testing.T) {
	t.Parallel()

	payload := samplePayload{Name: "alice", Age: 30}
	before := time.Now().UTC().Add(-time.Second)

	e, err := eventbus.NewEvent("user.created", "user-123", payload)
	if err != nil {
		t.Fatalf("unexpected NewEvent error: %v", err)
	}

	after := time.Now().UTC().Add(time.Second)

	// AC2: UUIDv7
	parsed, err := uuid.Parse(e.ID)
	if err != nil {
		t.Fatalf("ID %q is not a valid UUID: %v", e.ID, err)
	}
	if parsed.Version() != 7 {
		t.Fatalf("expected UUID version 7, got %d", parsed.Version())
	}

	if e.Type != "user.created" {
		t.Errorf("expected type user.created, got %s", e.Type)
	}
	if e.Subject != "user-123" {
		t.Errorf("expected subject user-123, got %s", e.Subject)
	}
	if e.At.Before(before) || e.At.After(after) {
		t.Errorf("expected timestamp between %v and %v, got %v", before, after, e.At)
	}
	if e.At.Location() != time.UTC {
		t.Errorf("expected UTC location, got %v", e.At.Location())
	}

	// Marshaling failure
	_, err = eventbus.NewEvent("test", "sub", make(chan int))
	if err == nil {
		t.Error("expected error for unmarshalable payload, got nil")
	}
}

func TestDecode_Immutability(t *testing.T) {
	t.Parallel()

	t.Run("struct unmarshaling", func(t *testing.T) {
		t.Parallel()
		e, err := eventbus.NewEvent("user.created", "user-1", samplePayload{Name: "bob", Age: 42})
		if err != nil {
			t.Fatalf("NewEvent error: %v", err)
		}
		got, err := eventbus.Decode[samplePayload](e)
		if err != nil {
			t.Fatalf("Decode error: %v", err)
		}
		if got.Name != "bob" || got.Age != 42 {
			t.Fatalf("unexpected decoded struct: %+v", got)
		}
	})

	t.Run("nil payload", func(t *testing.T) {
		t.Parallel()
		e, err := eventbus.NewEvent("ping", "server", nil)
		if err != nil {
			t.Fatalf("NewEvent error: %v", err)
		}
		got, err := eventbus.Decode[samplePayload](e)
		if err != nil {
			t.Fatalf("Decode error: %v", err)
		}
		if got != (samplePayload{}) {
			t.Fatalf("expected zero value, got %+v", got)
		}
	})

	t.Run("json.RawMessage immutability", func(t *testing.T) {
		t.Parallel()
		src := json.RawMessage(`{"field":"original"}`)
		e, err := eventbus.NewEvent("test", "sub", src)
		if err != nil {
			t.Fatalf("NewEvent error: %v", err)
		}

		// Mutating original input should not affect e.Payload
		src[2] = 'X'
		if bytes.Equal(src, e.Payload) {
			t.Fatal("e.Payload shares underlying array with NewEvent input")
		}

		// Mutating decoded result should not affect e.Payload
		decoded, err := eventbus.Decode[json.RawMessage](e)
		if err != nil {
			t.Fatalf("Decode error: %v", err)
		}
		decoded[2] = 'Z'
		if bytes.Equal(decoded, e.Payload) {
			t.Fatal("Decode handed out shared slice of e.Payload")
		}
	})

	t.Run("[]byte immutability", func(t *testing.T) {
		t.Parallel()
		src := []byte(`{"bytes":"raw"}`)
		e, err := eventbus.NewEvent("test", "sub", src)
		if err != nil {
			t.Fatalf("NewEvent error: %v", err)
		}

		src[2] = 'Y'
		if bytes.Equal(src, e.Payload) {
			t.Fatal("e.Payload shares underlying array with NewEvent input []byte")
		}

		decoded, err := eventbus.Decode[[]byte](e)
		if err != nil {
			t.Fatalf("Decode error: %v", err)
		}
		decoded[2] = 'W'
		if bytes.Equal(decoded, e.Payload) {
			t.Fatal("Decode handed out shared slice of e.Payload for []byte")
		}
	})

	t.Run("struct with json.RawMessage immutability", func(t *testing.T) {
		t.Parallel()
		e, err := eventbus.NewEvent("test", "sub", payloadWithRaw{Raw: json.RawMessage(`{"nested":"val"}`)})
		if err != nil {
			t.Fatalf("NewEvent error: %v", err)
		}

		decoded, err := eventbus.Decode[payloadWithRaw](e)
		if err != nil {
			t.Fatalf("Decode error: %v", err)
		}
		decoded.Raw[2] = 'Q'
		if bytes.Contains(e.Payload, []byte{'Q'}) {
			t.Fatal("Decode handed out shared slice into struct RawMessage field")
		}
	})

	t.Run("invalid json decode", func(t *testing.T) {
		t.Parallel()
		e := eventbus.Event{Payload: json.RawMessage(`invalid-json`)}
		_, err := eventbus.Decode[samplePayload](e)
		if err == nil {
			t.Fatal("expected decode error on invalid json, got nil")
		}
	})
}

func TestBus_SubscribeFiltering(t *testing.T) {
	t.Parallel()

	bus := eventbus.New()
	defer func() { _ = bus.Close(context.Background()) }()

	var subAllCount, subACount, subBCount atomic.Int32

	err := bus.Subscribe("subAll", func(ctx context.Context, e eventbus.Event) error {
		subAllCount.Add(1)
		return nil
	})
	if err != nil {
		t.Fatalf("subscribe subAll failed: %v", err)
	}

	err = bus.Subscribe("subA", func(ctx context.Context, e eventbus.Event) error {
		subACount.Add(1)
		return nil
	}, "event.A")
	if err != nil {
		t.Fatalf("subscribe subA failed: %v", err)
	}

	err = bus.Subscribe("subB", func(ctx context.Context, e eventbus.Event) error {
		subBCount.Add(1)
		return nil
	}, "event.B", "event.C")
	if err != nil {
		t.Fatalf("subscribe subB failed: %v", err)
	}

	ctx := context.Background()
	eA, _ := eventbus.NewEvent("event.A", "1", nil)
	eB, _ := eventbus.NewEvent("event.B", "2", nil)
	eD, _ := eventbus.NewEvent("event.D", "3", nil)

	_ = bus.Publish(ctx, eA)
	_ = bus.Publish(ctx, eB)
	_ = bus.Publish(ctx, eD)

	if err := bus.Close(context.Background()); err != nil {
		t.Fatalf("close failed: %v", err)
	}

	if got := subAllCount.Load(); got != 3 {
		t.Errorf("subAll expected 3 events, got %d", got)
	}
	if got := subACount.Load(); got != 1 {
		t.Errorf("subA expected 1 event, got %d", got)
	}
	if got := subBCount.Load(); got != 1 {
		t.Errorf("subB expected 1 event, got %d", got)
	}
}

func TestBus_DuplicateSubscriber(t *testing.T) {
	t.Parallel()

	bus := eventbus.New()
	defer func() { _ = bus.Close(context.Background()) }()

	dummy := func(ctx context.Context, e eventbus.Event) error { return nil }

	if err := bus.Subscribe("worker", dummy); err != nil {
		t.Fatalf("first subscribe failed: %v", err)
	}

	err := bus.Subscribe("worker", dummy)
	if !errors.Is(err, eventbus.ErrDuplicateSubscriber) {
		t.Fatalf("expected ErrDuplicateSubscriber, got %v", err)
	}
}

func TestBus_PerSubscriberOrdering(t *testing.T) {
	t.Parallel()

	bus := eventbus.New(eventbus.WithBuffer(5000))

	type pubMsg struct {
		PublisherID int `json:"publisher_id"`
		Seq         int `json:"seq"`
	}

	var mu sync.Mutex
	received := make(map[int][]int)

	err := bus.Subscribe("order-tracker", func(ctx context.Context, e eventbus.Event) error {
		msg, decErr := eventbus.Decode[pubMsg](e)
		if decErr != nil {
			return decErr
		}
		mu.Lock()
		received[msg.PublisherID] = append(received[msg.PublisherID], msg.Seq)
		mu.Unlock()
		return nil
	})
	if err != nil {
		t.Fatalf("subscribe failed: %v", err)
	}

	const (
		numPublishers = 8
		eventsPerPub  = 50
	)

	var wg sync.WaitGroup
	wg.Add(numPublishers)
	for i := 0; i < numPublishers; i++ {
		pubID := i
		go func() {
			defer wg.Done()
			for seq := 0; seq < eventsPerPub; seq++ {
				e, nErr := eventbus.NewEvent("seq.test", fmt.Sprintf("pub-%d", pubID), pubMsg{
					PublisherID: pubID,
					Seq:         seq,
				})
				if nErr != nil {
					t.Errorf("NewEvent failed: %v", nErr)
					return
				}
				if pErr := bus.Publish(context.Background(), e); pErr != nil {
					t.Errorf("Publish failed: %v", pErr)
					return
				}
			}
		}()
	}

	wg.Wait()

	if err := bus.Close(context.Background()); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	for p := 0; p < numPublishers; p++ {
		seqs := received[p]
		if len(seqs) != eventsPerPub {
			t.Fatalf("publisher %d: expected %d events, got %d", p, eventsPerPub, len(seqs))
		}
		for expectedSeq, actualSeq := range seqs {
			if actualSeq != expectedSeq {
				t.Fatalf("publisher %d: expected seq %d at index %d, got %d (order violated)", p, expectedSeq, expectedSeq, actualSeq)
			}
		}
	}
}

func TestBus_BlockedSubscriberDoesNotBlockPublishOrOthers(t *testing.T) {
	t.Parallel()

	// Buffer of 10
	bus := eventbus.New(eventbus.WithBuffer(10))

	slowBlocker := make(chan struct{})
	var fastReceived atomic.Int32

	// Slow subscriber: blocks on first event
	err := bus.Subscribe("slow", func(ctx context.Context, e eventbus.Event) error {
		<-slowBlocker
		return nil
	})
	if err != nil {
		t.Fatalf("subscribe slow failed: %v", err)
	}

	// Fast subscriber: completes immediately
	err = bus.Subscribe("fast", func(ctx context.Context, e eventbus.Event) error {
		fastReceived.Add(1)
		return nil
	})
	if err != nil {
		t.Fatalf("subscribe fast failed: %v", err)
	}

	const totalEvents = 25
	start := time.Now()
	for i := 0; i < totalEvents; i++ {
		e, _ := eventbus.NewEvent("test.overflow", fmt.Sprintf("subj-%d", i), i)
		if pErr := bus.Publish(context.Background(), e); pErr != nil {
			t.Fatalf("Publish failed: %v", pErr)
		}
		// Yield slightly so fast consumer can keep up with publisher
		time.Sleep(1 * time.Millisecond)
	}

	// Publish must not have blocked on slow subscriber
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("Publish blocked! Elapsed: %v", elapsed)
	}

	// Release slow subscriber before closing
	close(slowBlocker)

	if err := bus.Close(context.Background()); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Fast subscriber received all 25 events
	if got := fastReceived.Load(); got != totalEvents {
		t.Errorf("fast subscriber expected %d events, got %d", totalEvents, got)
	}

	// Slow subscriber must have dropped events due to overflow
	droppedSlow := bus.Dropped("slow")
	if droppedSlow == 0 {
		t.Errorf("expected slow subscriber dropped count > 0, got 0")
	}

	// Dropped for unknown subscriber should be 0
	if bus.Dropped("nonexistent") != 0 {
		t.Errorf("expected nonexistent subscriber dropped count 0, got %d", bus.Dropped("nonexistent"))
	}

	// Dropped for fast subscriber should be 0
	if bus.Dropped("fast") != 0 {
		t.Errorf("expected fast subscriber dropped count 0, got %d", bus.Dropped("fast"))
	}
}

func TestBus_QueueOverflowLogging(t *testing.T) {
	t.Parallel()

	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelError}))

	bus := eventbus.New(eventbus.WithBuffer(1), eventbus.WithLogger(logger))

	started := make(chan struct{})
	blocker := make(chan struct{})
	err := bus.Subscribe("overflow-sub", func(ctx context.Context, e eventbus.Event) error {
		select {
		case started <- struct{}{}:
		default:
		}
		<-blocker
		return nil
	})
	if err != nil {
		t.Fatalf("subscribe failed: %v", err)
	}

	e1, _ := eventbus.NewEvent("overflow.type", "1", nil)
	e2, _ := eventbus.NewEvent("overflow.type", "2", nil)
	e3, _ := eventbus.NewEvent("overflow.type", "3", nil)

	_ = bus.Publish(context.Background(), e1) // sent to channel
	<-started                                 // wait until worker starts processing e1 (channel is now empty)
	_ = bus.Publish(context.Background(), e2) // fills buffer capacity 1
	_ = bus.Publish(context.Background(), e3) // queue full -> dropped!

	close(blocker)
	_ = bus.Close(context.Background())

	if bus.Dropped("overflow-sub") != 1 {
		t.Fatalf("expected 1 dropped, got %d", bus.Dropped("overflow-sub"))
	}

	logOutput := logBuf.String()
	if !strings.Contains(logOutput, "overflow-sub") {
		t.Errorf("expected subscriber name in log, got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "overflow.type") {
		t.Errorf("expected event_type in log, got: %s", logOutput)
	}
}

func TestBus_HandlerPanicAndErrorRecovery(t *testing.T) {
	t.Parallel()

	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelError}))

	bus := eventbus.New(eventbus.WithLogger(logger))

	var handled atomic.Int32
	err := bus.Subscribe("resilient-worker", func(ctx context.Context, e eventbus.Event) error {
		cnt := handled.Add(1)
		if cnt == 1 {
			panic("boom! intentional panic")
		}
		if cnt == 2 {
			return errors.New("intentional error")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("subscribe failed: %v", err)
	}

	e1, _ := eventbus.NewEvent("panic.event", "1", nil)
	e2, _ := eventbus.NewEvent("err.event", "2", nil)
	e3, _ := eventbus.NewEvent("ok.event", "3", nil)

	ctx := context.Background()
	_ = bus.Publish(ctx, e1)
	_ = bus.Publish(ctx, e2)
	_ = bus.Publish(ctx, e3)

	if cErr := bus.Close(context.Background()); cErr != nil {
		t.Fatalf("Close failed: %v", cErr)
	}

	if handled.Load() != 3 {
		t.Fatalf("worker did not process all 3 events, processed %d", handled.Load())
	}

	logOutput := logBuf.String()
	if !strings.Contains(logOutput, "panic recovered") || !strings.Contains(logOutput, "resilient-worker") {
		t.Errorf("expected panic log with subscriber name, got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "panic.event") {
		t.Errorf("expected event_type in panic log, got: %s", logOutput)
	}
	if !strings.Contains(logOutput, e1.ID) {
		t.Errorf("expected event_id %s in panic log, got: %s", e1.ID, logOutput)
	}
}

func TestBus_CloseLifecycleAndDrain(t *testing.T) {
	t.Parallel()

	bus := eventbus.New()

	var count atomic.Int32
	err := bus.Subscribe("drain-test", func(ctx context.Context, e eventbus.Event) error {
		time.Sleep(10 * time.Millisecond)
		count.Add(1)
		return nil
	})
	if err != nil {
		t.Fatalf("subscribe failed: %v", err)
	}

	const n = 15
	for i := 0; i < n; i++ {
		e, _ := eventbus.NewEvent("drain.item", fmt.Sprintf("%d", i), nil)
		_ = bus.Publish(context.Background(), e)
	}

	if err := bus.Close(context.Background()); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	if got := count.Load(); got != n {
		t.Errorf("expected all %d events to drain, got %d", n, got)
	}

	// Publish and Subscribe after close must return ErrClosed
	dummyEvent, _ := eventbus.NewEvent("test", "sub", nil)
	if err := bus.Publish(context.Background(), dummyEvent); !errors.Is(err, eventbus.ErrClosed) {
		t.Errorf("expected ErrClosed on Publish after Close, got %v", err)
	}

	if err := bus.Subscribe("late", func(ctx context.Context, e eventbus.Event) error { return nil }); !errors.Is(err, eventbus.ErrClosed) {
		t.Errorf("expected ErrClosed on Subscribe after Close, got %v", err)
	}

	// Calling Close again returns ErrClosed
	if err := bus.Close(context.Background()); !errors.Is(err, eventbus.ErrClosed) {
		t.Errorf("expected ErrClosed on second Close, got %v", err)
	}
}

func TestBus_CloseDeadlineExpires_CancelsRunningHandler(t *testing.T) {
	t.Parallel()

	bus := eventbus.New()

	handlerSawCancel := make(chan bool, 1)

	err := bus.Subscribe("slow-canceler", func(ctx context.Context, e eventbus.Event) error {
		select {
		case <-ctx.Done():
			handlerSawCancel <- true
			return ctx.Err()
		case <-time.After(5 * time.Second):
			handlerSawCancel <- false
			return nil
		}
	})
	if err != nil {
		t.Fatalf("subscribe failed: %v", err)
	}

	e, _ := eventbus.NewEvent("slow.cancel", "1", nil)
	_ = bus.Publish(context.Background(), e)

	// Close with 50ms timeout
	closeCtx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	closeErr := bus.Close(closeCtx)
	if !errors.Is(closeErr, context.DeadlineExceeded) {
		t.Fatalf("expected context.DeadlineExceeded from Close, got %v", closeErr)
	}

	select {
	case sawCancel := <-handlerSawCancel:
		if !sawCancel {
			t.Fatal("handler did not see context cancellation")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("handler did not unblock within 2s after Close deadline")
	}
}

type ctxKey string

func TestBus_HandlerCtx_PublisherValueAndWithoutCancel(t *testing.T) {
	t.Parallel()

	bus := eventbus.New()

	valChan := make(chan string, 1)
	errChan := make(chan error, 1)

	err := bus.Subscribe("context-checker", func(ctx context.Context, e eventbus.Event) error {
		val, _ := ctx.Value(ctxKey("trace_id")).(string)
		valChan <- val
		errChan <- ctx.Err()
		return nil
	})
	if err != nil {
		t.Fatalf("subscribe failed: %v", err)
	}

	pubCtx, pubCancel := context.WithCancel(context.WithValue(context.Background(), ctxKey("trace_id"), "trace-xyz-123"))
	// Cancel publisher context immediately BEFORE publishing
	pubCancel()

	e, _ := eventbus.NewEvent("ctx.check", "sub", nil)
	if err := bus.Publish(pubCtx, e); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	if err := bus.Close(context.Background()); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	select {
	case val := <-valChan:
		if val != "trace-xyz-123" {
			t.Errorf("expected context value trace-xyz-123, got %q", val)
		}
	default:
		t.Fatal("did not receive context value from handler")
	}

	select {
	case err := <-errChan:
		if err != nil {
			t.Errorf("expected ctx.Err() to be nil (survived pubCtx cancel), got %v", err)
		}
	default:
		t.Fatal("did not receive ctx.Err() from handler")
	}
}

func TestBus_ConcurrentPublishAndClose(t *testing.T) {
	t.Parallel()

	bus := eventbus.New(eventbus.WithBuffer(100))

	var handled atomic.Int64
	_ = bus.Subscribe("worker", func(ctx context.Context, e eventbus.Event) error {
		handled.Add(1)
		return nil
	})

	var wg sync.WaitGroup
	const publishers = 16
	wg.Add(publishers)

	for i := 0; i < publishers; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				e, _ := eventbus.NewEvent("concurrent.evt", "subj", j)
				err := bus.Publish(context.Background(), e)
				if errors.Is(err, eventbus.ErrClosed) {
					return
				}
			}
		}()
	}

	// Concurrently close after brief delay
	time.Sleep(2 * time.Millisecond)
	_ = bus.Close(context.Background())

	wg.Wait()
}

func TestBus_DocAndPonytailComments(t *testing.T) {
	t.Parallel()

	content, err := os.ReadFile("eventbus.go")
	if err != nil {
		t.Fatalf("failed to read eventbus.go: %v", err)
	}
	src := string(content)

	// AC14: doc comment check
	requiredDocPhrases := []string{
		"at-most-once",
		"in-process",
		"best effort",
		"late subscribers miss earlier events",
	}
	for _, phrase := range requiredDocPhrases {
		if !strings.Contains(src, phrase) {
			t.Errorf("Bus doc comment missing required phrase: %q", phrase)
		}
	}

	// AC15: 3 ponytail comments
	requiredPonytailComments := []string{
		"// ponytail: one goroutine per subscriber; shard by hash(Subject) if a handler becomes a throughput bottleneck",
		"// ponytail: drop-newest on overflow; task store is still in-memory so a dropped event is lost for good. Buffer 256 + error log until the outbox lands",
		"// ponytail: no per-handler timeout; add one if a handler can hang while the bus is open",
	}
	for _, comment := range requiredPonytailComments {
		if !strings.Contains(src, comment) {
			t.Errorf("missing ponytail comment: %q", comment)
		}
	}
}
