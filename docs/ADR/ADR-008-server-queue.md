# ADR-008: Server Queue Architecture

**Status:** Accepted
**Date:** 2026-05-17
**Deciders:** Chief Architect
**Context:** Opus Server (`opus/server/`)

---

## 1. Context

Opus Server requires a durable, reliable messaging infrastructure to support three categories of asynchronous work:

1. **Background Job Processing** — Sending notifications, dispatching emails, post-processing agent outputs, and other deferred tasks that must not block the HTTP request lifecycle.
2. **Agent Task Execution** — Queuing autonomous agent evaluation cycles with support for priority, delay, retry, and failure handling. Agent tasks are long-running and must survive server restarts.
3. **Internal Domain Event Bus** — Decoupling feature domains from one another via a publish/subscribe mechanism. For example: an agent completing its run may trigger a workflow execution or a vault write without the `agent` package importing `workflow` or `vault` directly.

Without a unified, interface-driven queue abstraction, each domain risks implementing bespoke async patterns, creating operational inconsistency, coupling domains together implicitly, and making it impossible to swap queue backends without rewriting call sites.

This ADR establishes the canonical queue and event bus architecture for Opus Server, including interface contracts, backend configuration, implementation specifications for each supported backend, and dependency injection conventions.

---

## 2. Decision

Opus Server adopts a **dual-abstraction queue architecture** consisting of two independent interfaces:

- **`Queue`** — A durable, producer/consumer job queue for background tasks and agent execution. Supports priority, delay, retry, and dead-letter handling.
- **`EventBus`** — An in-process publish/subscribe event bus for internal domain decoupling. Subscribers are registered at startup; events are dispatched synchronously within the same process.

Both abstractions are defined in `internal/shared/queue/`. All backing engines are implementation details confined to `internal/adapter/queue/`. No application code outside `internal/adapter/queue/` references a concrete queue implementation.

The backing engine for `Queue` is **configurable** per deployment: Database (via Ent, default) or Redis (high-throughput). The `EventBus` is always in-process; it does not require a persistent backend.

---

### 2.1 Directory Structure

```text
opus/
└── server/
    ├── internal/
    │   └── shared/
    │       └── queue/
    │           ├── queue.go        # Queue interface + Job, JobOption, JobStatus types
    │           ├── eventbus.go     # EventBus interface + Event, Handler types
    │           ├── config.go       # queue.Config struct (hybrid composition — ADR-002)
    │           └── noop.go         # NoopQueue + NoopEventBus (testing utilities)
    └── internal/
        └── adapter/
            └── queue/
                ├── database/
                │   └── queue.go        # Ent-backed Queue implementation
                ├── redis/
                │   └── queue.go        # Redis-backed Queue implementation (Asynq)
                └── factory.go        # Backend construction logic
            └── memory/
                └── eventbus.go     # In-process EventBus implementation
```

---

### 2.2 Queue Interface

The `Queue` interface is the single contract for all background job and agent task processing. No concrete backend is referenced outside `internal/adapter/queue/`.

