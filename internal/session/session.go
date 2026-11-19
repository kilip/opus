// Package session is the agent session feature. Other packages depend on this
// package only, never on its subpackages.
package session

import (
	"context"
	"encoding/json"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/kilip/opus/ent"
	"github.com/kilip/opus/internal/node"
	"github.com/kilip/opus/internal/platform/eventbus"
	"github.com/kilip/opus/internal/session/domain"
	"github.com/kilip/opus/internal/session/handler"
	"github.com/kilip/opus/internal/session/repo"
	"github.com/kilip/opus/internal/session/service"
)

// Status constants re-exported from domain.
const (
	StatusQueued    = domain.StatusQueued
	StatusRunning   = domain.StatusRunning
	StatusCompleted = domain.StatusCompleted
	StatusFailed    = domain.StatusFailed
	StatusCancelled = domain.StatusCancelled

	EventQueued    = domain.EventQueued
	EventStarted   = domain.EventStarted
	EventCompleted = domain.EventCompleted
	EventFailed    = domain.EventFailed
	EventCancelled = domain.EventCancelled
)

// Sentinel errors re-exported from domain.
var (
	ErrSessionNotFound  = domain.ErrSessionNotFound
	ErrInvalidStatus    = domain.ErrInvalidStatus
	ErrProviderRequired = domain.ErrProviderRequired
	ErrInvalidSessionID = domain.ErrInvalidSessionID
	ErrEventKindEmpty   = domain.ErrEventKindEmpty
)

// Re-exported domain types.
type (
	Status                  = domain.SessionStatus
	SessionStatus           = domain.SessionStatus
	SessionRecord           = domain.Session
	SessionEvent            = domain.SessionEvent
	SessionResponse         = domain.SessionResponse
	SessionEventResponse    = domain.SessionEventResponse
	CreateInput             = domain.CreateInput
	ListFilter              = domain.ListFilter
	SessionLifecyclePayload = domain.SessionLifecyclePayload
)

// Config configures the session feature.
type Config struct {
	Bus       *eventbus.Bus
	Now       func() time.Time
	OnPromote func(domain.Session)
}

// Session is the public facade for the session feature.
type Session struct {
	svc     *service.Service
	handler *handler.Handler
}

// New creates a new Session feature facade backed by Ent.
func New(client *ent.Client, cfg Config) *Session {
	r := repo.New(client)
	svc := service.New(r, service.Config{
		Bus:       cfg.Bus,
		Now:       cfg.Now,
		OnPromote: cfg.OnPromote,
	})
	return &Session{
		svc:     svc,
		handler: handler.New(svc),
	}
}

// Register mounts the protected session HTTP routes on r.
func (s *Session) Register(r fiber.Router) {
	s.handler.Register(r)
}

// Enqueue creates a session, starting immediately or queueing in FIFO order.
func (s *Session) Enqueue(ctx context.Context, in CreateInput) (SessionRecord, error) {
	return s.svc.Enqueue(ctx, in)
}

// RecordEvent persists an ordered SessionEvent and broadcasts it to SSE subscribers.
func (s *Session) RecordEvent(ctx context.Context, sessionID uuid.UUID, kind string, payload json.RawMessage) (SessionEvent, error) {
	return s.svc.RecordEvent(ctx, sessionID, kind, payload)
}

// Complete marks sessionID as completed and promotes the next queued session if any.
func (s *Session) Complete(ctx context.Context, sessionID uuid.UUID) (SessionRecord, *SessionRecord, error) {
	return s.svc.Complete(ctx, sessionID)
}

// Fail marks sessionID as failed and promotes the next queued session if any.
func (s *Session) Fail(ctx context.Context, sessionID uuid.UUID, reason string) (SessionRecord, *SessionRecord, error) {
	return s.svc.Fail(ctx, sessionID, reason)
}

// MarkFailed is an alias for Fail for run recovery callers.
func (s *Session) MarkFailed(ctx context.Context, sessionID uuid.UUID, reason string) (SessionRecord, *SessionRecord, error) {
	return s.svc.Fail(ctx, sessionID, reason)
}

// Cancel marks sessionID as cancelled and promotes the next queued session if any.
func (s *Session) Cancel(ctx context.Context, sessionID uuid.UUID, reason string) (SessionRecord, *SessionRecord, error) {
	return s.svc.Cancel(ctx, sessionID, reason)
}

// Get returns a session by ID.
func (s *Session) Get(ctx context.Context, id uuid.UUID) (SessionRecord, error) {
	return s.svc.Get(ctx, id)
}

// List returns sessions matching filter.
func (s *Session) List(ctx context.Context, filter ListFilter) ([]SessionRecord, error) {
	return s.svc.List(ctx, filter)
}

// ListEventsAfter returns ordered SessionEvents for sessionID with seq > afterSeq.
func (s *Session) ListEventsAfter(ctx context.Context, sessionID uuid.UUID, afterSeq int) ([]SessionEvent, error) {
	return s.svc.ListEventsAfter(ctx, sessionID, afterSeq)
}

// LatestEventTime returns the timestamp of the latest event for sessionID, or nil if none exist.
func (s *Session) LatestEventTime(ctx context.Context, sessionID uuid.UUID) (*time.Time, error) {
	return s.svc.LatestEventTime(ctx, sessionID)
}

// HandleWorkerEvent processes an incoming worker event envelope from the node hub.
func (s *Session) HandleWorkerEvent(ctx context.Context, nodeID string, env node.Envelope) {
	s.svc.HandleWorkerEvent(ctx, nodeID, env)
}
