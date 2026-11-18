package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kilip/opus/internal/platform/eventbus"
	"github.com/kilip/opus/internal/run/domain"
	"github.com/kilip/opus/internal/session"
	"github.com/kilip/opus/internal/task"
)

const defaultPollInterval = 30 * time.Second

// ReconcilerConfig configures the run reconciler loop and its sibling feature dependencies.
type ReconcilerConfig struct {
	Tasks        *task.Task
	Sessions     *session.Session
	Stages       StageConfig
	PollInterval time.Duration
	StageTimeout time.Duration
	Logger       *slog.Logger
	IsAgyReady   func(ctx context.Context, nodeID uuid.UUID) bool
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
	if cfg.StageTimeout > 0 {
		stageCfg := cfg.Stages
		if stageCfg.Tasks == nil {
			stageCfg.Tasks = cfg.Tasks
		}
		if stageCfg.Sessions == nil {
			stageCfg.Sessions = cfg.Sessions
		}
		if err := s.EnforceStageTimeouts(ctx, stageCfg, cfg.StageTimeout); err != nil && ctx.Err() == nil {
			logger.ErrorContext(ctx, "enforce stage timeouts failed", slog.Any("error", err))
		}
	}

	if cfg.Sessions != nil {
		running := session.StatusRunning
		active, err := cfg.Sessions.List(ctx, session.ListFilter{Status: &running})
		if err != nil {
			return err
		}
		// ponytail: single-account global FIFO slot in MVP; check per-account capacity when multi-account pooling lands.
		if len(active) > 0 {
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
		err := s.bus.Subscribe("run-reconciler", func(_ context.Context, _ eventbus.Event) error {
			signal()
			return nil
		}, task.EventReceived, task.EventStatusChanged)
		if err != nil && logger != nil {
			logger.WarnContext(ctx, "run reconciler bus subscribe failed", slog.Any("error", err))
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
			if err := s.RecoverActiveRuns(ctx, cfg.Stages); err != nil && ctx.Err() == nil {
				logger.ErrorContext(ctx, "startup run recovery failed", slog.Any("error", err))
			}
			if err := s.ReconcileOnce(ctx, cfg); err != nil && ctx.Err() == nil {
				logger.ErrorContext(ctx, "periodic reconcile scan failed", slog.Any("error", err))
			}
		case <-trigger:
			if err := s.RecoverActiveRuns(ctx, cfg.Stages); err != nil && ctx.Err() == nil {
				logger.ErrorContext(ctx, "startup run recovery failed", slog.Any("error", err))
			}
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

	_, err = s.Create(ctx, domain.CreateInput{
		TaskID:    taskID,
		ProjectID: projectID,
		NodeID:    &selected.NodeID,
		Mode:      domain.ModeVibe,
	})
	return err
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