```go
// internal/shared/queue/queue.go
package queue

import (
    "context"
    "time"
)

// JobStatus represents the lifecycle state of a queued job.
type JobStatus string

const (
    JobStatusPending    JobStatus = "pending"
    JobStatusRunning    JobStatus = "running"
    JobStatusCompleted  JobStatus = "completed"
    JobStatusFailed     JobStatus = "failed"
    JobStatusDead       JobStatus = "dead" // exceeded max retries
)

// Job represents a unit of work to be processed by a queue consumer.
type Job struct {
    // ID is a unique identifier assigned by the queue backend at enqueue time.
    ID string

    // Type is a string identifier used to route the job to the correct handler.
    // Convention: "<domain>:<action>" (e.g. "agent:evaluate", "email:send").
    Type string

    // Payload is the serialised job arguments. JSON encoding is recommended.
    Payload []byte

    // Queue is the name of the queue this job belongs to.
    // Convention: use feature domain names (e.g. "agent", "email", "default").
    Queue string

    // Priority controls execution order within the same queue.
    // Higher values are processed first. Default: 0.
    Priority int

    // MaxRetries is the maximum number of retry attempts before the job is
    // moved to the dead-letter queue. Default: 3.
    MaxRetries int

    // RetryCount is the number of times this job has been retried.
    RetryCount int

    // ProcessAt is the earliest time at which this job may be processed.
    // Jobs with ProcessAt in the future are deferred.
    ProcessAt time.Time

    // CreatedAt is the time the job was enqueued.
    CreatedAt time.Time

    // Status is the current lifecycle state of the job.
    Status JobStatus
}

// JobOption is a functional option for configuring a job at enqueue time.
type JobOption func(*Job)

// WithPriority sets the job priority. Higher values are processed first.
func WithPriority(p int) JobOption {
    return func(j *Job) { j.Priority = p }
}

// WithDelay defers the job by the given duration from now.
func WithDelay(d time.Duration) JobOption {
    return func(j *Job) { j.ProcessAt = time.Now().Add(d) }
}

// WithProcessAt sets an absolute time at which the job may first be processed.
func WithProcessAt(t time.Time) JobOption {
    return func(j *Job) { j.ProcessAt = t }
}

// WithMaxRetries sets the maximum retry count before dead-lettering.
func WithMaxRetries(n int) JobOption {
    return func(j *Job) { j.MaxRetries = n }
}

// WithQueue assigns the job to a named queue.
func WithQueue(name string) JobOption {
    return func(j *Job) { j.Queue = name }
}

// Handler is a function that processes a single job.
// Returning a non-nil error causes the job to be retried (if retries remain)
// or moved to the dead-letter queue.
type Handler func(ctx context.Context, job *Job) error

// Queue defines the contract for all durable background job processing.
// Implementations must be safe for concurrent use.
type Queue interface {
    // Enqueue adds a job to the queue. Returns the assigned job ID.
    Enqueue(ctx context.Context, jobType string, payload []byte, opts ...JobOption) (string, error)

    // RegisterHandler registers a Handler function for the given job type.
    // Must be called before Start(). Panics if called after Start().
    RegisterHandler(jobType string, handler Handler)

    // Start begins consuming jobs from the queue.
    // It is non-blocking; workers run in background goroutines.
    // ctx cancellation triggers a graceful shutdown.
    Start(ctx context.Context) error

    // Shutdown stops all workers gracefully, waiting for in-flight jobs
    // to complete or until the context deadline is exceeded.
    Shutdown(ctx context.Context) error

    // Inspect returns the current status of a job by ID.
    Inspect(ctx context.Context, jobID string) (*Job, error)
}
```

---

### 2.3 EventBus Interface

The `EventBus` interface provides publish/subscribe semantics for internal domain decoupling. It is always in-process; no network or database round-trip is involved.

```go
// internal/shared/queue/eventbus.go
package queue

import "context"

// Event is an immutable domain event emitted by a feature domain.
type Event struct {
    // Topic is a dot-separated string identifying the event category.
    // Convention: "<domain>.<action>" (e.g. "agent.completed", "vault.written").
    Topic string

    // Payload is the serialised event data. JSON encoding is recommended.
    Payload []byte

    // Source is the domain package that emitted the event (for logging/tracing).
    Source string
}

// EventHandler is a function invoked for each matching event.
// Returning a non-nil error is logged but does not affect other subscribers.
type EventHandler func(ctx context.Context, event Event) error

// EventBus defines the contract for in-process publish/subscribe messaging.
// All subscribers are notified synchronously in the order they were registered.
// Implementations must be safe for concurrent use.
type EventBus interface {
    // Publish emits an event to all registered subscribers for the given topic.
    // Topic matching supports wildcards: "agent.*" matches "agent.completed",
    // "agent.failed", etc.
    Publish(ctx context.Context, event Event) error

    // Subscribe registers a handler for events matching the given topic pattern.
    // Must be called before the application begins processing requests.
    Subscribe(topic string, handler EventHandler)
}
```

