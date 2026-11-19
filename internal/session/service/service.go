package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kilip/opus/internal/node"
	"github.com/kilip/opus/internal/platform/eventbus"
	"github.com/kilip/opus/internal/session/domain"
	"github.com/kilip/opus/internal/session/repo"
)

// Config configures the session service.
type Config struct {
	Bus       *eventbus.Bus
	Now       func() time.Time
	OnPromote func(domain.Session)
	Logger    *slog.Logger
}

// Service owns session lifecycle, single-active-session FIFO queueing,
// ordered event persistence, and live SSE subscriber fanout.
type Service struct {
	repo      *repo.Ent
	bus       *eventbus.Bus
	now       func() time.Time
	onPromote func(domain.Session)
	logger    *slog.Logger

	// writeMu serializes SQLite write transactions for queue/seq consistency.
	writeMu sync.Mutex

	subsMu sync.RWMutex
	subs   map[uuid.UUID]map[chan domain.SessionEvent]struct{}
}

// New creates a new session Service.
func New(r *repo.Ent, cfg Config) *Service {
	now := cfg.Now
	if now == nil {
		now = time.Now
	}
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{
		repo:      r,
		bus:       cfg.Bus,
		now:       now,
		onPromote: cfg.OnPromote,
		logger:    logger,
		subs:      make(map[uuid.UUID]map[chan domain.SessionEvent]struct{}),
	}
}

// Enqueue creates a session, starting it immediately if no other session is running
// or queueing it in FIFO order otherwise.
func (s *Service) Enqueue(ctx context.Context, in domain.CreateInput) (domain.Session, error) {
	if in.Provider == "" {
		return domain.Session{}, domain.ErrProviderRequired
	}

	s.writeMu.Lock()
	sess, err := s.repo.EnqueueOrStart(ctx, in, s.now())
	s.writeMu.Unlock()
	if err != nil {
		return domain.Session{}, err
	}

	if sess.Status == domain.StatusRunning {
		s.publishLifecycle(ctx, domain.EventStarted, sess)
	} else {
		s.publishLifecycle(ctx, domain.EventQueued, sess)
	}
	return sess, nil
}

// RecordEvent persists a monotonic 1-based SessionEvent and broadcasts it to active SSE subscribers.
func (s *Service) RecordEvent(ctx context.Context, sessionID uuid.UUID, kind string, payload json.RawMessage) (domain.SessionEvent, error) {
	if kind == "" {
		return domain.SessionEvent{}, domain.ErrEventKindEmpty
	}

	s.writeMu.Lock()
	evt, err := s.repo.AppendEvent(ctx, sessionID, kind, payload, s.now())
	s.writeMu.Unlock()
	if err != nil {
		return domain.SessionEvent{}, err
	}

	s.broadcast(evt)
	return evt, nil
}

// Complete marks sessionID as completed and promotes the next queued session if any.
func (s *Service) Complete(ctx context.Context, sessionID uuid.UUID) (domain.Session, *domain.Session, error) {
	return s.finish(ctx, sessionID, domain.StatusCompleted, "", domain.EventCompleted)
}

// Fail marks sessionID as failed with reason and promotes the next queued session if any.
func (s *Service) Fail(ctx context.Context, sessionID uuid.UUID, reason string) (domain.Session, *domain.Session, error) {
	return s.finish(ctx, sessionID, domain.StatusFailed, reason, domain.EventFailed)
}

// Cancel marks sessionID as cancelled and promotes the next queued session if any.
func (s *Service) Cancel(ctx context.Context, sessionID uuid.UUID, reason string) (domain.Session, *domain.Session, error) {
	return s.finish(ctx, sessionID, domain.StatusCancelled, reason, domain.EventCancelled)
}

func (s *Service) finish(
	ctx context.Context,
	sessionID uuid.UUID,
	status domain.SessionStatus,
	reason string,
	busTopic string,
) (domain.Session, *domain.Session, error) {
	s.writeMu.Lock()
	ended, promoted, err := s.repo.TransitionTerminalAndPromoteNext(ctx, sessionID, status, reason, s.now())
	s.writeMu.Unlock()
	if err != nil {
		return domain.Session{}, nil, err
	}

	// Emit a terminal status event so active SSE streams observe completion and close cleanly.
	termPayload, _ := json.Marshal(map[string]any{
		"status": status,
		"reason": reason,
	})
	_, _ = s.RecordEvent(ctx, sessionID, busTopic, termPayload)

	s.publishLifecycle(ctx, busTopic, ended)
	if promoted != nil {
		s.publishLifecycle(ctx, domain.EventStarted, *promoted)
		if s.onPromote != nil {
			s.onPromote(*promoted)
		}
	}
	return ended, promoted, nil
}

