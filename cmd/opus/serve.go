package main

import (
	"context"

	"github.com/kilip/opus/internal/app"
	"github.com/kilip/opus/internal/node"
	"github.com/kilip/opus/internal/platform/eventbus"
	"github.com/kilip/opus/internal/project"
	"github.com/kilip/opus/internal/task"
	"github.com/kilip/opus/internal/tracker"
	"github.com/kilip/opus/internal/worker"
	"github.com/spf13/cobra"
)

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
			logger := app.GetLogger(cfg)
			flushSentry := app.InitSentry(cfg, logger)
			defer flushSentry()

			db, err := app.OpenDB(ctx, cfg.DatabaseURL)
			if err != nil {
				return err
			}
			bus := eventbus.New(eventbus.WithLogger(logger))
			nodeFeature := node.New(db, node.Config{BaseURL: cfg.BaseURL, Bus: bus})
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
			trackerFeature = tracker.New(db, tracker.Config{
				PollInterval: cfg.TrackerPollInterval,
				Logger:       logger,
				Tasks:        taskFeature,
			})
			fiberApp := app.GetFiber(cfg, logger, db, bus,
				app.WithNode(nodeFeature),
				app.WithProject(projectFeature),
				app.WithTask(taskFeature),
				app.WithTracker(trackerFeature),
			)
			masterID, _ := nodeFeature.MasterID(ctx)
			inProc := worker.NewInProcess()
			masterWorker := worker.New(worker.Config{
				NodeID:    masterID,
				HomeDir:   cfg.AppHome,
				AgyBin:    cfg.WorkerAgyBin,
				InProcess: inProc,
				Logger:    logger,
			})
			return app.New(cfg, fiberApp, logger, db, bus,
				app.WithNode(nodeFeature),
				app.WithTracker(trackerFeature),
				app.WithWorker(masterWorker, inProc),
			).Serve(ctx)
		},
	}
}