---

### 2.4 Configuration — Hybrid Composition (ADR-002)

Following the **Hybrid Config Composition Pattern** from ADR-002, the queue package owns its configuration struct. It is composed into the root `Config` in `internal/config/model.go`.

```go
// internal/shared/queue/config.go
package queue

// Driver identifies the backing engine for the job queue.
type Driver string

const (
    DriverDatabase Driver = "database"
    DriverRedis    Driver = "redis"
)

// Config holds all queue configuration.
// It is owned by the queue package and composed into the root config.Config
// by internal/config/model.go.
//
// Environment variable overrides follow the OPUS_ prefix convention:
//   OPUS_QUEUE_DRIVER       — sets Driver
//   OPUS_QUEUE_DSN          — sets DSN
//   OPUS_QUEUE_CONCURRENCY  — sets Concurrency
type Config struct {
    // Driver selects the queue backend.
    // Valid values: "database" (default), "redis".
    Driver Driver `mapstructure:"driver" json:"driver" jsonschema:"enum=database,enum=redis,default=database,description=Queue backend driver"`

    // DSN is the data source name for the selected driver.
    // database: unused (the global Ent client is used)
    // redis:    Redis URL (e.g. "redis://localhost:6379")
    DSN string `mapstructure:"dsn" json:"dsn" jsonschema:"description=Data source name for the queue backend. Use env var OPUS_QUEUE_DSN for secrets."`

    // Concurrency is the number of concurrent job workers.
    // Default: 10.
    Concurrency int `mapstructure:"concurrency" json:"concurrency" jsonschema:"default=10,description=Number of concurrent job processing workers"`

    // RetentionHours is the number of hours to retain completed/failed jobs
    // in the database before pruning. Default: 168 (7 days).
    RetentionHours int `mapstructure:"retention_hours" json:"retention_hours" jsonschema:"default=168,description=Hours to retain completed and failed jobs before pruning"`
}
```

Root config composition in `internal/config/model.go`:

```go
// internal/config/model.go (excerpt)
import "opus/server/internal/shared/queue"

type Config struct {
    // ... other fields from previous ADRs
    Queue queue.Config `mapstructure:"queue" json:"queue"`
}
```

---

### 2.5 Backend Implementation Specifications

#### 2.5.1 Database Backend (Ent)

The Database backend relies on the project's global Ent ORM client to manage jobs in a database table. This abstracts away the underlying SQL dialects and makes it compatible with any database supported by Ent (e.g. SQLite, PostgreSQL).

**Schema:**

```go
// server/ent/schema/job.go
package schema

import (
    "entgo.io/ent"
    "entgo.io/ent/schema/field"
    "entgo.io/ent/schema/index"
    "time"
)

type Job struct {
    ent.Schema
}

func (Job) Fields() []ent.Field {
    return []ent.Field{
        field.String("id").Unique().Immutable(),
        field.String("type").NotEmpty(),
        field.String("queue").Default("default"),
        field.Bytes("payload"),
        field.String("status").Default("pending"),
        field.Int("priority").Default(0),
        field.Int("max_retries").Default(3),
        field.Int("retry_count").Default(0),
        field.Time("process_at").Default(time.Now),
        field.Time("created_at").Default(time.Now).Immutable(),
        field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
    }
}

func (Job) Indexes() []ent.Index {
    return []ent.Index{
        index.Fields("queue", "status", "priority", "process_at"),
    }
}
```

**Polling loop:**

The database backend uses a polling loop with a configurable interval (default: 500ms). It fetches pending jobs via Ent.

