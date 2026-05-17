# Server Taskfile Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [x]`) syntax for tracking.

**Goal:** Create a `Taskfile.yml` configuration in `server/` to standardize development tasks (linting, testing, mocks generation, dependency management, config schema generation, compilation) using the Task runner.

**Architecture:** A unified, declarative `Taskfile.yml` situated in the `server/` root. Tasks will be highly descriptive, leveraging standard CLI commands defined in project guidelines (`go test`, `go generate`, `golangci-lint`).

**Tech Stack:** Go 1.26, Task runner (CLI).

---

### Task 1: Setup Taskfile and Default Task

**Files:**
- Create: `server/Taskfile.yml`

- [x] **Step 1: Write initial Taskfile with default task**

Create the file `server/Taskfile.yml` with the following contents:
```yaml
version: '3'

tasks:
  default:
    desc: Lists all available tasks
    cmds:
      - task --list
    silent: true
```

- [x] **Step 2: Verify default task listing**

Run: `cd server && task --list`
Expected output:
```
task: Available tasks for this project:
* default:             Lists all available tasks
```

- [x] **Step 3: Commit changes**

Run:
```bash
cd server
git add Taskfile.yml
git commit -m "chore(server): initialize taskfile with default task"
```

---

### Task 2: Implement Dependency Management and Code Generation Tasks

**Files:**
- Modify: `server/Taskfile.yml`

- [x] **Step 1: Add tidy and generate tasks**

Update `server/Taskfile.yml` to include the `tidy` and `generate` tasks:
```yaml
version: '3'

tasks:
  default:
    desc: Lists all available tasks
    cmds:
      - task --list
    silent: true

  tidy:
    desc: Run go mod tidy to clean up dependencies
    cmds:
      - go mod tidy

  generate:
    desc: Run code generation (mockgen, ent, etc.)
    cmds:
      - go generate ./...
```

- [x] **Step 2: Run and verify the tidy task**

Run: `cd server && task tidy`
Expected: Completes successfully with exit code 0.

- [x] **Step 3: Run and verify the generate task**

Run: `cd server && task generate`
Expected: Completes successfully with exit code 0 (runs `go generate` across all internal packages).

- [x] **Step 4: Commit changes**

Run:
```bash
cd server
git add Taskfile.yml
git commit -m "chore(server): add tidy and generate tasks to taskfile"
```

---

### Task 3: Implement Quality and Testing Tasks

**Files:**
- Modify: `server/Taskfile.yml`

- [x] **Step 1: Add test, test:integration, and lint tasks**

Update `server/Taskfile.yml` to include testing and linting commands:
```yaml
version: '3'

tasks:
  default:
    desc: Lists all available tasks
    cmds:
      - task --list
    silent: true

  tidy:
    desc: Run go mod tidy to clean up dependencies
    cmds:
      - go mod tidy

  generate:
    desc: Run code generation (mockgen, ent, etc.)
    cmds:
      - go generate ./...

  test:
    desc: Run unit tests with race detector
    cmds:
      - go test -race ./...

  test:integration:
    desc: Run unit and integration tests with race detector
    cmds:
      - go test -race -tags integration ./...

  lint:
    desc: Run golangci-lint
    cmds:
      - golangci-lint run ./...
```

- [x] **Step 2: Verify test task**

Run: `cd server && task test`
Expected: Executes unit tests using the `-race` detector flag and finishes with a success/coverage report.

- [x] **Step 3: Verify lint task**

Run: `cd server && task lint`
Expected: Runs `golangci-lint` check across the backend codebase and completes successfully.

- [x] **Step 4: Commit changes**

Run:
```bash
cd server
git add Taskfile.yml
git commit -m "chore(server): add testing and linting tasks to taskfile"
```

---

### Task 4: Implement Schema Generation and Compilation Tasks

**Files:**
- Modify: `server/Taskfile.yml`

- [x] **Step 1: Add schema and build tasks**

Update `server/Taskfile.yml` to include the `schema` generation task and the project `build` task:
```yaml
version: '3'

tasks:
  default:
    desc: Lists all available tasks
    cmds:
      - task --list
    silent: true

  tidy:
    desc: Run go mod tidy to clean up dependencies
    cmds:
      - go mod tidy

  generate:
    desc: Run code generation (mockgen, ent, etc.)
    cmds:
      - go generate ./...

  test:
    desc: Run unit tests with race detector
    cmds:
      - go test -race ./...

  test:integration:
    desc: Run unit and integration tests with race detector
    cmds:
      - go test -race -tags integration ./...

  lint:
    desc: Run golangci-lint
    cmds:
      - golangci-lint run ./...

  schema:
    desc: Generate JSON schema for configurations
    dir: internal/config
    cmds:
      - go run generate.go

  build:
    desc: Compile the server packages
    cmds:
      - go build ./...
```

- [x] **Step 2: Verify config schema generation**

Run: `cd server && task schema`
Expected: Re-generates configuration schema output to `docs/config.schema.json` without errors.

- [x] **Step 3: Verify server compilation**

Run: `cd server && task build`
Expected: Compiles Go packages successfully with exit code 0.

- [x] **Step 4: Commit changes**

Run:
```bash
cd server
git add Taskfile.yml
git commit -m "chore(server): add configuration schema and build tasks to taskfile"
```
