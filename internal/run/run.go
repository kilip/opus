// Package run is the automated task execution Run feature. Other packages
// depend on this package only, never on its subpackages.
package run

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/kilip/opus/ent"
	"github.com/kilip/opus/internal/platform/eventbus"
	"github.com/kilip/opus/internal/project"
	"github.com/kilip/opus/internal/run/domain"
	"github.com/kilip/opus/internal/run/handler"
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
	ErrInvalidRunID      = domain.ErrInvalidRunID
	ErrInvalidState      = domain.ErrInvalidState
	ErrInvalidMode       = domain.ErrInvalidMode
	ErrInvalidTransition = domain.ErrInvalidTransition
	ErrTaskIDRequired    = domain.ErrTaskIDRequired
	ErrProjectIDRequired = domain.ErrProjectIDRequired
	ErrMCPUnauthorized   = domain.ErrMCPUnauthorized
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
	RunResponse        = domain.RunResponse
	CancelRunRequest   = domain.CancelRunRequest
	WorkerTransport    = service.WorkerTransport
)

// Config configures the run feature.
type Config struct {
	Bus          *eventbus.Bus
	Now          func() time.Time
	AppHome      string
	Tasks        *task.Task
	Projects     *project.Project
	Sessions     *session.Session
	Worker       WorkerTransport
	PollInterval time.Duration
	StageTimeout time.Duration
	Logger       *slog.Logger
	IsAgyReady   func(ctx context.Context, nodeID uuid.UUID) bool
}

// Feature is the public facade for the run feature.
type Feature struct {
	svc        *service.Service
	handler    *handler.Handler
	stageCfg   service.StageConfig
	reconciler service.ReconcilerConfig
}

// New constructs the run feature wired to Ent and the shared event bus.
func New(db *ent.Client, cfg Config) *Feature {
	r := repo.New(db)
	svc := service.New(r, service.Config{
		Bus: cfg.Bus,
		Now: cfg.Now,
	})
	stageCfg := service.StageConfig{
		AppHome:  cfg.AppHome,
		Tasks:    cfg.Tasks,
		Projects: cfg.Projects,
		Sessions: cfg.Sessions,
		Worker:   cfg.Worker,
		Logger:   cfg.Logger,
	}
	return &Feature{
		svc:      svc,
		handler:  handler.New(svc, stageCfg),
		stageCfg: stageCfg,
		reconciler: service.ReconcilerConfig{
			Tasks:        cfg.Tasks,
			Sessions:     cfg.Sessions,
			Stages:       stageCfg,
			PollInterval: cfg.PollInterval,
			StageTimeout: cfg.StageTimeout,
			Logger:       cfg.Logger,
			IsAgyReady:   cfg.IsAgyReady,
		},
	}
}

// Register mounts the protected run HTTP routes on r.
func (f *Feature) Register(r fiber.Router) {
	f.handler.Register(r)
}

// Cancel cancels an active run, stops its session, and cleans up its worktree.
func (f *Feature) Cancel(ctx context.Context, id uuid.UUID, reason string) (Run, error) {
	return f.svc.CancelRun(ctx, f.stageCfg, id, reason)
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

// SetReport persists the structured report JSON on a Run.
func (f *Feature) SetReport(ctx context.Context, id uuid.UUID, report json.RawMessage) (Run, error) {
	return f.svc.SetReport(ctx, id, report)
}

// AuthenticateMCP verifies a run-scoped MCP Bearer token for an active run.
func (f *Feature) AuthenticateMCP(ctx context.Context, runID uuid.UUID, rawToken string) (Run, error) {
	return f.svc.AuthenticateMCP(ctx, runID, rawToken)
}