```go
// internal/adapter/queue/database/queue.go
package database

import (
    "context"
    "time"

    "github.com/google/uuid"
    "opus/server/ent"
    "opus/server/ent/job"
    "opus/server/internal/shared/queue"
)

const pollInterval = 500 * time.Millisecond

// DatabaseQueue implements queue.Queue backed by Ent.
type DatabaseQueue struct {
    db          *ent.Client
    handlers    map[string]queue.Handler
    concurrency int
    started     bool
}

// NewDatabaseQueue returns a ready-to-use DatabaseQueue.
func NewDatabaseQueue(db *ent.Client, concurrency int) *DatabaseQueue {
    return &DatabaseQueue{
        db:          db,
        handlers:    make(map[string]queue.Handler),
        concurrency: concurrency,
    }
}

func (q *DatabaseQueue) Enqueue(ctx context.Context, jobType string, payload []byte, opts ...queue.JobOption) (string, error) {
    j := &queue.Job{
        ID:         uuid.NewString(),
        Type:       jobType,
        Payload:    payload,
        Queue:      "default",
        Priority:   0,
        MaxRetries: 3,
        ProcessAt:  time.Now(),
        CreatedAt:  time.Now(),
        Status:     queue.JobStatusPending,
    }
    for _, opt := range opts {
        opt(j)
    }

    _, err := q.db.Job.Create().
        SetID(j.ID).
        SetType(j.Type).
        SetQueue(j.Queue).
        SetPayload(j.Payload).
        SetStatus(string(j.Status)).
        SetPriority(j.Priority).
        SetMaxRetries(j.MaxRetries).
        SetProcessAt(j.ProcessAt).
        SetCreatedAt(j.CreatedAt).
        Save(ctx)

    return j.ID, err
}

func (q *DatabaseQueue) RegisterHandler(jobType string, handler queue.Handler) {
    if q.started {
        panic("queue: RegisterHandler called after Start()")
    }
    q.handlers[jobType] = handler
}

func (q *DatabaseQueue) Start(ctx context.Context) error {
    q.started = true
    sem := make(chan struct{}, q.concurrency)
    go func() {
        ticker := time.NewTicker(pollInterval)
        defer ticker.Stop()
        for {
            select {
            case <-ctx.Done():
                return
            case <-ticker.C:
                sem <- struct{}{}
                go func() {
                    defer func() { <-sem }()
                    _ = q.processNext(ctx)
                }()
            }
        }
    }()
    return nil
}

func (q *DatabaseQueue) processNext(ctx context.Context) error {
    // Process next available job, with careful concurrency handling/transactions
    // Left abstracted to avoid lock-contention implementation details
    // depending on the underlying database dialect.
    return nil
}

func (q *DatabaseQueue) Inspect(ctx context.Context, jobID string) (*queue.Job, error) {
    record, err := q.db.Job.Query().Where(job.IDEQ(jobID)).Only(ctx)
    if err != nil {
        return nil, err
    }
    return &queue.Job{
        ID:         record.ID,
        Type:       record.Type,
        Queue:      record.Queue,
        Payload:    record.Payload,
        Status:     queue.JobStatus(record.Status),
        Priority:   record.Priority,
        MaxRetries: record.MaxRetries,
        RetryCount: record.RetryCount,
        ProcessAt:  record.ProcessAt,
        CreatedAt:  record.CreatedAt,
    }, nil
}

func (q *DatabaseQueue) Shutdown(ctx context.Context) error {
    return nil
}
```

**Retry backoff:** Exponential — `retryCount² × 30s` (30s, 120s, 270s, ...).

---

#### 2.5.2 Redis Backend (Asynq)

The Redis backend delegates to **Asynq** (`github.com/hibiken/asynq`), a production-grade Redis-backed job queue library. Asynq provides Redis Streams-based queuing, built-in retry with exponential backoff, dead-letter queues, delayed jobs, priority queues, and a web UI inspector.

The `internal/adapter/queue/redis/queue.go` adapter wraps Asynq's `Client` and `Server` types to satisfy the `queue.Queue` interface, translating `queue.Job` options into Asynq task options.

