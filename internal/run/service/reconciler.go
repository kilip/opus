package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kilip/opus/internal/node"
	"github.com/kilip/opus/internal/platform/eventbus"
	"github.com/kilip/opus/internal/run/domain"
	"github.com/kilip/opus/internal/session"
	"github.com/kilip/opus/internal/task"
)

const (
	defaultPollInterval = 30 * time.Second
	defaultNodeGrace    = 2 * time.Minute
)

// ReconcilerConfig configures the run reconciler loop and its sibling feature dependencies.
type ReconcilerConfig struct {
	Tasks           *task.Task
	Sessions        *session.Session
	Stages          StageConfig
	PollInterval    time.Duration
	StageTimeout    time.Duration
	NodeGrace       time.Duration
	Logger          *slog.Logger
	IsAgyReady      func(ctx context.Context, nodeID uuid.UUID) bool
	IsNodeConnected func(nodeID string) bool
}

// ReconcileOnce runs a single eligibility scan across tasks.
func (s *Service) ReconcileOnce(ctx context.Context, cfg ReconcilerConfig) error {
	if cfg.Tasks == nil {
		return nil
	}
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	if err := s.ResumePausedRuns(ctx); err != nil && ctx.Err() == nil {
		logger.ErrorContext(ctx, "resume paused runs failed", slog.Any("error", err))
	}

	stageCfg := cfg.Stages
	if stageCfg.Tasks == nil {
		stageCfg.Tasks = cfg.Tasks
	}
	if stageCfg.Sessions == nil {
		stageCfg.Sessions = cfg.Sessions
	}
	if cfg.NodeGrace > 0 {
		if err := s.enforceNodeDisconnectGrace(ctx, cfg, stageCfg); err != nil && ctx.Err() == nil {
			logger.ErrorContext(ctx, "enforce node disconnect grace failed", slog.Any("error", err))
		}
	}
	if cfg.StageTimeout > 0 {
		if err := s.EnforceStageTimeouts(ctx, stageCfg, cfg.StageTimeout); err != nil && ctx.Err() == nil {
			logger.ErrorContext(ctx, "enforce stage timeouts failed", slog.Any("error", err))
		}
	}

	if cfg.Sessions != nil {
		blocked, err := s.hasBlockingRunningSession(ctx, cfg)
		if err != nil {
			return err
		}
		// ponytail: single-account global FIFO slot in MVP; check per-account capacity when multi-account pooling lands.
		if blocked {
			return nil
		}
	}

	tasks, err := cfg.Tasks.List(ctx)
	if err != nil {
		return err
	}

	for _, t := range tasks {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err := s.reconcileTask(ctx, cfg, t, logger); err != nil {
			logger.ErrorContext(ctx, "reconcile task failed",
				slog.String("task_id", t.ID),
				slog.Any("error", err),
			)
		}
	}
	return nil
}

// Run starts the reconciler loop, running immediately at startup, on periodic ticks,
// and on task.received / task.status_changed bus events until ctx is cancelled.
func (s *Service) Run(ctx context.Context, cfg ReconcilerConfig) error {
	interval := cfg.PollInterval
	if interval <= 0 {
		interval = defaultPollInterval
	}
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	trigger := make(chan struct{}, 1)
	signal := func() {
		select {
		case trigger <- struct{}{}:
		default:
		}
	}

	if s.bus != nil {
		err := s.bus.Subscribe("run-reconciler", func(subCtx context.Context, e eventbus.Event) error {
			if e.Type == task.EventStatusChanged && cfg.Stages.Worker != nil {
				if err := s.HandleTaskStatusChanged(subCtx, cfg.Stages, e); err != nil {
					logger.ErrorContext(subCtx, "handle task status changed failed", slog.Any("error", err))
				}
			}
			signal()
			return nil
		}, task.EventReceived, task.EventStatusChanged)
		if err != nil && logger != nil {
			logger.WarnContext(ctx, "run reconciler bus subscribe failed", slog.Any("error", err))
		}

		if err := s.bus.Subscribe("run-session-watcher", func(subCtx context.Context, e eventbus.Event) error {
			if err := s.HandleSessionLifecycleEvent(subCtx, cfg.Stages, e); err != nil {
				logger.ErrorContext(subCtx, "handle session lifecycle event failed", slog.Any("error", err))
			}
			signal()
			return nil
		}, session.EventFailed, session.EventCancelled); err != nil && logger != nil {
			logger.WarnContext(ctx, "run session watcher bus subscribe failed", slog.Any("error", err))
		}

		if err := s.bus.Subscribe("run-node-watcher", func(_ context.Context, e eventbus.Event) error {
			s.HandleNodeLifecycleEvent(e)
			signal()
			return nil
		}, node.EventOffline, node.EventOnline); err != nil && logger != nil {
			logger.WarnContext(ctx, "run node watcher bus subscribe failed", slog.Any("error", err))
		}
	}

	if err := s.RecoverActiveRuns(ctx, cfg.Stages); err != nil && ctx.Err() == nil {
		logger.ErrorContext(ctx, "startup run recovery failed", slog.Any("error", err))
	}
	if err := s.ReconcileOnce(ctx, cfg); err != nil && ctx.Err() == nil {
		logger.ErrorContext(ctx, "initial reconcile scan failed", slog.Any("error", err))
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := s.ReconcileOnce(ctx, cfg); err != nil && ctx.Err() == nil {
				logger.ErrorContext(ctx, "periodic reconcile scan failed", slog.Any("error", err))
			}
		case <-trigger:
			if err := s.ReconcileOnce(ctx, cfg); err != nil && ctx.Err() == nil {
				logger.ErrorContext(ctx, "event-triggered reconcile scan failed", slog.Any("error", err))
			}
		}
	}
}

