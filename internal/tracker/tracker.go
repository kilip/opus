// Package tracker is the issue-tracker feature. Other packages depend on this package only,
// never on its subpackages.
package tracker

import (
	"context"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/opus/ent"
	"github.com/kilip/opus/internal/project"
	"github.com/kilip/opus/internal/task"
	"github.com/kilip/opus/internal/tracker/domain"
	"github.com/kilip/opus/internal/tracker/handler"
	"github.com/kilip/opus/internal/tracker/provider/github"
	"github.com/kilip/opus/internal/tracker/repo"
	"github.com/kilip/opus/internal/tracker/service"
)

// DefaultPollInterval is the default cadence between tracker sync cycles.
const DefaultPollInterval = service.DefaultPollInterval

// Sentinel errors re-exported from domain.
var (
	ErrRateLimited        = domain.ErrRateLimited
	ErrInvalidLabelConfig = domain.ErrInvalidLabelConfig
	ErrProjectNotFound    = domain.ErrProjectNotFound
	ErrInvalidProjectID   = domain.ErrInvalidProjectID
)

// Re-exported domain and provider types.
type (
	TrackerSync                  = domain.TrackerSync
	TrackerStatusResponse        = domain.TrackerStatusResponse
	TrackerSettingsResponse      = domain.TrackerSettingsResponse
	UpdateTrackerSettingsRequest = domain.UpdateTrackerSettingsRequest
	EffectiveLabelMap            = domain.EffectiveLabelMap
	Runner                       = github.Runner
)

// DefaultStatusLabels returns a copy of the built-in status-to-label map.
func DefaultStatusLabels() map[string]string {
	return domain.DefaultStatusLabels()
}

// Config configures the Tracker feature.
type Config struct {
	PollInterval time.Duration
	Runner       github.Runner
	Tasks        *task.Task
	Logger       *slog.Logger
	Now          func() time.Time
	OnStop       func()
}

// Tracker is the tracker feature facade.
type Tracker struct {
	svc     *service.Tracker
	poller  *service.Poller
	handler *handler.Handler
}

// New returns a new Tracker feature facade backed by Ent.
func New(client *ent.Client, cfg ...Config) *Tracker {
	var c Config
	if len(cfg) > 0 {
		c = cfg[0]
	}
	r := repo.New(client)
	svc := service.New(r)
	poller := service.NewPoller(r, service.PollerConfig{
		PollInterval: c.PollInterval,
		Runner:       c.Runner,
		Tasks:        c.Tasks,
		Logger:       c.Logger,
		Now:          c.Now,
		OnStop:       c.OnStop,
	})
	return &Tracker{
		svc:     svc,
		poller:  poller,
		handler: handler.New(svc),
	}
}

// Register mounts tracker HTTP routes on r.
func (t *Tracker) Register(r fiber.Router) {
	t.handler.Register(r)
}

// Start starts the background tracker poller.
func (t *Tracker) Start(ctx context.Context) error {
	if t.poller != nil {
		return t.poller.Start(ctx)
	}
	return nil
}

// Stop stops the background tracker poller and waits for any running cycle to complete.
func (t *Tracker) Stop() {
	if t.poller != nil {
		t.poller.Stop()
	}
}

// Running reports whether the background tracker poller is active.
func (t *Tracker) Running() bool {
	if t.poller != nil {
		return t.poller.Running()
	}
	return false
}

// SyncOnce executes a single tracker sync cycle synchronously.
func (t *Tracker) SyncOnce(ctx context.Context) {
	if t.poller != nil {
		t.poller.SyncOnce(ctx)
	}
}

// ValidateProjectLabels validates project-level status-label overrides against global settings and defaults.
func (t *Tracker) ValidateProjectLabels(ctx context.Context, projectLabels map[string]string) error {
	return t.svc.ValidateProjectLabels(ctx, projectLabels)
}

// EffectiveLabels returns the resolved bidirectional status-to-label map for a project.
func (t *Tracker) EffectiveLabels(ctx context.Context, proj project.Record) (EffectiveLabelMap, error) {
	return t.svc.EffectiveLabels(ctx, proj)
}