```go
// internal/adapter/queue/redis/queue.go
package redis

import (
    "context"
    "encoding/json"
    "time"

    "github.com/google/uuid"
    "github.com/hibiken/asynq"
    "opus/server/internal/shared/queue"
)

// RedisQueue implements queue.Queue backed by Asynq + Redis.
type RedisQueue struct {
    client   *asynq.Client
    server   *asynq.Server
    mux      *asynq.ServeMux
    handlers map[string]queue.Handler
    started  bool
}

// NewRedisQueue creates an Asynq client and server connected to Redis at the given URL.
func NewRedisQueue(redisURL string, concurrency int) (*RedisQueue, error) {
    opt, err := asynq.ParseRedisURI(redisURL)
    if err != nil {
        return nil, err
    }
    client := asynq.NewClient(opt)
    server := asynq.NewServer(opt, asynq.Config{
        Concurrency: concurrency,
        Queues: map[string]int{
            "agent":   10, // agent tasks: highest weight
            "default": 5,
            "email":   3,
        },
    })
    return &RedisQueue{
        client:   client,
        server:   server,
        mux:      asynq.NewServeMux(),
        handlers: make(map[string]queue.Handler),
    }, nil
}

func (q *RedisQueue) Enqueue(ctx context.Context, jobType string, payload []byte, opts ...queue.JobOption) (string, error) {
    job := &queue.Job{
        ID:         uuid.NewString(),
        Type:       jobType,
        Payload:    payload,
        Queue:      "default",
        Priority:   0,
        MaxRetries: 3,
        ProcessAt:  time.Now(),
    }
    for _, opt := range opts {
        opt(job)
    }

    asynqOpts := []asynq.Option{
        asynq.TaskID(job.ID),
        asynq.MaxRetry(job.MaxRetries),
        asynq.Queue(job.Queue),
    }
    if job.ProcessAt.After(time.Now()) {
        asynqOpts = append(asynqOpts, asynq.ProcessAt(job.ProcessAt))
    }

    task := asynq.NewTask(jobType, payload)
    info, err := q.client.EnqueueContext(ctx, task, asynqOpts...)
    if err != nil {
        return "", err
    }
    return info.ID, nil
}

func (q *RedisQueue) RegisterHandler(jobType string, handler queue.Handler) {
    if q.started {
        panic("queue: RegisterHandler called after Start()")
    }
    q.handlers[jobType] = handler
    // Wrap queue.Handler in asynq.HandlerFunc
    q.mux.HandleFunc(jobType, func(ctx context.Context, task *asynq.Task) error {
        job := &queue.Job{
            ID:      task.ResultWriter().TaskID(),
            Type:    task.Type(),
            Payload: task.Payload(),
            Queue:   "default",
        }
        return handler(ctx, job)
    })
}

func (q *RedisQueue) Start(ctx context.Context) error {
    q.started = true
    go func() {
        _ = q.server.Run(q.mux)
    }()
    return nil
}

func (q *RedisQueue) Shutdown(ctx context.Context) error {
    q.server.Shutdown()
    return q.client.Close()
}

func (q *RedisQueue) Inspect(ctx context.Context, jobID string) (*queue.Job, error) {
    inspector := asynq.NewInspector(q.client.(*asynq.Client))
    // Asynq inspector requires a queue name; search across known queues.
    for _, qname := range []string{"agent", "default", "email"} {
        info, err := inspector.GetTaskInfo(qname, jobID)
        if err != nil {
            continue
        }
        return &queue.Job{
            ID:         info.ID,
            Type:       info.Type,
            Payload:    info.Payload,
            Queue:      info.Queue,
            MaxRetries: info.MaxRetry,
            RetryCount: info.Retried,
            ProcessAt:  info.NextProcessAt,
            CreatedAt:  info.EnqueuedAt,
        }, nil
    }
    return nil, context.DeadlineExceeded // Note: context.DeadlineExceeded is used instead of sql.ErrNoRows
}
```

---

#### 2.5.3 In-Process EventBus Implementation

