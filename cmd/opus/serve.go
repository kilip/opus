package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync/atomic"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/kilip/opus/ent"
	"github.com/kilip/opus/internal/app"
	"github.com/kilip/opus/internal/node"
	"github.com/kilip/opus/internal/platform/eventbus"
	"github.com/kilip/opus/internal/project"
	"github.com/kilip/opus/internal/run"
	"github.com/kilip/opus/internal/session"
	"github.com/kilip/opus/internal/task"
	"github.com/kilip/opus/internal/tracker"
	"github.com/kilip/opus/internal/worker"
	"github.com/spf13/cobra"
)

type server struct {
	opus      *app.Opus
	fiber     *fiber.App
	bus       *eventbus.Bus
	nodes     *node.Node
	projects  *project.Project
	tasks     *task.Task
	sessions  *session.Session
	tracker   *tracker.Tracker
	runs      *run.Feature
	transport run.WorkerTransport
	masterID  string
	worker    *worker.Worker
	inProc    *worker.InProcessTransport
}

type serverOptions struct {
	trackerRunner   tracker.Runner
	onStartRun      worker.RunFunc
	capabilityProbe worker.CapabilityProbe
}

type serverOption func(*serverOptions)

func withTrackerRunner(r tracker.Runner) serverOption {
	return func(o *serverOptions) {
		o.trackerRunner = r
	}
}

func withOnStartRun(fn worker.RunFunc) serverOption {
	return func(o *serverOptions) {
		o.onStartRun = fn
	}
}

func withCapabilityProbe(fn worker.CapabilityProbe) serverOption {
	return func(o *serverOptions) {
		o.capabilityProbe = fn
	}
}

func buildServer(
	ctx context.Context,
	cfg app.Config,
	logger *slog.Logger,
	db *ent.Client,
	opts ...serverOption,
) (*server, error) {
	var o serverOptions
	for _, opt := range opts {
		if opt != nil {
			opt(&o)
		}
	}

	bus := eventbus.New(eventbus.WithLogger(logger))
	nodeFeature := node.New(db, node.Config{BaseURL: cfg.BaseURL, Bus: bus})
	if err := nodeFeature.Bootstrap(ctx); err != nil && logger != nil {
		logger.ErrorContext(ctx, "node registry bootstrap failed", "error", err)
	}
	masterID, _ := nodeFeature.MasterID(ctx)

	var (
		taskFeature    *task.Task
		trackerFeature *tracker.Tracker
	)
	projectFeature := project.New(db,
		project.WithNodeReader(nodeFeature),
		project.WithTaskCounter(func(ctx context.Context, projectID string) (int, error) {
			if taskFeature == nil {
				return 0, nil
			}
			return taskFeature.CountByProject(ctx, projectID)
		}),
		project.WithTrackerLabelValidator(func(ctx context.Context, labels map[string]string) error {
			if trackerFeature == nil {
				return nil
			}
			return trackerFeature.ValidateProjectLabels(ctx, labels)
		}),
	)
	taskFeature = task.New(db, bus, task.WithProjectChecker(projectFeature))

	// OnPromote is intentionally unset: run.ReconcileOnce gates on len(active running sessions) > 0
	// before calling PrepareAndStartSession/Enqueue, so queued sessions are unreachable from the
	// run reconciler in single-account MVP.
	sessionFeature := session.New(db, session.Config{Bus: bus})
	nodeFeature.Hub().OnEvent(func(nodeID string, env node.Envelope) {
		if env.Type == node.EnvelopeTypeEvent && len(env.Payload) > 0 {
			var evt worker.Event
			if json.Unmarshal(env.Payload, &evt) == nil && evt.Type == worker.EventCapabilities && evt.Capabilities != nil {
				nodeFeature.SetCapabilities(nodeID, *evt.Capabilities)
			}
		}
		sessionFeature.HandleWorkerEvent(ctx, nodeID, env)
	})

	trackerFeature = tracker.New(db, tracker.Config{
		PollInterval: cfg.TrackerPollInterval,
		Runner:       o.trackerRunner,
		Logger:       logger,
		Tasks:        taskFeature,
		Bus:          bus,
	})

	transport := &worker.HubTransport{
		Hub:      nodeFeature.Hub(),
		Git:      worker.NewGitManager(cfg.AppHome),
		MasterID: masterID,
	}
	runFeature := run.New(db, run.Config{
		Bus:          bus,
		AppHome:      cfg.AppHome,
		Tasks:        taskFeature,
		Projects:     projectFeature,
		Sessions:     sessionFeature,
		Worker:       transport,
		PollInterval: cfg.RunPollInterval,
		StageTimeout: cfg.RunStageTimeout,
		NodeGrace:    cfg.RunNodeGrace,
		Logger:       logger,
		IsAgyReady: func(_ context.Context, nodeID uuid.UUID) bool {
			return nodeFeature.IsAgyReady(nodeID.String())
		},
		IsNodeConnected: nodeFeature.Hub().Connected,
	})

	shuttingDown := &atomic.Bool{}
	fiberApp, err := app.GetFiber(cfg, logger, db, bus,
		app.WithNode(nodeFeature),
		app.WithProject(projectFeature),
		app.WithRun(runFeature),
		app.WithSession(sessionFeature),
		app.WithTask(taskFeature),
		app.WithTracker(trackerFeature),
		app.WithShuttingDown(shuttingDown),
	)
	if err != nil {
		return nil, err
	}

	inProc := worker.NewInProcess()
	masterWorker := worker.New(worker.Config{
		NodeID:          masterID,
		HomeDir:         cfg.AppHome,
		AgyBin:          cfg.WorkerAgyBin,
		InProcess:       inProc,
		Logger:          logger,
		CapabilityProbe: o.capabilityProbe,
		OnStartRun:      o.onStartRun,
	})

	opusApp := app.New(cfg, fiberApp, logger, db, bus,
		app.WithNode(nodeFeature),
		app.WithRun(runFeature),
		app.WithSession(sessionFeature),
		app.WithTracker(trackerFeature),
		app.WithWorker(masterWorker, inProc),
		app.WithShuttingDown(shuttingDown),
	)

	return &server{
		opus:      opusApp,
		fiber:     fiberApp,
		bus:       bus,
		nodes:     nodeFeature,
		projects:  projectFeature,
		tasks:     taskFeature,
		sessions:  sessionFeature,
		tracker:   trackerFeature,
		runs:      runFeature,
		transport: transport,
		masterID:  masterID,
		worker:    masterWorker,
		inProc:    inProc,
	}, nil
}

func newServeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Start HTTP server",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			cfg, err := app.GetConfig(ctx)
			if err != nil {
				return err
			}
			if cfg.Version == "" {
				cfg.Version = Version
			}
			logger := app.GetLogger(cfg)
			flushSentry := app.InitSentry(cfg, logger)
			defer flushSentry()

			db, err := app.OpenDBWithVersion(ctx, cfg.DatabaseURL, cfg.Version, cfg.AllowDowngrade)
			if err != nil {
				return err
			}
			srv, err := buildServer(ctx, cfg, logger, db)
			if err != nil {
				return err
			}
			return srv.opus.Serve(ctx)
		},
	}
}