func (s *Service) reconcileTask(ctx context.Context, cfg ReconcilerConfig, t task.Record, logger *slog.Logger) error {
	if !isBasicEligible(t) {
		return nil
	}

	taskID, err := uuid.Parse(t.ID)
	if err != nil {
		return err
	}
	projectID, err := uuid.Parse(t.ProjectID)
	if err != nil {
		return err
	}

	hasActive, err := s.repo.HasActiveRun(ctx, taskID)
	if err != nil {
		return err
	}
	if hasActive {
		return nil
	}

	if blocked := s.isBlocked(ctx, cfg.Tasks, t.Description); blocked {
		return nil
	}

	if missing := lintTaskStructure(t.Description); len(missing) > 0 {
		return s.failStructureLint(ctx, cfg.Tasks, taskID, projectID, t.ID, missing, logger)
	}

	candidates, err := s.repo.ListProjectNodes(ctx, projectID)
	if err != nil {
		return err
	}
	selected, ok := selectNode(ctx, candidates, cfg.IsAgyReady)
	if !ok {
		return nil
	}

	created, err := s.Create(ctx, domain.CreateInput{
		TaskID:    taskID,
		ProjectID: projectID,
		NodeID:    &selected.NodeID,
		Mode:      domain.ModeVibe,
	})
	if err != nil {
		return err
	}
	if cfg.Stages.Worker != nil {
		if _, err := s.Claim(ctx, cfg.Stages, created.ID); err != nil {
			return err
		}
		if _, err := s.PrepareAndStartSession(ctx, cfg.Stages, created.ID); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) isBlocked(ctx context.Context, tasks *task.Task, description string) bool {
	refs := parseBlockedByIssues(description)
	for _, num := range refs {
		dep, err := tasks.FindByExternal(ctx, task.SourceGitHub, num)
		if err != nil || dep == nil || dep.Status != task.StatusDone {
			return true
		}
	}
	return false
}

func (s *Service) failStructureLint(
	ctx context.Context,
	tasks *task.Task,
	taskID, projectID uuid.UUID,
	rawTaskID string,
	missing []string,
	logger *slog.Logger,
) error {
	reason := fmt.Sprintf("task description missing %s", strings.Join(missing, ", "))

	run, err := s.Create(ctx, domain.CreateInput{
		TaskID:    taskID,
		ProjectID: projectID,
		Mode:      domain.ModeVibe,
	})
	if err != nil {
		return err
	}

	if _, err := s.Transition(ctx, run.ID, domain.StatePreparing, domain.TransitionInput{}); err != nil {
		return err
	}
	if _, err := s.Transition(ctx, run.ID, domain.StateNeedsInfo, domain.TransitionInput{
		FailureReason: &reason,
	}); err != nil {
		return err
	}

	if err := tasks.AddLabel(ctx, rawTaskID, labelVibeNeedsInfo); err != nil {
		logger.ErrorContext(ctx, "failed to add vibe:needs-info label", slog.String("task_id", rawTaskID), slog.Any("error", err))
	}
	if _, err := tasks.AddActivity(ctx, rawTaskID, task.ActivityKindComment, "vibe", reason); err != nil {
		logger.ErrorContext(ctx, "failed to add vibe activity", slog.String("task_id", rawTaskID), slog.Any("error", err))
	}
	return nil
}

// HandleSessionLifecycleEvent propagates session.failed and session.cancelled events to active runs.
func (s *Service) HandleSessionLifecycleEvent(ctx context.Context, cfg StageConfig, e eventbus.Event) error {
	if e.Type != session.EventFailed && e.Type != session.EventCancelled {
		return nil
	}
	var payload session.SessionLifecyclePayload
	if err := json.Unmarshal(e.Payload, &payload); err != nil || payload.SessionID == uuid.Nil {
		return nil
	}

	reason := "session failed"
	if e.Type == session.EventCancelled {
		reason = "session cancelled"
	}
	if cfg.Sessions != nil {
		if sess, err := cfg.Sessions.Get(ctx, payload.SessionID); err == nil && sess.FailureReason != nil && *sess.FailureReason != "" {
			reason = *sess.FailureReason
		}
	}

	for _, st := range []domain.RunState{domain.StatePreparing, domain.StateRunning} {
		state := st
		runs, err := s.List(ctx, domain.ListFilter{State: &state})
		if err != nil {
			return err
		}
		for _, r := range runs {
			if r.SessionID == nil || *r.SessionID != payload.SessionID {
				continue
			}
			if e.Type == session.EventFailed {
				if _, err := s.FailRun(ctx, cfg, r.ID, reason); err != nil {
					return err
				}
			} else {
				if _, err := s.Transition(ctx, r.ID, domain.StateCancelled, domain.TransitionInput{
					FailureReason: &reason,
				}); err != nil {
					return err
				}
				s.resetTaskToTodoWithFailedLabel(ctx, cfg, r.TaskID.String(), reason)
			}
			s.cleanupRunWorktree(ctx, cfg, r)
		}
	}
	return nil
}

// HandleNodeLifecycleEvent tracks node offline/online timestamps for disconnect grace enforcement.
func (s *Service) HandleNodeLifecycleEvent(e eventbus.Event) {
	var payload node.NodeEventPayload
	if err := json.Unmarshal(e.Payload, &payload); err != nil || payload.ID == "" {
		return
	}
	switch e.Type {
	case node.EventOffline:
		s.MarkNodeOffline(payload.ID)
	case node.EventOnline:
		s.ClearNodeOffline(payload.ID)
	}
}

func (s *Service) enforceNodeDisconnectGrace(ctx context.Context, cfg ReconcilerConfig, stageCfg StageConfig) error {
	now := s.now()
	for _, st := range []domain.RunState{domain.StatePreparing, domain.StateRunning} {
		state := st
		runs, err := s.List(ctx, domain.ListFilter{State: &state})
		if err != nil {
			return err
		}
		for _, r := range runs {
			if r.NodeID == nil {
				continue
			}
			nid := r.NodeID.String()
			if cfg.IsNodeConnected != nil && cfg.IsNodeConnected(nid) {
				s.ClearNodeOffline(nid)
				continue
			}
			since, offline := s.nodeOfflineSince(nid)
			if !offline || now.Sub(since) <= cfg.NodeGrace {
				continue
			}
			if stageCfg.Sessions != nil && r.SessionID != nil {
				_, _, _ = stageCfg.Sessions.MarkFailed(ctx, *r.SessionID, "worker node offline")
			}
			if _, err := s.FailRun(ctx, stageCfg, r.ID, "worker node offline"); err != nil {
				return err
			}
			s.cleanupRunWorktree(ctx, stageCfg, r)
		}
	}
	return nil
}

func (s *Service) hasBlockingRunningSession(ctx context.Context, cfg ReconcilerConfig) (bool, error) {
	running := session.StatusRunning
	active, err := cfg.Sessions.List(ctx, session.ListFilter{Status: &running})
	if err != nil {
		return false, err
	}
	if len(active) == 0 {
		return false, nil
	}

	terminalRunSessions := make(map[uuid.UUID]struct{})
	for _, st := range []domain.RunState{domain.StateMerged, domain.StateFailed, domain.StateNeedsInfo, domain.StateCancelled} {
		state := st
		runs, err := s.List(ctx, domain.ListFilter{State: &state})
		if err != nil {
			return false, err
		}
		for _, r := range runs {
			if r.SessionID != nil {
				terminalRunSessions[*r.SessionID] = struct{}{}
			}
		}
	}

	now := s.now()
	grace := cfg.NodeGrace
	if grace <= 0 {
		grace = defaultNodeGrace
	}
	for _, sess := range active {
		if _, terminal := terminalRunSessions[sess.ID]; terminal {
			continue
		}
		nid := sess.NodeID.String()
		if cfg.IsNodeConnected != nil && !cfg.IsNodeConnected(nid) {
			if since, offline := s.nodeOfflineSince(nid); offline && now.Sub(since) > grace {
				continue
			}
		}
		return true, nil
	}
	return false, nil
}