The EventBus is always in-process. No persistent backend is required. Topic matching supports exact strings and single-level wildcards (`*`).

```go
// internal/adapter/queue/memory/eventbus.go
package memory

import (
    "context"
    "path"
    "sync"

    "opus/server/internal/shared/queue"
)

type subscription struct {
    pattern string
    handler queue.EventHandler
}

// InMemoryEventBus implements queue.EventBus using in-process, synchronous dispatch.
type InMemoryEventBus struct {
    mu            sync.RWMutex
    subscriptions []subscription
}

// NewInMemoryEventBus returns a ready-to-use in-process EventBus.
func NewInMemoryEventBus() *InMemoryEventBus {
    return &InMemoryEventBus{}
}

// Subscribe registers a handler for events whose topic matches the given pattern.
// Supports glob-style wildcards: "agent.*" matches "agent.completed", etc.
func (b *InMemoryEventBus) Subscribe(topic string, handler queue.EventHandler) {
    b.mu.Lock()
    defer b.mu.Unlock()
    b.subscriptions = append(b.subscriptions, subscription{pattern: topic, handler: handler})
}

// Publish dispatches the event to all matching subscribers synchronously.
// Errors from individual handlers are logged but do not prevent other handlers
// from receiving the event.
func (b *InMemoryEventBus) Publish(ctx context.Context, event queue.Event) error {
    b.mu.RLock()
    subs := make([]subscription, len(b.subscriptions))
    copy(subs, b.subscriptions)
    b.mu.RUnlock()

    for _, sub := range subs {
        matched, err := path.Match(sub.pattern, event.Topic)
        if err != nil || !matched {
            continue
        }
        // Errors are intentionally swallowed here; callers should log within handlers.
        _ = sub.handler(ctx, event)
    }
    return nil
}
```

---

### 2.6 Queue Factory — Backend Selection at Startup

A factory function in `internal/adapter/queue/` selects and constructs the correct backend from the resolved `queue.Config`. This is the only place in the codebase that references concrete backend types.

```go
// internal/adapter/queue/factory.go
package queue

import (
    "fmt"

    "opus/server/ent"
    "opus/server/internal/adapter/queue/database"
    "opus/server/internal/adapter/queue/memory"
    "opus/server/internal/adapter/queue/redis"
    "opus/server/internal/shared/queue"
)

// NewQueue constructs and returns the Queue implementation specified by cfg.Driver.
func NewQueue(cfg queue.Config, db *ent.Client) (queue.Queue, error) {
    switch cfg.Driver {
    case queue.DriverDatabase:
        return database.NewDatabaseQueue(db, cfg.Concurrency), nil
    case queue.DriverRedis:
        return redis.NewRedisQueue(cfg.DSN, cfg.Concurrency)
    default:
        return nil, fmt.Errorf("queue: unsupported driver %q", cfg.Driver)
    }
}

// NewEventBus always returns the in-process EventBus implementation.
// The EventBus does not require a persistent backend.
func NewEventBus() queue.EventBus {
    return memory.NewInMemoryEventBus()
}
```

---

### 2.7 Dependency Injection

Both the `Queue` and `EventBus` are constructed once in `main.go` and injected into every Service that requires them. No global variables are used.

```go
// main.go (queue wiring excerpt)
package main

import (
    "context"
    "log"

    "opus/server/ent"
    adapterqueue "opus/server/internal/adapter/queue"
    "opus/server/internal/agent"
    "opus/server/internal/config"
    "opus/server/internal/shared/queue"
)

func main() {
    cfg, err := config.Load()
    if err != nil {
        log.Fatal(err)
    }

    // Initialize ent.Client (db)
    // client, err := ent.Open(...)
    // if err != nil {
    //     log.Fatal(err)
    // }
    // We assume 'client' is constructed.

    // Construct queue and event bus from config
    q, err := adapterqueue.NewQueue(cfg.Queue, client)
    if err != nil {
        log.Fatal(err)
    }
    bus := adapterqueue.NewEventBus()

    // Register job handlers before calling Start()
    agentService := agent.NewService(agentRepo, cfg.Agent, q, bus)
    q.RegisterHandler("agent:evaluate", agentService.HandleEvaluateJob)
    q.RegisterHandler("agent:retry",    agentService.HandleRetryJob)

    // Register event subscribers
    bus.Subscribe("agent.completed", workflowService.OnAgentCompleted)
    bus.Subscribe("agent.*",         notifService.OnAgentEvent)

    // Start the queue worker loop
    ctx := context.Background()
    if err := q.Start(ctx); err != nil {
        log.Fatal(err)
    }

    // ... HTTP server bootstrap
}
```

