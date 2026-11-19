package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/kilip/opus/internal/platform/eventbus"
	"github.com/kilip/opus/internal/project"
	"github.com/kilip/opus/internal/run/domain"
	"github.com/kilip/opus/internal/session"
	"github.com/kilip/opus/internal/task"
	"github.com/kilip/opus/internal/worker"
)

var nonAlnumRe = regexp.MustCompile(`[^a-z0-9]+`)

// WorkerTransport abstracts worker command dispatch and worktree cleanup for run stages.
type WorkerTransport interface {
	SendStartRun(ctx context.Context, nodeID uuid.UUID, cmd worker.StartRun) error
	SendCancelRun(ctx context.Context, nodeID uuid.UUID, cmd worker.CancelRun) error
	CleanupWorktree(ctx context.Context, nodeID uuid.UUID, projectSlug, taskKey, branch, mappedPath string) error
}

// StageConfig configures stage execution dependencies.
type StageConfig struct {
	AppHome  string
	Tasks    *task.Task
	Projects *project.Project
	Sessions *session.Session
	Worker   WorkerTransport
	Logger   *slog.Logger
}

// Claim transitions a queued Run to preparing, sets the task to in-progress, and emits run.started.
func (s *Service) Claim(ctx context.Context, cfg StageConfig, runID uuid.UUID) (domain.Run, error) {
	run, err := s.Transition(ctx, runID, domain.StatePreparing, domain.TransitionInput{
		IncrementAttempt: true,
	})
	if err != nil {
		return domain.Run{}, err
	}

	if cfg.Tasks != nil {
		inProgress := task.StatusInProgress
		if _, err := cfg.Tasks.Update(ctx, run.TaskID.String(), task.UpdateTaskRequest{
			Status: &inProgress,
		}); err != nil {
			return domain.Run{}, err
		}
	}
	return run, nil
}

// PrepareAndStartSession builds the branch/worktree/prompt/policy, transitions Run to running,
// enqueues the agy session, and dispatches StartRun to the worker.
func (s *Service) PrepareAndStartSession(ctx context.Context, cfg StageConfig, runID uuid.UUID) (domain.Run, error) {
	run, err := s.Get(ctx, runID)
	if err != nil {
		return domain.Run{}, err
	}
	if run.NodeID == nil {
		return domain.Run{}, fmt.Errorf("run %s has no assigned node", run.ID)
	}
	if cfg.Tasks == nil || cfg.Projects == nil || cfg.Sessions == nil || cfg.Worker == nil {
		return domain.Run{}, fmt.Errorf("stage dependencies not configured")
	}

	t, err := cfg.Tasks.Get(ctx, run.TaskID.String())
	if err != nil {
		return domain.Run{}, err
	}
	proj, err := cfg.Projects.Get(ctx, run.ProjectID.String())
	if err != nil {
		return domain.Run{}, err
	}

	branch := run.Branch
	if branch == "" {
		branch = deriveBranch(t)
	}

	mappedPath := s.findMappedPath(ctx, run.ProjectID, *run.NodeID)
	worktreeRoot := filepath.Join(cfg.AppHome, "wt", proj.Slug, t.Key)

	prompt, err := BuildPrompt(run.Mode, t)
	if err != nil {
		return domain.Run{}, err
	}
	pol, err := BuildPolicy(run.Mode, worktreeRoot)
	if err != nil {
		return domain.Run{}, err
	}

	sess, err := cfg.Sessions.Enqueue(ctx, session.CreateInput{
		TaskID:   run.TaskID,
		NodeID:   *run.NodeID,
		Provider: "agy",
	})
	if err != nil {
		return domain.Run{}, err
	}

	rawToken, tokenHash, err := generateMCPToken()
	if err != nil {
		return domain.Run{}, err
	}

	run, err = s.Transition(ctx, run.ID, domain.StateRunning, domain.TransitionInput{
		SessionID:    &sess.ID,
		Branch:       &branch,
		MCPTokenHash: &tokenHash,
	})
	if err != nil {
		return domain.Run{}, err
	}

	cmd := worker.StartRun{
		RunID:         run.ID.String(),
		SessionID:     sess.ID.String(),
		TaskID:        t.ID,
		TaskKey:       t.Key,
		ProjectID:     proj.ID,
		ProjectSlug:   proj.Slug,
		RepoURL:       buildRepoURL(proj.RepoProvider, proj.RepoRef),
		DefaultBranch: proj.DefaultBranch,
		MappedPath:    mappedPath,
		Branch:        branch,
		Prompt:        prompt,
		Policy:        &pol,
		McpToken:      rawToken,
	}
	if err := cfg.Worker.SendStartRun(ctx, *run.NodeID, cmd); err != nil {
		return domain.Run{}, err
	}
	return run, nil
}

