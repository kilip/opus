# Modular Queue System Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement a driver-agnostic, priority-aware background queue and worker system for the Opus platform.

**Architecture:** Clean Architecture with interfaces in `service/`, driver implementations in `repository/queue/`, and a goroutine-based `worker/engine`. Supports persistence, retries with exponential backoff, cron scheduling, and dead-letter queues.

**Tech Stack:** Go (GoFiber v3), EntGo, Redis, robfig/cron/v3, Viper, go.uber.org/mock.

---

### Task 1: Data Models & EntGo Schemas

**Files:**
- Create: `api/internal/model/job.go`
- Create: `api/internal/model/cron.go`
- Create: `api/ent/schema/job.go`
- Create: `api/ent/schema/cron_schedule.go`
- Create: `api/ent/schema/dead_letter.go`

- [ ] **Step 1: Create Go models**
Implement the `Job`, `JobStatus`, `DeadLetter`, and `CronSchedule` structs in `api/internal/model/`.

- [ ] **Step 2: Define EntGo schemas**
Create the schema files in `api/ent/schema/` with appropriate fields and indices (e.g., index on `status` and `priority` for `Job`).

- [ ] **Step 3: Generate EntGo code**
Run: `cd api && task ent:generate`
Expected: Successful generation of EntGo files in `api/ent/`.

- [ ] **Step 4: Commit**
```bash
git add api/internal/model api/ent/schema api/ent
git commit -m "feat: add queue and cron schemas and models"
```

---

### Task 2: Core Interfaces

**Files:**
- Create: `api/internal/service/queue.go`
- Create: `api/internal/service/worker.go`
- Create: `api/internal/repository/queue/driver.go`

- [ ] **Step 1: Define Service interfaces**
Implement `Queue`, `Scheduler`, and `Worker` interfaces in `api/internal/service/`.

- [ ] **Step 2: Define Repository interface**
Implement `QueueDriver` interface in `api/internal/repository/queue/driver.go`.

- [ ] **Step 3: Commit**
```bash
git add api/internal/service api/internal/repository/queue/driver.go
git commit -m "feat: define queue system interfaces"
```

---

### Task 3: EntGo Queue Driver Implementation

**Files:**
- Create: `api/internal/repository/queue/entgo.go`
- Create: `api/internal/repository/queue/entgo_integration_test.go`

- [ ] **Step 1: Implement EntGo driver**
Implement `Push`, `Pop`, `UpdateStatus`, and `MoveToDead` using the EntGo client. Handle SQLite mutex locking for `Pop`.

- [ ] **Step 2: Write integration test**
Write a test that performs a full cycle: Push -> Pop -> UpdateStatus.

- [ ] **Step 3: Run integration test**
Run: `cd api && task test:integration`
Expected: PASS

- [ ] **Step 4: Commit**
```bash
git add api/internal/repository/queue/entgo.go api/internal/repository/queue/entgo_integration_test.go
git commit -m "feat: implement EntGo queue driver"
```

---

### Task 4: Worker Engine Core

**Files:**
- Create: `api/internal/worker/engine.go`
- Create: `api/internal/worker/engine_test.go`

- [ ] **Step 1: Implement Worker Engine**
Implement the `Engine` struct with goroutine pool logic, job polling, and handler dispatching.

- [ ] **Step 2: Implement Exponential Backoff**
Add the `backoff` helper function as described in the spec.

- [ ] **Step 3: Write unit test with mocks**
Mock the `QueueDriver` and verify that the engine polls jobs and calls the correct handler.

- [ ] **Step 4: Run tests**
Run: `cd api && go test ./internal/worker/...`
Expected: PASS

- [ ] **Step 5: Commit**
```bash
git add api/internal/worker/engine.go api/internal/worker/engine_test.go
git commit -m "feat: implement worker engine core"
```

---

### Task 5: Cron Scheduler Implementation

**Files:**
- Create: `api/internal/worker/scheduler.go`
- Create: `api/internal/worker/scheduler_test.go`

- [ ] **Step 1: Implement Cron Scheduler**
Implement the ticker-based scheduler that polls `CronSchedule` and enqueues jobs. Use `robfig/cron/v3` for expression parsing.

- [ ] **Step 2: Write unit test**
Verify that the scheduler correctly calculates the next run time and enqueues the job.

- [ ] **Step 3: Run tests**
Run: `cd api && go test ./internal/worker/scheduler_test.go`
Expected: PASS

- [ ] **Step 4: Commit**
```bash
git add api/internal/worker/scheduler.go api/internal/worker/scheduler_test.go
git commit -m "feat: implement cron scheduler"
```

---

### Task 6: Redis Driver Implementation

**Files:**
- Create: `api/internal/repository/queue/redis.go`
- Create: `api/internal/repository/queue/redis_test.go`

- [ ] **Step 1: Implement Redis driver**
Implement the `QueueDriver` interface using Redis sorted sets (ZSET) for priority.

- [ ] **Step 2: Write unit test**
Mock the Redis client or use a local Redis if available (miniredis is recommended if possible, or just unit test the logic).

- [ ] **Step 3: Commit**
```bash
git add api/internal/repository/queue/redis.go api/internal/repository/queue/redis_test.go
git commit -m "feat: implement Redis queue driver"
```

---

### Task 7: Config & Integration

**Files:**
- Modify: `api/internal/config/config.go`
- Create: `api/internal/config/queue.go`
- Modify: `api/cmd/opus/start.go`

- [ ] **Step 1: Add Queue Config**
Add `QueueConfig` and `RedisConfig` to the main config struct.

- [ ] **Step 2: Implement Driver Factory**
Implement `GetQueueDriver()` in `api/internal/config/queue.go`.

- [ ] **Step 3: Wire up in Start Command**
Initialize the `Worker Engine` and `Scheduler` in `cmd/opus/start.go` and handle graceful shutdown.

- [ ] **Step 4: Commit**
```bash
git add api/internal/config api/cmd/opus/start.go
git commit -m "feat: integrate queue system into application startup"
```

---

### Task 8: Dead Letter Queue API

**Files:**
- Create: `api/internal/delivery/fiber/queue.go`
- Modify: `api/internal/delivery/fiber/router.go`

- [ ] **Step 1: Implement Dead Letter Handlers**
Implement `ListDeadLetters`, `RetryJob`, and `DeleteDeadLetter` handlers.

- [ ] **Step 2: Register Routes**
Add the `/api/v1/queue/dead` routes to the Fiber router.

- [ ] **Step 3: Commit**
```bash
git add api/internal/delivery/fiber
git commit -m "feat: add dead letter queue API endpoints"
```