---

### 2.8 Job Type Naming Convention

All job types follow the `"<domain>:<action>"` convention. All event topics follow the `"<domain>.<action>"` convention (dot-separated to support wildcard matching).

| Job Type | Description |
|---|---|
| `agent:evaluate` | Trigger an autonomous agent evaluation cycle |
| `agent:retry` | Retry a failed agent task |
| `email:send` | Send a transactional email |
| `vault:index` | Re-index vault entries after a write |
| `workflow:trigger` | Trigger a workflow execution |

| Event Topic | Description |
|---|---|
| `agent.completed` | An agent run completed successfully |
| `agent.failed` | An agent run failed and was dead-lettered |
| `agent.started` | An agent run began execution |
| `vault.written` | A vault entry was created or updated |
| `workflow.completed` | A workflow execution completed |

---

### 2.9 Testing — NoopQueue and NoopEventBus

A `NoopQueue` and `NoopEventBus` are provided in `internal/shared/queue/noop.go` for unit tests. They discard all operations while satisfying the respective interfaces, eliminating the need to spin up a real queue backend in tests that do not exercise queue behaviour.

```go
// internal/shared/queue/noop.go
package queue

import "context"

// NoopQueue is a Queue implementation that discards all operations.
// Use in unit tests to satisfy Queue dependencies without a real backend.
type NoopQueue struct{}

func (n *NoopQueue) Enqueue(_ context.Context, _ string, _ []byte, _ ...JobOption) (string, error) {
    return "noop-job-id", nil
}
func (n *NoopQueue) RegisterHandler(_ string, _ Handler) {}
func (n *NoopQueue) Start(_ context.Context) error        { return nil }
func (n *NoopQueue) Shutdown(_ context.Context) error     { return nil }
func (n *NoopQueue) Inspect(_ context.Context, _ string) (*Job, error) {
    return &Job{ID: "noop-job-id", Status: JobStatusCompleted}, nil
}

// NoopEventBus is an EventBus implementation that discards all operations.
// Use in unit tests to satisfy EventBus dependencies without real dispatch.
type NoopEventBus struct{}

func (n *NoopEventBus) Publish(_ context.Context, _ Event) error { return nil }
func (n *NoopEventBus) Subscribe(_ string, _ EventHandler)       {}
```

For tests that assert specific enqueue or publish calls, generate mocks using `go.uber.org/mock`:

```go
//go:generate mockgen -destination=mock_queue.go    -package=queue . Queue
//go:generate mockgen -destination=mock_eventbus.go -package=queue . EventBus
```

---

### 2.10 Alignment with Existing ADRs

| Concern | This ADR | Related ADR |
|---|---|---|
| Interface-driven design | `queue.Queue` + `queue.EventBus` interfaces | ADR-001 (Repository pattern) |
| Feature config co-location | `queue.Config` owned by queue package | ADR-002 (Hybrid composition) |
| Delivery layer logging | `logger.Logger` injected into queue workers | ADR-006 (Logger interface) |
| Domain decoupling via EventBus | `agent` → `workflow` via `bus.Publish` | ADR-001 (Dependency rule) |

---

## 3. Alternatives Considered

### 3.1 Single Abstraction (Queue Only, No EventBus)

Use the `Queue` interface for both background jobs and domain event dispatch. Rejected because:

