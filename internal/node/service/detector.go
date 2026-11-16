package service

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// ponytail: fixed 45s stale threshold (3 missed 15s heartbeats); make configurable via worker policy when heterogeneous node pools exist
const DefaultDetectorTimeout = 45 * time.Second

// ponytail: check interval fixed at 15s to match heartbeat cadence; upgrade to adaptive ticker when node count exceeds 10k
const DefaultDetectorInterval = 15 * time.Second

// DetectorConfig holds configuration for the stale node detector.
type DetectorConfig struct {
	Now      func() time.Time
	Interval time.Duration
	Timeout  time.Duration
	Logger   *slog.Logger
	OnStop   func()
}

// Detector periodically detects silent worker nodes and marks them offline.
type Detector struct {
	svc      *Node
	now      func() time.Time
	interval time.Duration
	timeout  time.Duration
	logger   *slog.Logger
	onStop   func()

	mu      sync.Mutex
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	running bool
}

// NewDetector creates a new Detector.
func NewDetector(svc *Node, cfg DetectorConfig) *Detector {
	now := cfg.Now
	if now == nil {
		now = time.Now
	}
	interval := cfg.Interval
	if interval <= 0 {
		interval = DefaultDetectorInterval
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = DefaultDetectorTimeout
	}
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	return &Detector{
		svc:      svc,
		now:      now,
		interval: interval,
		timeout:  timeout,
		logger:   logger,
		onStop:   cfg.OnStop,
	}
}

// Start launches the background detection loop.
func (d *Detector) Start(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.running {
		return nil
	}

	runCtx, cancel := context.WithCancel(ctx)
	d.cancel = cancel
	d.running = true
	d.wg.Add(1)

	go func() {
		defer d.wg.Done()
		ticker := time.NewTicker(d.interval)
		defer ticker.Stop()

		for {
			select {
			case <-runCtx.Done():
				return
			case <-ticker.C:
				if err := d.CheckStale(runCtx); err != nil && runCtx.Err() == nil {
					d.logger.Error("detector check stale failed", "error", err)
				}
			}
		}
	}()

	return nil
}

// Stop stops the background detection loop and waits for the goroutine to finish.
func (d *Detector) Stop() {
	d.mu.Lock()
	if !d.running {
		d.mu.Unlock()
		return
	}
	cancel := d.cancel
	d.running = false
	d.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	d.wg.Wait()
	if d.onStop != nil {
		d.onStop()
	}
}

// Running returns true if the background detection loop is running.
func (d *Detector) Running() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.running
}

// CheckStale runs a single stale detection check.
func (d *Detector) CheckStale(ctx context.Context) error {
	cutoff := d.now().Add(-d.timeout)
	return d.svc.DetectStale(ctx, cutoff)
}
