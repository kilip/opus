package service

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kilip/opus/internal/platform/eventbus"
	"github.com/kilip/opus/internal/run/domain"
	"github.com/kilip/opus/internal/run/repo"
)

// Config configures the run service.
type Config struct {
	Bus *eventbus.Bus
	Now func() time.Time
}

// Service owns run lifecycle orchestration, state persistence, and event emission.
type Service struct {
	repo    *repo.Ent
	bus     *eventbus.Bus
	now     func() time.Time
	writeMu sync.Mutex
}

// New creates a new run Service.
func New(r *repo.Ent, cfg Config) *Service {
	now := cfg.Now
	if now == nil {
		now = time.Now
	}
	return &Service{
		repo: r,
		bus:  cfg.Bus,
		now:  now,
	}
}

// Create validates input and persists a new Run in the queued state.
func (s *Service) Create(ctx context.Context, in domain.CreateInput) (domain.Run, error) {
	if in.TaskID == uuid.Nil {
		return domain.Run{}, domain.ErrTaskIDRequired
	}
	if in.ProjectID == uuid.Nil {
		return domain.Run{}, domain.ErrProjectIDRequired
	}
	if in.Mode != "" && !domain.IsValidMode(in.Mode) {
		return domain.Run{}, domain.ErrInvalidMode
	}

	s.writeMu.Lock()
	run, err := s.repo.Create(ctx, in, s.now())
	s.writeMu.Unlock()
	if err != nil {
		return domain.Run{}, err
	}
	return run, nil
}

// Get returns a Run by ID.
func (s *Service) Get(ctx context.Context, id uuid.UUID) (domain.Run, error) {
	return s.repo.Get(ctx, id)
}

// List returns Runs matching filter.
func (s *Service) List(ctx context.Context, filter domain.ListFilter) ([]domain.Run, error) {
	return s.repo.List(ctx, filter)
}

// Transition validates and persists a state transition and emits lifecycle events.
func (s *Service) Transition(
	ctx context.Context,
	id uuid.UUID,
	to domain.RunState,
	in domain.TransitionInput,
) (domain.Run, error) {
	if !domain.IsValidState(to) {
		return domain.Run{}, domain.ErrInvalidState
	}

	s.writeMu.Lock()
	run, err := s.repo.Transition(ctx, id, to, in, s.now())
	s.writeMu.Unlock()
	if err != nil {
		return domain.Run{}, err
	}

	if to == domain.StatePreparing {
		s.publishStarted(ctx, run)
	}
	if domain.IsFinishedOutcome(to) {
		reason := ""
		if run.FailureReason != nil {
			reason = *run.FailureReason
		}
		s.publishFinished(ctx, run, to, reason)
	}
	return run, nil
}

// SetReport persists the structured report JSON on a Run.
func (s *Service) SetReport(ctx context.Context, id uuid.UUID, report json.RawMessage) (domain.Run, error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	return s.repo.SetReport(ctx, id, report, s.now())
}

// HashMCPToken returns the lowercase hex SHA-256 digest of a raw MCP run token.
func HashMCPToken(rawToken string) string {
	sum := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(sum[:])
}

// AuthenticateMCP verifies rawToken against the run's stored mcp_token_hash and ensures
// the run is in an active execution state (preparing or running).
func (s *Service) AuthenticateMCP(ctx context.Context, runID uuid.UUID, rawToken string) (domain.Run, error) {
	if runID == uuid.Nil || rawToken == "" {
		return domain.Run{}, domain.ErrMCPUnauthorized
	}
	r, err := s.Get(ctx, runID)
	if err != nil {
		return domain.Run{}, domain.ErrMCPUnauthorized
	}
	if r.State != domain.StatePreparing && r.State != domain.StateRunning {
		return domain.Run{}, domain.ErrMCPUnauthorized
	}
	if r.MCPTokenHash == nil || *r.MCPTokenHash == "" {
		return domain.Run{}, domain.ErrMCPUnauthorized
	}
	got := HashMCPToken(rawToken)
	if subtle.ConstantTimeCompare([]byte(got), []byte(*r.MCPTokenHash)) != 1 {
		return domain.Run{}, domain.ErrMCPUnauthorized
	}
	return r, nil
}

func (s *Service) publishStarted(ctx context.Context, run domain.Run) {
	s.publish(ctx, domain.EventStarted, run.ID, domain.RunStartedPayload{
		RunID:     run.ID,
		TaskID:    run.TaskID,
		ProjectID: run.ProjectID,
		Mode:      run.Mode,
	})
}

func (s *Service) publishFinished(ctx context.Context, run domain.Run, outcome domain.RunState, reason string) {
	s.publish(ctx, domain.EventFinished, run.ID, domain.RunFinishedPayload{
		RunID:   run.ID,
		TaskID:  run.TaskID,
		Mode:    run.Mode,
		Outcome: outcome,
		Reason:  reason,
	})
}

func (s *Service) publish(ctx context.Context, topic string, runID uuid.UUID, payload any) {
	if s.bus == nil {
		return
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return
	}
	_ = s.bus.Publish(ctx, eventbus.Event{
		ID:      uuid.NewString(),
		Type:    topic,
		Subject: runID.String(),
		At:      s.now(),
		Payload: raw,
	})
}
