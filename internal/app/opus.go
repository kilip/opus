package app

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/opus/ent"
	"github.com/kilip/opus/internal/node"
	"github.com/kilip/opus/internal/platform/eventbus"
	"github.com/kilip/opus/internal/platform/mailer"
	"github.com/kilip/opus/internal/project"
	"github.com/kilip/opus/internal/task"
	"github.com/kilip/opus/internal/tracker"
	"github.com/kilip/opus/internal/worker"
)

// NewMailer creates a mailer.Mailer based on configuration.
// If SMTPHost is empty, it returns a console mailer and logs a startup warning.
// Otherwise, it returns an SMTP mailer configured with SMTP settings.
func NewMailer(cfg Config, logger *slog.Logger) mailer.Mailer {
	if cfg.SMTPHost == "" {
		if logger != nil {
			logger.Warn("APP_SMTP_HOST not set, falling back to console mailer")
		}
		return mailer.NewConsole(logger)
	}
	return mailer.NewSMTP(mailer.SMTPConfig{
		Host:     cfg.SMTPHost,
		Port:     cfg.SMTPPort,
		Username: cfg.SMTPUsername,
		Password: cfg.SMTPPassword,
		From:     cfg.SMTPFrom,
	})
}

// Option configures optional feature instances wired into App/Opus.
type Option func(*options)

type options struct {
	node    *node.Node
	project *project.Project
	task    *task.Task
	tracker *tracker.Tracker
	worker  *worker.Worker
	inProc  *worker.InProcessTransport
}

// WithNode injects a pre-configured Node feature into GetFiber and Opus.
func WithNode(n *node.Node) Option {
	return func(o *options) {
		o.node = n
	}
}

// WithProject injects a pre-configured Project feature into GetFiber.
func WithProject(p *project.Project) Option {
	return func(o *options) {
		o.project = p
	}
}

// WithTask injects a pre-configured Task feature into GetFiber.
func WithTask(t *task.Task) Option {
	return func(o *options) {
		o.task = t
	}
}

// WithTracker injects a pre-configured Tracker feature into GetFiber and Opus.
func WithTracker(t *tracker.Tracker) Option {
	return func(o *options) {
		o.tracker = t
	}
}

// WithWorker injects an in-process Worker and its transport into Opus.
func WithWorker(w *worker.Worker, inProc *worker.InProcessTransport) Option {
	return func(o *options) {
		o.worker = w
		o.inProc = inProc
	}
}

type Opus struct {
	cfg     Config
	fiber   *fiber.App
	logger  *slog.Logger
	db      *ent.Client
	bus     *eventbus.Bus
	node    *node.Node
	tracker *tracker.Tracker
	worker  *worker.Worker
	inProc  *worker.InProcessTransport
}

func New(cfg Config, fiberApp *fiber.App, logger *slog.Logger, db *ent.Client, bus *eventbus.Bus, opts ...Option) *Opus {
	var o options
	for _, opt := range opts {
		if opt != nil {
			opt(&o)
		}
	}
	return &Opus{
		cfg:     cfg,
		fiber:   fiberApp,
		logger:  logger,
		db:      db,
		bus:     bus,
		node:    o.node,
		tracker: o.tracker,
		worker:  o.worker,
		inProc:  o.inProc,
	}
}

func (o *Opus) Serve(ctx context.Context) error {
	if o.node != nil {
		if err := o.node.Start(ctx); err != nil {
			o.logger.ErrorContext(ctx, "failed to start node detector", "error", err)
		}
	}
	if o.tracker != nil {
		if err := o.tracker.Start(ctx); err != nil {
			o.logger.ErrorContext(ctx, "failed to start tracker poller", "error", err)
		}
	}

	stopWorker := o.startMasterWorker(ctx)

	addr := net.JoinHostPort(o.cfg.Host, o.cfg.Port)
	o.logger.InfoContext(ctx, "starting server", "addr", addr)
	err := o.fiber.Listen(addr, fiber.ListenConfig{
		DisableStartupMessage: true,
		GracefulContext:       ctx,
		ShutdownTimeout:       o.cfg.ShutdownTimeout,
	})

	stopWorker()
	if o.node != nil {
		o.node.Stop()
	}
	if o.tracker != nil {
		o.tracker.Stop()
	}

	if o.bus != nil {
		timeout := o.cfg.ShutdownTimeout
		if timeout <= 0 {
			timeout = 10 * time.Second
		}
		closeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), timeout)
		if closeErr := o.bus.Close(closeCtx); closeErr != nil {
			o.logger.ErrorContext(ctx, "failed to close event bus", "error", closeErr)
		}
		cancel()
	}

	if o.db != nil {
		if closeErr := o.db.Close(); closeErr != nil {
			o.logger.ErrorContext(ctx, "failed to close database client", "error", closeErr)
		}
	}
	return err
}

func (o *Opus) startMasterWorker(ctx context.Context) func() {
	if o.worker == nil {
		return func() {}
	}

	workerCtx, cancel := context.WithCancel(ctx)
	var wg sync.WaitGroup
	var unregister func()

	if o.inProc != nil && o.node != nil {
		masterID, err := o.node.MasterID(ctx)
		if err != nil {
			o.logger.ErrorContext(ctx, "failed to resolve master node id", "error", err)
		} else if masterID != "" {
			unregister = o.node.Hub().RegisterLocal(masterID, &inProcessConn{transport: o.inProc})
			wg.Add(1)
			go func() {
				defer wg.Done()
				for {
					select {
					case <-workerCtx.Done():
						return
					case env, ok := <-o.inProc.Events:
						if !ok {
							return
						}
						if err := o.node.Hub().Dispatch(masterID, env); err != nil {
							o.logger.WarnContext(workerCtx, "failed to dispatch master worker envelope", "type", env.Type, "error", err)
						}
					}
				}
			}()
		}
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := o.worker.Run(workerCtx); err != nil && !errors.Is(err, context.Canceled) {
			o.logger.ErrorContext(workerCtx, "master worker stopped with error", "error", err)
		}
	}()

	return func() {
		cancel()
		if unregister != nil {
			unregister()
		}
		if o.inProc != nil {
			_ = o.inProc.Close()
		}
		wg.Wait()
	}
}

type inProcessConn struct {
	transport *worker.InProcessTransport
}

func (c *inProcessConn) WriteJSON(v any) error {
	env, ok := v.(node.Envelope)
	if !ok {
		raw, err := json.Marshal(v)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(raw, &env); err != nil {
			return err
		}
	}
	return c.transport.SendCommand(context.Background(), env)
}

func (c *inProcessConn) Close() error {
	return c.transport.Close()
}
