# Queue & Worker System Architecture

**Product:** Opus  
**Version:** 1.0.1  
**Status:** Draft  
**Last Updated:** 2026-05-16  
**Authors:** Product & Architecture Team

---

## Table of Contents

1. [Overview](#1-overview)
2. [Requirements](#2-requirements)
3. [Architecture](#3-architecture)
4. [Data Models](#4-data-models)
5. [Core Interfaces](#5-core-interfaces)
6. [Driver Implementations](#6-driver-implementations)
7. [Worker Engine](#7-worker-engine)
8. [Scheduler](#8-scheduler)
9. [Dead Letter Queue](#9-dead-letter-queue)
10. [Configuration](#10-configuration)
11. [Testing Strategy](#11-testing-strategy)
12. [Task Automation](#12-task-automation)

---

## 1. Overview

Opus requires a robust, driver-agnostic queue and worker system to handle all background tasks — agent processing, notifications, reminders, and scheduled jobs (cron). The system is embedded directly into the single Opus binary and runs as goroutines alongside the HTTP server and SSE engine. No external service manager is required.

```
opus (single binary)
├── HTTP Server     (GoFiber v3)
├── SSE Engine      (agent streaming)
├── Worker Engine   (job consumer)       ← this document
└── Scheduler       (cron ticker)        ← this document
```

---

## 2. Requirements

| ID | Requirement |
|----|-------------|
| QW-01 | Driver-based architecture: swap Redis ↔ PostgreSQL ↔ SQLite without changing business logic. |
| QW-02 | Priority queue: numerical priority 0–10 (higher = more urgent). |
| QW-03 | Persistence: jobs survive process restart. |
| QW-04 | Cron/scheduled jobs stored in the database; dynamically manageable at runtime. |
| QW-05 | Configurable worker concurrency via `OPUS_QUEUE_WORKERS`. |
| QW-06 | Dead letter queue: jobs exceeding `MaxRetries` are moved to a `dead_letter` table for inspection and manual retry. |
| QW-07 | Graceful shutdown: in-flight jobs complete before process exits. |
| QW-08 | Exponential backoff for job retries. |
| QW-09 | Structured logging via `slog`. |
| QW-10 | All interfaces compatible with `go.uber.org/mock` for unit testing. |
| QW-11 | Embedded in the Opus binary — no separate service required. |

---

## 3. Architecture

### 3.1 Component Overview

```
┌─────────────────────────────────────────────────────┐
│                   Opus Process                       │
│                                                      │
│  ┌────────────┐    ┌──────────────┐                 │
│  │  Handler   │    │  Scheduler   │                 │
│  │ (HTTP/SSE) │    │ (cron ticker)│                 │
│  └─────┬──────┘    └──────┬───────┘                 │
│        │ Enqueue()        │ Enqueue()               │
│        ▼                  ▼                          │
│  ┌─────────────────────────────┐                    │
│  │       Queue Service         │                    │
│  └──────────────┬──────────────┘                    │
│                 │                                    │
│                 ▼                                    │
│  ┌──────────────────────────┐                       │
│  │       Queue Driver        │                      │
│  │  (Redis / PG / SQLite)    │                      │
│  └──────────────┬────────────┘                      │
│                 │ Pop()                              │
│                 ▼                                    │
│  ┌──────────────────────────┐                       │
│  │      Worker Engine        │                      │
│  │  (N concurrent goroutines)│                      │
│  └──────────────┬────────────┘                      │
│                 │ Dispatch                           │
│                 ▼                                    │
│  ┌──────────────────────────┐                       │
│  │    Handler Registry       │                      │
│  │  map[string]HandlerFunc   │                      │
│  └──────────────────────────┘                       │
└─────────────────────────────────────────────────────┘
```

### 3.2 Folder Structure

```
api/internal/
├── model/
│   ├── job.go              # Job, JobStatus, DeadLetter models
│   └── cron.go             # CronSchedule model
├── service/
│   ├── queue.go            # Queue & Scheduler interfaces
│   └── worker.go           # Worker, HandlerFunc definitions
├── repository/
│   └── queue/
│       ├── driver.go       # QueueDriver interface
│       ├── entgo.go        # EntGo driver (PostgreSQL + SQLite)
│       └── redis.go        # Redis driver
├── worker/
│   ├── engine.go           # Worker engine (goroutine pool)
│   ├── scheduler.go        # Cron scheduler (ticker-based)
│   └── handlers/
│       └── registry.go     # Handler registry
└── config/
    └── queue.go            # Viper integration for OPUS_QUEUE_*
```

### 3.3 Clean Architecture Compliance

Dependency direction follows the existing Opus convention — strictly inward:

```
handler → service → repository/queue → model
worker/engine → service → repository/queue → model
```

- `worker/engine` and `worker/scheduler` depend on `service` interfaces only.
- `repository/queue` implements the `QueueDriver` interface defined in `service/`.
- `model/` has no external dependencies.

---

## 4. Data Models

### 4.1 Job (`model/job.go`)

```go
package model

import "time"

// JobStatus represents the current lifecycle state of a job.
type JobStatus string

const (
    StatusPending   JobStatus = "pending"
    StatusRunning   JobStatus = "running"
    StatusCompleted JobStatus = "completed"
    StatusFailed    JobStatus = "failed"
)

// Job represents a unit of background work to be executed by the worker engine.
type Job struct {
    ID          string    `json:"id"`
    Type        string    `json:"type"`            // matches registered HandlerFunc key
    Payload     []byte    `json:"payload"`         // JSON-encoded handler input
    Priority    int       `json:"priority"`        // 0-10, higher = more urgent
    Status      JobStatus `json:"status"`
    Retries     int       `json:"retries"`
    MaxRetries  int       `json:"max_retries"`
    ScheduledAt time.Time `json:"scheduled_at"`    // zero = immediate
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
    Error       string    `json:"error,omitempty"` // last error message
}

// DeadLetter represents a job that has exceeded its MaxRetries threshold.
type DeadLetter struct {
    ID        string    `json:"id"`
    JobID     string    `json:"job_id"`
    Type      string    `json:"type"`
    Payload   []byte    `json:"payload"`
    LastError string    `json:"last_error"`
    Retries   int       `json:"retries"`
    CreatedAt time.Time `json:"created_at"`
}
```

### 4.2 CronSchedule (`model/cron.go`)

```go
package model

import "time"

// CronSchedule defines a recurring job trigger stored in the database.
type CronSchedule struct {
    ID        string    `json:"id"`
    Name      string    `json:"name"`
    CronExpr  string    `json:"cron_expression"` // standard 5-field cron expression
    JobType   string    `json:"job_type"`         // matches registered HandlerFunc key
    Payload   []byte    `json:"payload"`
    IsActive  bool      `json:"is_active"`
    LastRunAt time.Time `json:"last_run_at"`
    NextRunAt time.Time `json:"next_run_at"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}
```

---

## 5. Core Interfaces

### 5.1 Service Layer (`service/queue.go`)

```go
package service

import (
    "context"
    "github.com/kilip/opus/api/internal/model"
)

// Queue is the entry point for enqueuing background jobs.
type Queue interface {
    // Enqueue submits a job to the queue for asynchronous execution.
    Enqueue(ctx context.Context, job *model.Job) error
}

// Scheduler manages cron-based recurring job schedules.
type Scheduler interface {
    // AddCron registers or updates a cron schedule.
    AddCron(ctx context.Context, schedule *model.CronSchedule) error
    // RemoveCron deactivates a cron schedule by ID.
    RemoveCron(ctx context.Context, id string) error
    // Start begins the scheduler background ticker.
    Start(ctx context.Context) error
    // Stop gracefully halts the scheduler.
    Stop() error
}
```

### 5.2 Service Layer (`service/worker.go`)

```go
package service

import (
    "context"
    "github.com/kilip/opus/api/internal/model"
)

// HandlerFunc is the function signature all job handlers must implement.
type HandlerFunc func(ctx context.Context, job *model.Job) error

// Worker defines the worker engine lifecycle.
type Worker interface {
    // Register associates a job type string with a handler function.
    Register(jobType string, handler HandlerFunc)
    // Start launches N worker goroutines and begins consuming jobs.
    Start(ctx context.Context) error
    // Stop signals all workers to finish in-flight jobs and shut down.
    Stop() error
}
```

### 5.3 Repository Layer (`repository/queue/driver.go`)

```go
package queue

import (
    "context"
    "time"
    "github.com/kilip/opus/api/internal/model"
)

// QueueDriver is the persistence abstraction for the queue system.
// Implementations: entgo.go (PostgreSQL/SQLite), redis.go (Redis).
type QueueDriver interface {
    // Push persists a job to the queue backend.
    Push(ctx context.Context, job *model.Job) error
    // Pop atomically retrieves and locks the highest-priority pending job.
    // Returns nil, nil if no job is available.
    Pop(ctx context.Context) (*model.Job, error)
    // UpdateStatus updates the status and error message of a job.
    UpdateStatus(ctx context.Context, id string, status model.JobStatus, errMsg string) error
    // MoveToDead moves a failed job to the dead letter store.
    MoveToDead(ctx context.Context, job *model.Job) error

    // UpsertCron creates or updates a cron schedule.
    UpsertCron(ctx context.Context, cron *model.CronSchedule) error
    // DeleteCron removes a cron schedule by ID.
    DeleteCron(ctx context.Context, id string) error
    // ListPendingCrons returns all active cron schedules due for execution.
    ListPendingCrons(ctx context.Context) ([]*model.CronSchedule, error)
    // UpdateCronNextRun updates LastRunAt and NextRunAt after a cron fires.
    UpdateCronNextRun(ctx context.Context, id string, lastRun, nextRun time.Time) error
}
```

---

## 6. Driver Implementations

### 6.1 Driver Selection (`config/queue.go`)

The driver is selected once at startup and injected via the existing `internal/config` singleton pattern:

```go
// internal/config/queue.go

// GetQueueDriver returns the configured QueueDriver singleton.
func GetQueueDriver() queue.QueueDriver {
    cfg := GetConfig()
    switch cfg.Queue.Driver {
    case "redis":
        return queue.NewRedisDriver(cfg.Queue.Redis)
    case "postgres", "sqlite":
        return queue.NewEntGoDriver(GetDatabase(), cfg.Database.Driver)
    default:
        panic("unsupported queue driver: " + cfg.Queue.Driver)
    }
}
```

### 6.2 EntGo Driver (`repository/queue/entgo.go`)

Used for both PostgreSQL and SQLite backends via the shared EntGo client.

**PostgreSQL** uses `SELECT FOR UPDATE SKIP LOCKED` for concurrent-safe dequeuing.

**SQLite** uses an in-process `sync.Mutex`. Since Opus is a multi-user, single-process deployment, a mutex is sufficient to guarantee that only one goroutine dequeues at a time — no workaround complexity required.

```go
type entGoDriver struct {
    client *ent.Client
    mu     sync.Mutex // guards Pop() for SQLite dialect only
    driver string     // "sqlite" | "postgres"
}

// Pop atomically dequeues the highest-priority pending job.
func (d *entGoDriver) Pop(ctx context.Context) (*model.Job, error) {
    if d.driver == "sqlite" {
        d.mu.Lock()
        defer d.mu.Unlock()
    }
    // query: highest priority + scheduled_at <= now + status = pending
    // postgres: append FOR UPDATE SKIP LOCKED
}
```

### 6.3 Redis Driver (`repository/queue/redis.go`)

Uses Redis sorted sets for priority-aware queuing.

| Operation | Redis Command | Notes |
|-----------|--------------|-------|
| `Push` | `ZADD queue:pending <score> <job_id>` | Score = `(10 - priority) * 1e12 + unixNano` |
| `Pop` | `ZPOPMIN queue:pending 1` + `SET job:<id>` | Lowest score = highest priority |
| `UpdateStatus` | `SET job:<id> <json>` | JSON blob per job |
| `MoveToDead` | `RPUSH queue:dead <job_json>` | Simple list for dead letters |
| `UpsertCron` | `HSET crons <id> <json>` | Hash map keyed by cron ID |
| `ListPendingCrons` | `HGETALL crons` | Filter `is_active` + `next_run_at` in Go |

---

## 7. Worker Engine

### 7.1 Implementation Skeleton (`worker/engine.go`)

```go
// Engine is the concrete implementation of service.Worker.
type Engine struct {
    driver   queue.QueueDriver
    handlers map[string]service.HandlerFunc
    workers  int
    log      *slog.Logger
    wg       sync.WaitGroup
    quit     chan struct{}
}

// NewEngine constructs a worker engine with the given driver and concurrency.
func NewEngine(driver queue.QueueDriver, workers int, log *slog.Logger) *Engine

// Register associates a job type with its handler.
func (e *Engine) Register(jobType string, handler service.HandlerFunc)

// Start launches the worker pool.
func (e *Engine) Start(ctx context.Context) error

// Stop signals workers to finish in-flight jobs and waits for completion.
func (e *Engine) Stop() error
```

### 7.2 Worker Loop

```
for {
    select {
    case <-quit:
        return
    default:
        job = driver.Pop()
        if job == nil:
            sleep(pollInterval)     // back-off when queue is empty
            continue

        handler = registry[job.Type]
        if handler == nil:
            log.Error("no handler registered", "type", job.Type)
            driver.UpdateStatus(job.ID, StatusFailed, "no handler registered")
            continue

        driver.UpdateStatus(job.ID, StatusRunning, "")
        err = handler(ctx, job)

        if err == nil:
            driver.UpdateStatus(job.ID, StatusCompleted, "")
        else:
            job.Retries++
            if job.Retries >= job.MaxRetries:
                driver.MoveToDead(job)
            else:
                job.ScheduledAt = now + backoff(job.Retries)
                driver.Push(job)    // re-enqueue with delay
    }
}
```

### 7.3 Retry Backoff

Exponential backoff with ±10% jitter, capped at 1 hour:

```go
// backoff returns the retry delay for the given attempt number.
// attempt 1 → 30s, attempt 2 → 60s, attempt 3 → 120s, ...
func backoff(attempt int) time.Duration {
    base := 30 * time.Second
    max  := 1 * time.Hour
    d    := base * time.Duration(math.Pow(2, float64(attempt-1)))
    if d > max {
        d = max
    }
    jitter := time.Duration(rand.Int63n(int64(d / 10)))
    return d + jitter
}
```

### 7.4 Graceful Shutdown

The engine integrates with the Opus process shutdown sequence in `cmd/opus/start.go`:

```go
sigCh := make(chan os.Signal, 1)
signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
<-sigCh

engine.Stop()    // blocks until in-flight jobs complete
server.Shutdown()
```

---

## 8. Scheduler

### 8.1 Design

The scheduler is a background goroutine that ticks every minute, loads all active `CronSchedule` records where `next_run_at <= now`, enqueues the corresponding jobs, and updates `last_run_at` / `next_run_at`.

Cron expression parsing uses `robfig/cron` v3 (standard 5-field format).

### 8.2 Implementation Skeleton (`worker/scheduler.go`)

```go
// cronScheduler is the concrete implementation of service.Scheduler.
type cronScheduler struct {
    driver queue.QueueDriver
    queue  service.Queue
    log    *slog.Logger
    ticker *time.Ticker
    quit   chan struct{}
}

func (s *cronScheduler) Start(ctx context.Context) error
func (s *cronScheduler) Stop() error
```

### 8.3 Scheduler Loop

```
every 1 minute:
    crons = driver.ListPendingCrons()    // is_active=true AND next_run_at <= now
    for each cron:
        job = &model.Job{
            Type:     cron.JobType,
            Payload:  cron.Payload,
            Priority: 5,               // default mid-priority for scheduled jobs
        }
        queue.Enqueue(ctx, job)
        next = parseExpr(cron.CronExpr).Next(now)
        driver.UpdateCronNextRun(cron.ID, now, next)
```

### 8.4 Cron Expression Format

Standard 5-field cron syntax:

```
┌─────────── minute       (0–59)
│ ┌───────── hour         (0–23)
│ │ ┌─────── day of month (1–31)
│ │ │ ┌───── month        (1–12)
│ │ │ │ ┌─── day of week  (0–6, Sunday=0)
│ │ │ │ │
* * * * *
```

| Expression | Meaning |
|------------|---------|
| `0 8 * * *` | Every day at 08:00 |
| `0 9 * * 1` | Every Monday at 09:00 |
| `*/30 * * * *` | Every 30 minutes |
| `0 0 1 * *` | First day of every month at midnight |

---

## 9. Dead Letter Queue

### 9.1 Behaviour

A job is moved to the dead letter store when `retries >= max_retries`. The entry preserves the full payload and last error for inspection and manual retry from the dashboard.

### 9.2 API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/queue/dead` | List all dead letter entries |
| `POST` | `/queue/dead/:id/retry` | Re-enqueue with `retries = 0` |
| `DELETE` | `/queue/dead/:id` | Permanently remove entry |

### 9.3 Storage

- **PostgreSQL / SQLite**: dedicated `dead_letters` table via EntGo schema (`ent/schema/deadletter.go`).
- **Redis**: `queue:dead` list as JSON blobs; retry/delete implemented via list operations.

---

## 10. Configuration

### 10.1 TOML Structure

```toml
[queue]
driver  = "sqlite"   # "sqlite" | "postgres" | "redis"
workers = 5          # concurrent worker goroutines

[queue.redis]
host     = "localhost"
port     = 6379
password = ""
db       = 0
```

### 10.2 Environment Variable Mapping

| TOML Key | Environment Variable | Default |
|----------|---------------------|---------|
| `queue.driver` | `OPUS_QUEUE_DRIVER` | `sqlite` |
| `queue.workers` | `OPUS_QUEUE_WORKERS` | `5` |
| `queue.redis.host` | `OPUS_REDIS_HOST` | `localhost` |
| `queue.redis.port` | `OPUS_REDIS_PORT` | `6379` |
| `queue.redis.password` | `OPUS_REDIS_PASSWORD` | `` |
| `queue.redis.db` | `OPUS_REDIS_DB` | `0` |

### 10.3 `.env.example` Additions

```dotenv
# Queue
OPUS_QUEUE_DRIVER=sqlite        # sqlite | postgres | redis
OPUS_QUEUE_WORKERS=5

# Redis (only required when OPUS_QUEUE_DRIVER=redis)
OPUS_REDIS_HOST=localhost
OPUS_REDIS_PORT=6379
OPUS_REDIS_PASSWORD=
OPUS_REDIS_DB=0
```

---

## 11. Testing Strategy

### 11.1 Unit Tests (Service Layer)

- Location: `internal/service/queue_test.go`, `internal/worker/engine_test.go`
- Mocks: `QueueDriver` and `Worker` interfaces via `go.uber.org/mock/mockgen`
- Scope: `Enqueue`, retry logic, dead letter promotion, backoff calculation

```go
// Example: internal/service/queue_test.go
mockDriver := mocks.NewMockQueueDriver(ctrl)
mockDriver.EXPECT().Push(ctx, gomock.Any()).Return(nil)

svc := service.NewQueueService(mockDriver)
err := svc.Enqueue(ctx, &model.Job{Type: "ai_task", Priority: 8})
assert.NoError(t, err)
```

### 11.2 Integration Tests (Repository Layer)

- Location: `internal/repository/queue/entgo_integration_test.go`
- Build tag: `//go:build integration`
- Database: SQLite in-memory (`file::memory:?cache=shared&_fk=1`)
- Scope: full `Push → Pop → UpdateStatus → MoveToDead` cycle

```go
//go:build integration

client, _ := ent.Open("sqlite3", "file::memory:?cache=shared&_fk=1")
client.Schema.Create(ctx)
driver := queue.NewEntGoDriver(client, "sqlite")

err := driver.Push(ctx, &model.Job{Type: "test", Priority: 5})
assert.NoError(t, err)

job, err := driver.Pop(ctx)
assert.NoError(t, err)
assert.Equal(t, "test", job.Type)
```

### 11.3 Running Tests

```bash
task test               # unit tests
task test:integration   # integration tests
task test:all           # all tests
```

---

## 12. Task Automation

The following tasks are added to `api/Taskfile.yml`:

| Task | Description |
|------|-------------|
| `task queue:mock` | Regenerate `QueueDriver` and `Worker` mocks via `mockgen` |
| `task ent:generate` | Regenerate EntGo code — also covers `Job`, `CronSchedule`, `DeadLetter` schemas |

---

## Appendix: Handler Registration Example

All handlers are registered explicitly at startup in `cmd/opus/start.go`. No `init()` magic.

```go
engine := worker.NewEngine(
    config.GetQueueDriver(),
    config.GetConfig().Queue.Workers,
    config.GetLogger(),
)

engine.Register("ai_task",     handlers.AITask)
engine.Register("send_email",  handlers.SendEmail)
engine.Register("reminder",    handlers.Reminder)
engine.Register("daily_brief", handlers.DailyBrief)

if err := engine.Start(ctx); err != nil {
    log.Fatal("worker engine failed to start", "error", err)
}
```

Each handler follows the `service.HandlerFunc` signature:

```go
// worker/handlers/ai_task.go

// AITask processes an AI agent job from the queue.
func AITask(ctx context.Context, job *model.Job) error {
    var input AITaskInput
    if err := json.Unmarshal(job.Payload, &input); err != nil {
        return fmt.Errorf("invalid payload: %w", err)
    }
    // ... process
    return nil
}
```