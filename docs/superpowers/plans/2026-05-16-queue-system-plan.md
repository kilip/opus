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

- [x] **Step 1: Create Go models**
- [x] **Step 2: Define EntGo schemas**
- [x] **Step 3: Generate EntGo code**
- [x] **Step 4: Commit**
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

- [x] **Step 1: Define Service interfaces**
- [x] **Step 2: Define Repository interface**
- [x] **Step 3: Commit**
```bash
git add api/internal/service api/internal/repository/queue/driver.go
git commit -m "feat: define queue system interfaces"
```

---

### Task 3: EntGo Queue Driver Implementation

**Files:**
- Create: `api/internal/repository/queue/entgo.go`
- Create: `api/internal/repository/queue/entgo_integration_test.go`

- [x] **Step 1: Implement EntGo driver**
- [x] **Step 2: Write integration test**
- [x] **Step 3: Run integration test**
- [x] **Step 4: Commit**
```bash
git add api/internal/repository/queue/entgo.go api/internal/repository/queue/entgo_integration_test.go
git commit -m "feat: implement EntGo queue driver"
```

---

### Task 4: Worker Engine Core

**Files:**
- Create: `api/internal/worker/engine.go`
- Create: `api/internal/worker/engine_test.go`

- [x] **Step 1: Implement Worker Engine**
- [x] **Step 2: Implement Exponential Backoff**
- [x] **Step 3: Write unit test with mocks**
- [x] **Step 4: Run tests**
- [x] **Step 5: Commit**
```bash
git add api/internal/worker/engine.go api/internal/worker/engine_test.go
git commit -m "feat: implement worker engine core"
```

---

### Task 5: Cron Scheduler Implementation

**Files:**
- Create: `api/internal/worker/scheduler.go`
- Create: `api/internal/worker/scheduler_test.go`

- [x] **Step 1: Implement Cron Scheduler**
- [x] **Step 2: Write unit test**
- [x] **Step 3: Run tests**
- [x] **Step 4: Commit**
```bash
git add api/internal/worker/scheduler.go api/internal/worker/scheduler_test.go
git commit -m "feat: implement cron scheduler"
```

---

### Task 6: Redis Driver Implementation

**Files:**
- Create: `api/internal/repository/queue/redis.go`
- Create: `api/internal/repository/queue/redis_test.go`

- [x] **Step 1: Implement Redis driver**
- [x] **Step 2: Write unit test**
- [x] **Step 3: Commit**
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

- [x] **Step 1: Add Queue Config**
- [x] **Step 2: Implement Driver Factory**
- [x] **Step 3: Wire up in Start Command**
- [x] **Step 4: Commit**
```bash
git add api/internal/config api/cmd/opus/start.go
git commit -m "feat: integrate queue system into application startup"
```

---

### Task 8: Dead Letter Queue API

**Files:**
- Create: `api/internal/delivery/fiber/handler/queue.go`
- Modify: `api/internal/delivery/fiber/server.go`

- [x] **Step 1: Implement Dead Letter Handlers**
- [x] **Step 2: Register Routes**
- [x] **Step 3: Commit**
```bash
git add api/internal/delivery/fiber
git commit -m "feat: add dead letter queue API endpoints"
```
