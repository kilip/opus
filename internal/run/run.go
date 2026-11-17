// Package run is the automated task execution Run feature. Other packages
// depend on this package only, never on its subpackages.
package run

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/kilip/opus/ent"
	"github.com/kilip/opus/internal/platform/eventbus"
	"github.com/kilip/opus/internal/run/domain"
	"github.com/kilip/opus/internal/run/repo"
	"github.com/kilip/opus/internal/run/service"
	"github.com/kilip/opus/internal/session"
	"github.com/kilip/opus/internal/task"
)

// Mode, state, and event constants re-exported from domain.
const (
	ModeVibe = domain.ModeVibe

	StateQueued    = domain.StateQueued
	StatePreparing = domain.StatePreparing
	StateRunning   = domain.StateRunning
	StateMerged    = domain.StateMerged
	StateFailed    = domain.StateFailed
	StateNeedsInfo = domain.StateNeedsInfo
	StatePaused    = domain.StatePaused
	StateCancelled = domain.StateCancelled

	EventStarted  = domain.EventStarted
	EventFinished = domain.EventFinished
)

// Sentinel errors re-exported from domain.
var (
	ErrRunNotFound       = domain.ErrRunNotFound
	ErrInvalidState      = domain.ErrInvalidState
	ErrInvalidMode       = domain.ErrInvalidMode
	ErrInvalidTransition = domain.ErrInvalidTransition
	ErrTaskIDRequired    = domain.ErrTaskIDRequired
	ErrProjectIDRequired = domain.ErrProjectIDRequired
)

// Re-exported domain types.
type (
	RunMode            = domain.RunMode
	RunState           = domain.RunState
	Run                = domain.Run
	CreateInput        = domain.CreateInput
	TransitionInput    = domain.TransitionInput
	ListFilter         = domain.ListFilter
	RunStartedPayload  = domain.RunStartedPayload
	RunFinishedPayload = domain.RunFinishedPayload
)

// Config configures the run feature.
type Config struct {
	Bus          *eventbus.Bus
	Now          func() time.Time
	Tasks        *task.Task
	Sessions     *session.Session
	PollInterval time.Duration
	Logger       *slog.Logger
	IsAgyReady   func(ctx context.Context, nodeID uuid.UUID) bool
}

// Feature is the public facade for the run feature.
type Feature struct {
	svc        *service.Service
	reconciler service.ReconcilerConfig
}

// New constructs the run feature wired to Ent and the shared event bus.
func New(db *ent.Client, cfg Config) *Feature {
	r := repo.New(db)
	svc := service.New(r, service.Config{
		Bus: cfg.Bus,
		Now: cfg.Now,
	})
	return &Feature{
		svc: svc,
		reconciler: service.ReconcilerConfig{
			Tasks:        cfg.Tasks,
			Sessions:     cfg.Sessions,
			PollInterval: cfg.PollInterval,
			Logger:       cfg.Logger,
			IsAgyReady:   cfg.IsAgyReady,
		},
	}
}

// Run starts the reconciler loop and blocks until ctx is cancelled.
func (f *Feature) Run(ctx context.Context) error {
	return f.svc.Run(ctx, f.reconciler)
}

// ReconcileOnce runs a single eligibility scan across tasks.
func (f *Feature) ReconcileOnce(ctx context.Context) error {
	return f.svc.ReconcileOnce(ctx, f.reconciler)
}

// Create creates a new Run in the queued state.
func (f *Feature) Create(ctx context.Context, in CreateInput) (Run, error) {
	return f.svc.Create(ctx, in)
}

// Get returns a Run by ID.
func (f *Feature) Get(ctx context.Context, id uuid.UUID) (Run, error) {
	return f.svc.Get(ctx, id)
}

// List returns Runs matching filter.
func (f *Feature) List(ctx context.Context, filter ListFilter) ([]Run, error) {
	return f.svc.List(ctx, filter)
}

// Transition validates and persists a state transition and emits lifecycle events.
func (f *Feature) Transition(ctx context.Context, id uuid.UUID, to RunState, in TransitionInput) (Run, error) {
	return f.svc.Transition(ctx, id, to, in)
}