// Get returns a session by ID or domain.ErrSessionNotFound.
func (s *Service) Get(ctx context.Context, id uuid.UUID) (domain.Session, error) {
	return s.repo.Get(ctx, id)
}

// List returns sessions matching filter.
func (s *Service) List(ctx context.Context, filter domain.ListFilter) ([]domain.Session, error) {
	return s.repo.List(ctx, filter)
}

// ListEventsAfter returns ordered SessionEvents for sessionID with seq > afterSeq.
func (s *Service) ListEventsAfter(ctx context.Context, sessionID uuid.UUID, afterSeq int) ([]domain.SessionEvent, error) {
	return s.repo.ListEventsAfter(ctx, sessionID, afterSeq)
}

// LatestEventTime returns the timestamp of the latest event for sessionID, or nil if none exist.
func (s *Service) LatestEventTime(ctx context.Context, sessionID uuid.UUID) (*time.Time, error) {
	return s.repo.LatestEventTime(ctx, sessionID)
}

// Subscribe registers a live event stream channel for sessionID and returns an unsubscribe func.
func (s *Service) Subscribe(sessionID uuid.UUID) (<-chan domain.SessionEvent, func()) {
	ch := make(chan domain.SessionEvent, 64)
	s.subsMu.Lock()
	if s.subs[sessionID] == nil {
		s.subs[sessionID] = make(map[chan domain.SessionEvent]struct{})
	}
	s.subs[sessionID][ch] = struct{}{}
	s.subsMu.Unlock()

	unsubscribe := func() {
		s.subsMu.Lock()
		if m, ok := s.subs[sessionID]; ok {
			if _, exists := m[ch]; exists {
				delete(m, ch)
				close(ch)
			}
			if len(m) == 0 {
				delete(s.subs, sessionID)
			}
		}
		s.subsMu.Unlock()
	}
	return ch, unsubscribe
}

// HandleWorkerEvent processes an incoming worker event envelope from the node hub.
func (s *Service) HandleWorkerEvent(ctx context.Context, nodeID string, env node.Envelope) {
	if env.Type != node.EnvelopeTypeEvent || len(env.Payload) == 0 {
		return
	}

	var msg struct {
		SessionID string `json:"sessionId"`
		Kind      string `json:"kind"`
		Type      string `json:"type"`
		Reason    string `json:"reason"`
		Error     string `json:"error"`
	}
	if err := json.Unmarshal(env.Payload, &msg); err != nil || msg.SessionID == "" {
		return
	}

	sessionID, err := uuid.Parse(msg.SessionID)
	if err != nil {
		return
	}

	sess, err := s.Get(ctx, sessionID)
	if err != nil {
		return
	}
	if sess.NodeID.String() != nodeID {
		s.logger.WarnContext(ctx, "ignoring worker event from mismatched node",
			"node_id", nodeID,
			"session_id", sessionID.String(),
		)
		return
	}

	kind := msg.Kind
	if kind == "" {
		kind = msg.Type
	}
	if kind == "" {
		return
	}

	switch kind {
	case "session_completed":
		_, _, _ = s.Complete(ctx, sessionID)
	case "session_failed", "run_error":
		reason := msg.Reason
		if reason == "" {
			reason = msg.Error
		}
		_, _, _ = s.Fail(ctx, sessionID, reason)
	case "session_cancelled", "run_cancelled":
		_, _, _ = s.Cancel(ctx, sessionID, msg.Reason)
	default:
		_, _ = s.RecordEvent(ctx, sessionID, kind, env.Payload)
	}
}

func (s *Service) broadcast(evt domain.SessionEvent) {
	s.subsMu.RLock()
	m := s.subs[evt.SessionID]
	for ch := range m {
		select {
		case ch <- evt:
		default:
		}
	}
	s.subsMu.RUnlock()
}

func (s *Service) publishLifecycle(ctx context.Context, topic string, sess domain.Session) {
	if s.bus == nil {
		return
	}
	raw, err := json.Marshal(domain.SessionLifecyclePayload{
		SessionID: sess.ID,
		TaskID:    sess.TaskID,
		NodeID:    sess.NodeID,
		Status:    sess.Status,
	})
	if err != nil {
		return
	}
	_ = s.bus.Publish(ctx, eventbus.Event{
		ID:      uuid.NewString(),
		Type:    topic,
		Subject: sess.ID.String(),
		At:      s.now(),
		Payload: raw,
	})
}