- Domain events are ephemeral and in-process; adding them to a persistent queue introduces unnecessary serialisation and database overhead for events that do not require durability.
- A pub/sub EventBus provides wildcard topic matching and multi-subscriber fan-out natively; emulating this with a job queue requires manual routing logic.
- Separating concerns allows the EventBus to remain synchronous and zero-latency while the Queue handles durable, retriable work.

### 3.2 Third-Party Queue Library as Primary Abstraction (Asynq / River)

Adopt Asynq or pgx-backed River as the single queue solution without an intermediate interface. Rejected because:

- Asynq requires Redis; forcing Redis as a hard dependency violates the zero-mandatory-cost principle.
- River requires PostgreSQL; precludes SQLite for local-first single-user deployments.
- A thin interface layer allows backend selection per deployment without modifying any application code.

### 3.3 Channel-Based In-Process Queue

Use Go channels for all async work. Rejected because:

- Channels are not durable; pending jobs are lost on server restart.
- No built-in retry, dead-letter, delay, or priority semantics.
- Does not scale to multi-process deployments if Opus Server is ever run as multiple replicas.

### 3.4 Dedicated Message Broker (NATS, RabbitMQ)

Introduce a standalone broker process. Rejected because:

- Violates the local-first, zero-mandatory-infrastructure principle.
- Adds operational complexity (process management, network configuration) for a self-hosted tool.
- A configurable Database (Ent)/Redis backend covers all required use cases without external broker process management.

---

## 4. Consequences

### 4.1 Positive

- **Database-agnostic default** — The Ent-backed default queue runs on any database supported by Ent (e.g. SQLite for zero config, PostgreSQL for production), unifying code paths.
- **Configurable scale** — Redis backend via Asynq allows Opus to scale to high-throughput production workloads without API or interface changes.
- **Domain decoupling** — EventBus eliminates direct imports between feature domains, preserving the strict dependency rule from ADR-001.
- **Durability by default** — All Queue backends persist jobs to disk or Redis; no job is lost on server restart.
- **Testability** — `NoopQueue` and `NoopEventBus` eliminate infrastructure setup in unit tests; mock generation provides precise call assertions.
- **Retry and dead-letter** — Built into the Queue interface and all implementations; failed agent tasks are automatically retried with exponential backoff.

### 4.2 Negative / Trade-offs

- **Polling overhead** — The database backend uses a 500ms polling interval rather than event-driven notification; acceptable for MVP workloads but introduces up to 500ms latency between enqueue and processing.
- **EventBus is not durable** — In-process pub/sub events are lost if the server crashes between `Publish` and handler execution. For critical cross-domain side effects that require guaranteed delivery, use the `Queue` instead of the `EventBus`.
- **Asynq wrapping complexity** — The Redis backend wraps Asynq types; Asynq's internal retry logic (exponential backoff, jitter) may diverge slightly from the database implementations. The interface contract is satisfied, but retry timing is not identical across backends.
- **Lock Contention in Database** — Depending on the underlying database dialect and locking strategy, database-backed queues can experience lock contention at high concurrency. Switch to Redis via Asynq for higher throughput.
- **EventBus wildcard depth** — Wildcard matching uses `path.Match`, which supports single-level `*` but not recursive `**`. Deep topic hierarchies beyond two levels are not supported in the initial implementation.

---

## 5. References

- [ADR-001: Server Clean Architecture](./ADR-001-server-clean-architecture.md)
- [ADR-002: Configuration Management](./ADR-002-server-configuration.md)
- [ADR-005: Server Delivery Layer with GoFiber v3](./ADR-005-server-delivery-layer-with-gofiber-v3.md)
- [ADR-006: Server Logger Architecture](./ADR-006-server-logger.md)
- [Asynq — github.com/hibiken/asynq](https://github.com/hibiken/asynq)
- [go.uber.org/mock](https://github.com/uber-go/mock)
- [path.Match — Go standard library](https://pkg.go.dev/path#Match)