package service

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/kilip/opus/internal/project"
	"github.com/kilip/opus/internal/task"
	"github.com/kilip/opus/internal/tracker/provider/github"
	"github.com/kilip/opus/internal/tracker/repo"
)

// DefaultPollInterval is the default cadence between tracker sync cycles.
const DefaultPollInterval = 60 * time.Second

// PollerConfig configures the background tracker poller.
type PollerConfig struct {
	PollInterval time.Duration
	Runner       github.Runner
	Tasks        *task.Task
	Logger       *slog.Logger
	Now          func() time.Time
	OnStop       func()
}

// Poller periodically syncs tracker state for tracked projects.
type Poller struct {
	repo         *repo.Ent
	runner       github.Runner
	tasks        *task.Task
	pollInterval time.Duration
	logger       *slog.Logger
	now          func() time.Time
	onStop       func()

	mu      sync.Mutex
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	running bool
}

// NewPoller creates a new Poller.
func NewPoller(r *repo.Ent, cfg PollerConfig) *Poller {
	interval := cfg.PollInterval
	if interval <= 0 {
		interval = DefaultPollInterval
	}
	runner := cfg.Runner
	if runner == nil {
		runner = github.NewRunner()
	}
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	now := cfg.Now
	if now == nil {
		now = time.Now
	}

	return &Poller{
		repo:         r,
		runner:       runner,
		tasks:        cfg.Tasks,
		pollInterval: interval,
		logger:       logger,
		now:          now,
		onStop:       cfg.OnStop,
	}
}

// Start runs one sync cycle immediately and then polls on PollInterval until Stop or ctx cancellation.
func (p *Poller) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.running {
		return nil
	}

	runCtx, cancel := context.WithCancel(ctx)
	p.cancel = cancel
	p.running = true
	p.wg.Add(1)

	go func() {
		defer p.wg.Done()
		p.SyncOnce(runCtx)

		ticker := time.NewTicker(p.pollInterval)
		defer ticker.Stop()

		for {
			select {
			case <-runCtx.Done():
				return
			case <-ticker.C:
				p.SyncOnce(runCtx)
			}
		}
	}()

	return nil
}

// Stop stops the poller loop and waits for any active sync cycle to finish.
func (p *Poller) Stop() {
	p.mu.Lock()
	if !p.running {
		p.mu.Unlock()
		return
	}
	cancel := p.cancel
	p.running = false
	p.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	p.wg.Wait()
	if p.onStop != nil {
		p.onStop()
	}
}

// Running returns true while the background poller is active.
func (p *Poller) Running() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.running
}

// SyncOnce executes a single tracker sync cycle.
func (p *Poller) SyncOnce(ctx context.Context) {
	projects, err := p.repo.ListProjects(ctx)
	if err != nil {
		if ctx.Err() == nil {
			p.logger.ErrorContext(ctx, "tracker list projects failed", "error", err)
		}
		return
	}

	var ghProjects []project.Record
	for _, proj := range projects {
		if proj.TrackerProvider == "github" {
			ghProjects = append(ghProjects, proj)
		}
	}
	if len(ghProjects) == 0 {
		return
	}

	if err := github.CheckAuth(ctx, p.runner); err != nil {
		if ctx.Err() != nil {
			return
		}
		p.logger.ErrorContext(ctx, "github auth status failed", "error", err)
		for _, proj := range ghProjects {
			if saveErr := p.repo.UpsertLastError(ctx, proj.ID, err.Error()); saveErr != nil {
				p.logger.ErrorContext(ctx, "failed to record tracker auth error", "project_id", proj.ID, "error", saveErr)
			}
		}
		return
	}

	for _, proj := range ghProjects {
		if ctx.Err() != nil {
			return
		}
		if clearErr := p.repo.ClearLastError(ctx, proj.ID); clearErr != nil {
			p.logger.ErrorContext(ctx, "failed to clear tracker last error", "project_id", proj.ID, "error", clearErr)
		}
	}

	for _, proj := range ghProjects {
		if ctx.Err() != nil {
			return
		}
		p.syncProject(ctx, proj)
	}
}

func (p *Poller) syncProject(ctx context.Context, proj project.Record) {
	syncState, err := p.repo.GetSyncByProject(ctx, proj.ID)
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		p.recordSyncError(ctx, "tracker get sync state failed", proj, err)
		return
	}

	issues, err := github.ListIssues(ctx, p.runner, proj.TrackerRef, syncState.Cursor)
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		p.recordSyncError(ctx, "tracker sync issues failed", proj, err)
		return
	}

	prs, err := github.ListPRs(ctx, p.runner, proj.TrackerRef, syncState.Cursor)
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		p.recordSyncError(ctx, "tracker sync prs failed", proj, err)
		return
	}

	now := p.now().UTC()
	if p.tasks != nil && len(issues) > 0 {
		if err := p.syncIssues(ctx, proj, issues, now); err != nil {
			if ctx.Err() != nil {
				return
			}
			p.recordSyncError(ctx, "tracker inbound issue sync failed", proj, err)
			return
		}
	}

	if len(prs) > 0 {
		if err := p.syncPRs(ctx, proj, prs); err != nil {
			if ctx.Err() != nil {
				return
			}
			p.recordSyncError(ctx, "tracker inbound pr sync failed", proj, err)
			return
		}
	}

	if p.tasks != nil {
		if err := p.reconcileProjectOutbound(ctx, proj); err != nil {
			if ctx.Err() != nil {
				return
			}
			p.recordSyncError(ctx, "tracker outbound sync failed", proj, err)
			return
		}
	}

	var maxUpdated time.Time
	for _, iss := range issues {
		if iss.UpdatedAt.After(maxUpdated) {
			maxUpdated = iss.UpdatedAt
		}
	}
	for _, pr := range prs {
		if pr.UpdatedAt.After(maxUpdated) {
			maxUpdated = pr.UpdatedAt
		}
	}

	cursor := syncState.Cursor
	if !maxUpdated.IsZero() {
		cursor = maxUpdated.UTC().Format(time.RFC3339)
	} else if cursor == "" {
		cursor = now.Format(time.RFC3339)
	}

	if _, err := p.repo.SaveSyncSuccess(ctx, proj.ID, cursor, now); err != nil && ctx.Err() == nil {
		p.logger.ErrorContext(ctx, "failed to save tracker sync state", "project_id", proj.ID, "error", err)
	}
}

func (p *Poller) recordSyncError(ctx context.Context, msg string, proj project.Record, err error) {
	p.logger.WarnContext(ctx, msg, "project_id", proj.ID, "repo", proj.TrackerRef, "error", err)
	if saveErr := p.repo.UpsertLastError(ctx, proj.ID, err.Error()); saveErr != nil {
		p.logger.ErrorContext(ctx, "failed to store tracker error", "project_id", proj.ID, "error", saveErr)
	}
}
