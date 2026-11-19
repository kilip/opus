package service

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/kilip/opus/internal/run/domain"
	"github.com/kilip/opus/internal/task"
	"github.com/kilip/opus/internal/worker"
)

const defaultQuotaPauseDuration = 15 * time.Minute

// FailRun transitions runID to failed, emits run.finished, moves the task back to todo
// with label vibe:failed and an activity authored vibe, and preserves the worktree for inspection.
func (s *Service) FailRun(ctx context.Context, cfg StageConfig, runID uuid.UUID, reason string) (domain.Run, error) {
	run, err := s.Transition(ctx, runID, domain.StateFailed, domain.TransitionInput{
		FailureReason: &reason,
	})
	if err != nil {
		return domain.Run{}, err
	}

	s.resetTaskToTodoWithFailedLabel(ctx, cfg, run.TaskID.String(), reason)
	return run, nil
}

// PauseOnQuota transitions runID to paused with paused_until set from resetAt
// (falling back to now + 15m if resetAt is nil or zero) and emits run.finished (outcome: paused).
func (s *Service) PauseOnQuota(ctx context.Context, runID uuid.UUID, resetAt *time.Time, reason string) (domain.Run, error) {
	var until time.Time
	if resetAt != nil && !resetAt.IsZero() {
		until = *resetAt
	} else {
		// ponytail: fallback 15m pause when provider reset timestamp is absent; refine per-provider backoff post-MVP.
		until = s.now().Add(defaultQuotaPauseDuration)
	}

	return s.Transition(ctx, runID, domain.StatePaused, domain.TransitionInput{
		PausedUntil:   &until,
		FailureReason: &reason,
	})
}

// ResumePausedRuns transitions any paused Run whose paused_until has elapsed back to queued.
func (s *Service) ResumePausedRuns(ctx context.Context) error {
	paused := domain.StatePaused
	runs, err := s.List(ctx, domain.ListFilter{State: &paused})
	if err != nil {
		return err
	}

	now := s.now()
	for _, r := range runs {
		if r.PausedUntil != nil && r.PausedUntil.After(now) {
			continue
		}
		if _, err := s.Transition(ctx, r.ID, domain.StateQueued, domain.TransitionInput{}); err != nil {
			return err
		}
	}
	return nil
}

// CancelRun sends CancelRun to the worker, stops any active session, cleans up the worktree,
// transitions the Run to cancelled (emitting run.finished), and moves the task to todo with vibe:failed.
func (s *Service) CancelRun(ctx context.Context, cfg StageConfig, runID uuid.UUID, reason string) (domain.Run, error) {
	run, err := s.Get(ctx, runID)
	if err != nil {
		return domain.Run{}, err
	}
	if reason == "" {
		reason = "cancelled by user"
	}

	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	cancelled, err := s.Transition(ctx, run.ID, domain.StateCancelled, domain.TransitionInput{
		FailureReason: &reason,
	})
	if err != nil {
		return domain.Run{}, err
	}

	if cfg.Worker != nil && run.NodeID != nil {
		if err := cfg.Worker.SendCancelRun(ctx, *run.NodeID, worker.CancelRun{
			RunID:  run.ID.String(),
			Reason: reason,
		}); err != nil {
			logger.WarnContext(ctx, "failed to send cancel_run to worker",
				slog.String("run_id", run.ID.String()),
				slog.Any("error", err),
			)
		}
	}

	if cfg.Sessions != nil && run.SessionID != nil {
		if _, _, err := cfg.Sessions.Cancel(ctx, *run.SessionID, reason); err != nil {
			logger.WarnContext(ctx, "failed to cancel session",
				slog.String("run_id", run.ID.String()),
				slog.String("session_id", run.SessionID.String()),
				slog.Any("error", err),
			)
		}
	}

	s.cleanupRunWorktree(ctx, cfg, run)
	s.resetTaskToTodoWithFailedLabel(ctx, cfg, run.TaskID.String(), reason)
	return cancelled, nil
}

// RecoverActiveRuns fails interrupted preparing/running runs on master restart so their tasks
// can be inspected or retried cleanly. Queued runs remain queued for reconciler pickup.
func (s *Service) RecoverActiveRuns(ctx context.Context, cfg StageConfig) error {
	var errs []error
	for _, st := range []domain.RunState{domain.StatePreparing, domain.StateRunning} {
		state := st
		runs, err := s.List(ctx, domain.ListFilter{State: &state})
		if err != nil {
			errs = append(errs, err)
			continue
		}
		for _, r := range runs {
			if cfg.Sessions != nil && r.SessionID != nil {
				_, _, _ = cfg.Sessions.MarkFailed(ctx, *r.SessionID, "interrupted by server restart")
			}
			if _, err := s.FailRun(ctx, cfg, r.ID, "interrupted by server restart"); err != nil {
				errs = append(errs, err)
				continue
			}
			s.cleanupRunWorktree(ctx, cfg, r)
		}
	}
	return errors.Join(errs...)
}

// EnforceStageTimeouts fails preparing/running runs that have exceeded timeout since their last activity.
func (s *Service) EnforceStageTimeouts(ctx context.Context, cfg StageConfig, timeout time.Duration) error {
	if timeout <= 0 {
		return nil
	}
	cutoff := s.now().Add(-timeout)
	var errs []error
	for _, st := range []domain.RunState{domain.StatePreparing, domain.StateRunning} {
		state := st
		runs, err := s.List(ctx, domain.ListFilter{State: &state})
		if err != nil {
			errs = append(errs, err)
			continue
		}
		for _, r := range runs {
			lastActivity := r.UpdatedAt
			if cfg.Sessions != nil && r.SessionID != nil {
				if ts, err := cfg.Sessions.LatestEventTime(ctx, *r.SessionID); err == nil && ts != nil && ts.After(lastActivity) {
					lastActivity = *ts
				}
			}
			if lastActivity.Before(cutoff) {
				if cfg.Sessions != nil && r.SessionID != nil {
					_, _, _ = cfg.Sessions.MarkFailed(ctx, *r.SessionID, "stage timeout exceeded")
				}
				if _, err := s.FailRun(ctx, cfg, r.ID, "stage timeout exceeded"); err != nil {
					errs = append(errs, err)
				}
			}
		}
	}
	return errors.Join(errs...)
}

func (s *Service) resetTaskToTodoWithFailedLabel(ctx context.Context, cfg StageConfig, taskID, reason string) {
	if cfg.Tasks == nil {
		return
	}
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	todo := task.StatusTodo
	if _, err := cfg.Tasks.Update(ctx, taskID, task.UpdateTaskRequest{Status: &todo}); err != nil {
		logger.ErrorContext(ctx, "failed to reset task to todo", slog.String("task_id", taskID), slog.Any("error", err))
	}
	if err := cfg.Tasks.AddLabel(ctx, taskID, labelVibeFailed); err != nil {
		logger.ErrorContext(ctx, "failed to add vibe:failed label", slog.String("task_id", taskID), slog.Any("error", err))
	}
	if reason != "" {
		if _, err := cfg.Tasks.AddActivity(ctx, taskID, task.ActivityKindComment, "vibe", reason); err != nil {
			logger.ErrorContext(ctx, "failed to add vibe activity", slog.String("task_id", taskID), slog.Any("error", err))
		}
	}
}