func generateMCPToken() (rawToken, tokenHash string, err error) {
	var buf [32]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", "", fmt.Errorf("generate mcp token: %w", err)
	}
	rawToken = base64.RawURLEncoding.EncodeToString(buf[:])
	return rawToken, HashMCPToken(rawToken), nil
}

// CompleteAndCleanup cleans up the worker worktree, transitions Run to merged, and emits run.finished.
func (s *Service) CompleteAndCleanup(ctx context.Context, cfg StageConfig, runID uuid.UUID) (domain.Run, error) {
	run, err := s.Get(ctx, runID)
	if err != nil {
		return domain.Run{}, err
	}
	if cfg.Tasks != nil && cfg.Projects != nil && cfg.Worker != nil && run.NodeID != nil {
		t, tErr := cfg.Tasks.Get(ctx, run.TaskID.String())
		proj, pErr := cfg.Projects.Get(ctx, run.ProjectID.String())
		if tErr == nil && pErr == nil {
			mappedPath := s.findMappedPath(ctx, run.ProjectID, *run.NodeID)
			if err := cfg.Worker.CleanupWorktree(ctx, *run.NodeID, proj.Slug, t.Key, run.Branch, mappedPath); err != nil {
				logger := cfg.Logger
				if logger == nil {
					logger = slog.Default()
				}
				logger.WarnContext(ctx, "worktree cleanup failed",
					slog.String("run_id", run.ID.String()),
					slog.Any("error", err),
				)
			}
		}
	}

	return s.Transition(ctx, run.ID, domain.StateMerged, domain.TransitionInput{})
}

// HandleTaskStatusChanged reacts to task.status_changed events, completing any running Run
// when the task reaches done.
func (s *Service) HandleTaskStatusChanged(ctx context.Context, cfg StageConfig, e eventbus.Event) error {
	if e.Type != task.EventStatusChanged || len(e.Payload) == 0 {
		return nil
	}
	var payload task.TaskStatusChangedPayload
	if err := json.Unmarshal(e.Payload, &payload); err != nil {
		return nil
	}
	if payload.To != task.StatusDone {
		return nil
	}

	taskID, err := uuid.Parse(payload.ID)
	if err != nil {
		return nil
	}
	running := domain.StateRunning
	runs, err := s.List(ctx, domain.ListFilter{
		TaskID: &taskID,
		State:  &running,
	})
	if err != nil {
		return err
	}
	for _, r := range runs {
		if _, err := s.CompleteAndCleanup(ctx, cfg, r.ID); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) findMappedPath(ctx context.Context, projectID, nodeID uuid.UUID) string {
	candidates, err := s.repo.ListProjectNodes(ctx, projectID)
	if err != nil {
		return ""
	}
	for _, c := range candidates {
		if c.NodeID == nodeID {
			return c.MappedPath
		}
	}
	return ""
}

func deriveBranch(t task.Record) string {
	branchType := "feat"
	for _, l := range t.Labels {
		lower := strings.ToLower(strings.TrimSpace(l))
		if strings.HasPrefix(lower, "type:") {
			candidate := strings.TrimSpace(strings.TrimPrefix(lower, "type:"))
			candidate = nonAlnumRe.ReplaceAllString(candidate, "-")
			candidate = strings.Trim(candidate, "-")
			if candidate != "" {
				branchType = candidate
				break
			}
		}
	}

	slug := nonAlnumRe.ReplaceAllString(strings.ToLower(t.Title), "-")
	slug = strings.Trim(slug, "-")
	if len(slug) > 50 {
		slug = strings.TrimRight(slug[:50], "-")
	}
	if slug == "" {
		slug = strings.ToLower(t.Key)
	}
	if slug == "" {
		slug = "task"
	}
	return branchType + "/" + slug
}

func buildRepoURL(provider, ref string) string {
	if strings.Contains(ref, "://") || strings.HasPrefix(ref, "git@") {
		return ref
	}
	if provider == "github" && ref != "" {
		return "https://github.com/" + ref + ".git"
	}
	return ref
}
