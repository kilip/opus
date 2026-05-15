# Phase 1 — Monorepo Setup & Tooling

**Product:** Opus  
**Version:** 1.0.1  
**Status:** Completed  
**Last Updated:** 2026-05-15  
**Authors:** Product & Architecture Team

---

## Phase Goal

Establish the complete monorepo skeleton: directory structure, task automation, Docker configuration, CI/CD scaffolding, and environment variable documentation. No business logic is implemented in this phase. The output is a clean, runnable project scaffold that all future phases build upon.

---

## Prerequisites

Before starting this phase, ensure the following tools are installed on the development machine:

- Go 1.23+
- Node.js LTS (22.x)
- pnpm (`npm install -g pnpm`)
- Task (`go install github.com/go-task/task/v3/cmd/task@latest`)
- Docker + Docker Compose
- Git

---

## Context for AI Agent

> Provide `docs/CONVENTIONS.md`, `docs/PRD.md`, and `docs/ARCHITECTURE.md` alongside this file to your AI agent before starting.

**You are scaffolding the Opus monorepo.** Opus is a self-hosted AI agent platform with two components: a Go backend (`api/`) and a Next.js 16 frontend (`dash/`). This phase creates only the skeleton — no Go or TypeScript source code is written yet.

---

## P1-T1 — Initialize Monorepo Directory Structure

### What to Do

Create the complete directory and file skeleton for the Opus monorepo. Create empty placeholder files (`.gitkeep`) in directories that will be populated in later phases.

### Directory Structure to Create

```
opus/
├── api/
│   ├── cmd/opus/
│   ├── internal/
│   │   ├── model/
│   │   ├── service/
│   │   ├── repository/
│   │   ├── handler/
│   │   ├── middleware/
│   │   └── config/
│   ├── ent/schema/
│   └── .gitkeep
├── dash/
│   ├── app/
│   │   ├── (auth)/login/
│   │   ├── (dash)/
│   │   └── offline/
│   ├── components/
│   │   ├── ui/
│   │   └── shared/
│   ├── lib/
│   │   ├── api/
│   │   └── utils/
│   ├── public/icons/
│   └── .gitkeep
├── docs/
├── .github/
│   └── workflows/
├── .gitignore
└── README.md (empty placeholder)
```

### Constraints

- Do not create any Go `.go` files or TypeScript `.ts`/`.tsx` files yet.
- Do not run `go mod init` or `pnpm init` yet — those are done in Phase 2 and Phase 3.
- Use `.gitkeep` files to preserve empty directories in Git.

### Acceptance Criteria

- [x] Directory `opus/api/cmd/opus/` exists.
- [x] Directory `opus/api/internal/model/` exists.
- [x] Directory `opus/api/internal/service/` exists.
- [x] Directory `opus/api/internal/repository/` exists.
- [x] Directory `opus/api/internal/handler/` exists.
- [x] Directory `opus/api/internal/middleware/` exists.
- [x] Directory `opus/api/internal/config/` exists.
- [x] Directory `opus/api/ent/schema/` exists.
- [x] Directory `opus/dash/app/(auth)/login/` exists.
- [x] Directory `opus/dash/app/(dash)/` exists.
- [x] Directory `opus/dash/app/offline/` exists.
- [x] Directory `opus/dash/components/ui/` exists.
- [x] Directory `opus/dash/components/shared/` exists.
- [x] Directory `opus/dash/lib/api/` exists.
- [x] Directory `opus/dash/lib/utils/` exists.
- [x] Directory `opus/dash/public/icons/` exists.
- [x] Directory `opus/.github/workflows/` exists.
- [x] Directory `opus/docs/` exists.
- [x] File `opus/.gitignore` exists (see content below).

### `.gitignore` Content

```gitignore
# Go
api/bin/
api/tmp/
*.exe
*.test

# Node.js
dash/node_modules/
dash/.next/
dash/out/
dash/.pnpm-store/

# Environment
.env
.env.local
.env.*.local

# OS
.DS_Store
Thumbs.db

# Editors
.vscode/
.idea/
*.swp

# Docker
*.log

# EntGo generated (do not ignore, but do not manually edit)
# ent/ is committed
```

---

## P1-T2 — Configure Root `Taskfile.yml`

### What to Do

Create the root `Taskfile.yml` at `opus/Taskfile.yml`. This file is a **pure orchestrator** — it delegates all tasks to `api/Taskfile.yml` and `dash/Taskfile.yml` using the `includes` directive. It does not define build logic directly.

### File to Create

`opus/Taskfile.yml`

### Content

```yaml
# opus/Taskfile.yml
version: "3"

includes:
  api:
    taskfile: ./api/Taskfile.yml
    dir: ./api

  dash:
    taskfile: ./dash/Taskfile.yml
    dir: ./dash

tasks:
  setup:
    desc: Install all dependencies for api/ and dash/
    deps: [api:setup, dash:setup]

  dev:
    desc: Start api/ and dash/ in development mode concurrently
    deps: [api:dev, dash:dev]

  build:
    desc: Build api/ binary and dash/ production bundle
    deps: [api:build, dash:build]

  test:all:
    desc: Run all tests across api/ and dash/
    deps: [api:test:all, dash:test]

  lint:
    desc: Lint api/ and dash/
    deps: [api:lint, dash:lint]

  migrate:
    desc: Run database migrations
    cmds:
      - task: api:migrate
```

### Constraints

- Do not add any task that contains build logic directly — all logic lives in the sub-Taskfiles.
- The `includes` section must reference `./api/Taskfile.yml` and `./dash/Taskfile.yml`.

### Acceptance Criteria

- [x] File `opus/Taskfile.yml` exists.
- [x] Contains `includes` block with `api` and `dash` entries.
- [x] Task `setup` delegates to `api:setup` and `dash:setup`.
- [x] Task `dev` delegates to `api:dev` and `dash:dev`.
- [x] Task `build` delegates to `api:build` and `dash:build`.
- [x] Task `test:all` delegates to `api:test:all` and `dash:test`.
- [x] Task `lint` delegates to `api:lint` and `dash:lint`.
- [x] Task `migrate` delegates to `api:migrate`.
- [x] Running `task --list` from `opus/` lists all tasks without errors.

---

## P1-T3 — Configure Root `.env.example`

### What to Do

Create `opus/.env.example` documenting every `OPUS_*` environment variable consumed by the API. This file is the **canonical reference** for all environment-based configuration.

### File to Create

`opus/.env.example`

### Content

```dotenv
# =============================================================================
# Opus Environment Variables
# All variables are prefixed with OPUS_
# These override values in ~/.opus/config.toml
# =============================================================================

# -----------------------------------------------------------------------------
# Server
# -----------------------------------------------------------------------------
OPUS_SERVER_PORT=8080
OPUS_SERVER_ENV=production          # development | production

# -----------------------------------------------------------------------------
# Database
# -----------------------------------------------------------------------------
OPUS_DATABASE_DRIVER=sqlite         # sqlite | postgres
OPUS_DATABASE_DSN=~/.opus/opus.db  # For postgres: postgres://user:pass@host:5432/db

# -----------------------------------------------------------------------------
# Auth — JWT
# -----------------------------------------------------------------------------
OPUS_AUTH_SECRET=                   # REQUIRED — JWT signing secret (min 32 chars)
OPUS_AUTH_ACCESS_TOKEN_TTL=15       # Access token TTL in minutes
OPUS_AUTH_REFRESH_TOKEN_TTL=10080   # Refresh token TTL in minutes (7 days)

# -----------------------------------------------------------------------------
# Auth — Google OAuth2
# -----------------------------------------------------------------------------
OPUS_AUTH_GOOGLE_CLIENT_ID=
OPUS_AUTH_GOOGLE_CLIENT_SECRET=
OPUS_AUTH_GOOGLE_REDIRECT_URL=http://localhost:8080/auth/google/callback

# -----------------------------------------------------------------------------
# Auth — GitHub OAuth2
# -----------------------------------------------------------------------------
OPUS_AUTH_GITHUB_CLIENT_ID=
OPUS_AUTH_GITHUB_CLIENT_SECRET=
OPUS_AUTH_GITHUB_REDIRECT_URL=http://localhost:8080/auth/github/callback
```

### Constraints

- Never create a `.env` file — only `.env.example`.
- Every variable must have an inline comment explaining its purpose and valid values.
- `OPUS_AUTH_SECRET` must be marked as `REQUIRED`.

### Acceptance Criteria

- [x] File `opus/.env.example` exists.
- [x] Contains `OPUS_SERVER_PORT` and `OPUS_SERVER_ENV`.
- [x] Contains `OPUS_DATABASE_DRIVER` and `OPUS_DATABASE_DSN`.
- [x] Contains `OPUS_AUTH_SECRET` marked as REQUIRED.
- [x] Contains `OPUS_AUTH_ACCESS_TOKEN_TTL` and `OPUS_AUTH_REFRESH_TOKEN_TTL`.
- [x] Contains all four Google OAuth2 variables.
- [x] Contains all four GitHub OAuth2 variables.
- [x] Every variable has an inline comment.

---

## P1-T4 — Configure `docker-compose.yml` (Production)

### What to Do

Create `opus/docker-compose.yml` for production deployment. This file orchestrates three services: `api`, `dash`, and `db` (PostgreSQL). It must also include a SQLite-only variant via comments.

### File to Create

`opus/docker-compose.yml`

### Content

```yaml
# opus/docker-compose.yml
# Production configuration
# For SQLite deployment, remove the db service and set:
#   OPUS_DATABASE_DRIVER=sqlite
#   OPUS_DATABASE_DSN=/data/opus.db
#   Add volume: ./data:/data to api service

services:
  api:
    image: ghcr.io/opus/opus-api:latest
    restart: unless-stopped
    environment:
      OPUS_SERVER_PORT: "8080"
      OPUS_SERVER_ENV: "production"
      OPUS_DATABASE_DRIVER: "postgres"
      OPUS_DATABASE_DSN: "postgres://opus:secret@db:5432/opus"
      OPUS_AUTH_SECRET: "${OPUS_AUTH_SECRET}"
      OPUS_AUTH_GOOGLE_CLIENT_ID: "${OPUS_AUTH_GOOGLE_CLIENT_ID}"
      OPUS_AUTH_GOOGLE_CLIENT_SECRET: "${OPUS_AUTH_GOOGLE_CLIENT_SECRET}"
      OPUS_AUTH_GOOGLE_REDIRECT_URL: "${OPUS_AUTH_GOOGLE_REDIRECT_URL}"
      OPUS_AUTH_GITHUB_CLIENT_ID: "${OPUS_AUTH_GITHUB_CLIENT_ID}"
      OPUS_AUTH_GITHUB_CLIENT_SECRET: "${OPUS_AUTH_GITHUB_CLIENT_SECRET}"
      OPUS_AUTH_GITHUB_REDIRECT_URL: "${OPUS_AUTH_GITHUB_REDIRECT_URL}"
    ports:
      - "8080:8080"
    depends_on:
      db:
        condition: service_healthy

  dash:
    image: ghcr.io/opus/opus-dash:latest
    restart: unless-stopped
    ports:
      - "3000:3000"
    environment:
      NEXT_PUBLIC_API_URL: "http://localhost:8080"
    depends_on:
      - api

  db:
    image: postgres:16-alpine
    restart: unless-stopped
    environment:
      POSTGRES_USER: opus
      POSTGRES_PASSWORD: secret
      POSTGRES_DB: opus
    volumes:
      - pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U opus"]
      interval: 10s
      timeout: 5s
      retries: 5

volumes:
  pgdata:
```

### Constraints

- Use `ghcr.io/opus/opus-api:latest` and `ghcr.io/opus/opus-dash:latest` as image names.
- The `api` service must depend on `db` with `condition: service_healthy`.
- Secrets must use `${VARIABLE}` substitution — never hardcoded values.
- Include a comment block explaining the SQLite-only variant.

### Acceptance Criteria

- [x] File `opus/docker-compose.yml` exists.
- [x] Contains `api`, `dash`, and `db` services.
- [x] `api` service uses environment variable substitution for secrets.
- [x] `db` service has a `healthcheck` block.
- [x] `api` depends on `db` with `condition: service_healthy`.
- [x] `dash` depends on `api`.
- [x] `pgdata` volume is defined.
- [x] Comment block explaining SQLite-only variant is present.
- [x] Running `docker compose config` validates without errors.

---

## P1-T5 — Configure `docker-compose.dev.yml` (Development)

### What to Do

Create `opus/docker-compose.dev.yml` as a development override file. This file enables live reload for both `api/` and `dash/` and exposes additional ports for debugging.

### File to Create

`opus/docker-compose.dev.yml`

### Content

```yaml
# opus/docker-compose.dev.yml
# Development overrides — use with:
#   docker compose -f docker-compose.yml -f docker-compose.dev.yml up

services:
  api:
    build:
      context: ./api
      dockerfile: Dockerfile
    volumes:
      - ./api:/app
    environment:
      OPUS_SERVER_ENV: "development"
      OPUS_DATABASE_DRIVER: "sqlite"
      OPUS_DATABASE_DSN: "/app/tmp/opus.db"

  dash:
    build:
      context: ./dash
      dockerfile: Dockerfile
    volumes:
      - ./dash:/app
      - /app/node_modules
      - /app/.next
    environment:
      NODE_ENV: "development"
      NEXT_PUBLIC_API_URL: "http://localhost:8080"
    ports:
      - "3000:3000"
```

### Constraints

- Do not define a `db` service — development uses SQLite.
- Mount `node_modules` and `.next` as anonymous volumes to prevent host override.
- Development must use `OPUS_SERVER_ENV: "development"` to enable Email/Password login.

### Acceptance Criteria

- [x] File `opus/docker-compose.dev.yml` exists.
- [x] `api` service uses `build.context: ./api`.
- [x] `api` service mounts `./api:/app`.
- [x] `api` service sets `OPUS_SERVER_ENV: "development"` and `OPUS_DATABASE_DRIVER: "sqlite"`.
- [x] `dash` service mounts `./dash:/app`.
- [x] `dash` service has anonymous volumes for `node_modules` and `.next`.
- [x] Usage comment at the top of the file is present.
- [x] Running `docker compose -f docker-compose.yml -f docker-compose.dev.yml config` validates without errors.

---

## P1-T6 — Scaffold GitHub Actions Workflows

### What to Do

Create three GitHub Actions workflow files under `opus/.github/workflows/`. These are scaffolds — they define the structure and steps but reference tasks that will be fully functional after Phase 2 and Phase 3 are complete.

### Files to Create

- `opus/.github/workflows/ci.yml`
- `opus/.github/workflows/build.yml`
- `opus/.github/workflows/release.yml`

### `ci.yml` — Continuous Integration

```yaml
# .github/workflows/ci.yml
name: CI

on:
  pull_request:
    branches: [main]

jobs:
  test:
    name: Lint & Test
    runs-on: ubuntu-latest

    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: "1.23"

      - name: Setup Node.js
        uses: actions/setup-node@v4
        with:
          node-version: "22"

      - name: Install pnpm
        uses: pnpm/action-setup@v3
        with:
          version: 9

      - name: Install Task
        uses: arduino/setup-task@v2

      - name: Lint API
        run: task api:lint

      - name: Test API
        run: task api:test:all

      - name: Lint Dash
        run: task dash:lint

      - name: Test Dash
        run: task dash:test
```

### `build.yml` — Build & Docker

```yaml
# .github/workflows/build.yml
name: Build

on:
  push:
    branches: [main]

jobs:
  build:
    name: Build & Push Docker Images
    runs-on: ubuntu-latest

    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: "1.23"

      - name: Setup Node.js
        uses: actions/setup-node@v4
        with:
          node-version: "22"

      - name: Install pnpm
        uses: pnpm/action-setup@v3
        with:
          version: 9

      - name: Install Task
        uses: arduino/setup-task@v2

      - name: Build API
        run: task api:build

      - name: Build Dash
        run: task dash:build

      - name: Login to GHCR
        uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}

      - name: Build and Push API Image
        uses: docker/build-push-action@v5
        with:
          context: ./api
          push: true
          tags: ghcr.io/opus/opus-api:latest

      - name: Build and Push Dash Image
        uses: docker/build-push-action@v5
        with:
          context: ./dash
          push: true
          tags: ghcr.io/opus/opus-dash:latest
```

### `release.yml` — Release

```yaml
# .github/workflows/release.yml
name: Release

on:
  push:
    tags:
      - "v*"

jobs:
  release:
    name: Cross-Compile & Publish
    runs-on: ubuntu-latest

    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: "1.23"

      - name: Cross-Compile Binaries
        run: |
          TARGETS=(
            "linux/amd64"
            "linux/arm64"
            "darwin/amd64"
            "darwin/arm64"
            "windows/amd64"
          )
          mkdir -p dist
          for TARGET in "${TARGETS[@]}"; do
            OS=${TARGET%/*}
            ARCH=${TARGET#*/}
            OUTPUT="dist/opus-${OS}-${ARCH}"
            if [ "$OS" = "windows" ]; then OUTPUT="${OUTPUT}.exe"; fi
            GOOS=$OS GOARCH=$ARCH go build -o $OUTPUT ./api/cmd/opus
          done

      - name: Create GitHub Release
        uses: softprops/action-gh-release@v2
        with:
          files: dist/*
          generate_release_notes: true

      - name: Setup Node.js
        uses: actions/setup-node@v4
        with:
          node-version: "22"
          registry-url: "https://registry.npmjs.org"

      - name: Publish npm Package
        run: npm publish
        working-directory: ./installer
        env:
          NODE_AUTH_TOKEN: ${{ secrets.NPM_TOKEN }}
```

### Constraints

- All workflow files must use the latest stable action versions (`@v4`, `@v5`).
- The `release.yml` must cross-compile for all five target platforms.
- Do not hardcode any secrets — use `${{ secrets.VARIABLE }}` syntax.

### Acceptance Criteria

- [x] File `.github/workflows/ci.yml` exists.
- [x] File `.github/workflows/build.yml` exists.
- [x] File `.github/workflows/release.yml` exists.
- [x] `ci.yml` runs on pull requests to `main`.
- [x] `ci.yml` includes Go setup, Node.js setup, pnpm setup, and Task setup steps.
- [x] `ci.yml` runs `task api:lint`, `task api:test:all`, `task dash:lint`, `task dash:test`.
- [x] `build.yml` runs on push to `main`.
- [x] `build.yml` builds and pushes Docker images to `ghcr.io/opus/`.
- [x] `release.yml` runs on version tag push.
- [x] `release.yml` cross-compiles for `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`, `windows/amd64`.
- [x] `release.yml` publishes npm package from `./installer` directory.
- [x] No secrets are hardcoded.

---

## P1-T7 — Write Root `README.md`

### What to Do

Create `opus/README.md` with project overview, quick start instructions for all three installation paths, and developer setup instructions.

### File to Create

`opus/README.md`

### Required Sections

1. **Project title and one-line description**
2. **Quick Start** — three paths:
   - `npx opus install` (end users)
   - Docker Compose (technical users)
   - Bare metal binary (manual)
3. **Developer Setup** — `task setup` and `task dev`
4. **CLI Commands** — table of all `opus` commands
5. **Configuration** — reference to `.env.example` and `~/.opus/config.toml`
6. **Architecture** — brief description with link to `docs/ARCHITECTURE.md`
7. **License**

### Constraints

- Use Markdown with clear headings.
- Include code blocks for all commands.
- Do not duplicate the full content of `ARCHITECTURE.md` or `PRD.md` — link to them instead.
- Keep it concise — this is the entry point, not the full documentation.

### Acceptance Criteria

- [x] File `opus/README.md` exists.
- [x] Contains project title and one-line description.
- [x] Contains Quick Start section with `npx opus install` instructions.
- [x] Contains Quick Start section with Docker Compose instructions.
- [x] Contains Quick Start section with bare metal binary instructions.
- [x] Contains Developer Setup section with `task setup` and `task dev`.
- [x] Contains CLI Commands table (`start`, `stop`, `restart`, `status`, `logs`).
- [x] Contains Configuration section referencing `.env.example`.
- [x] Contains link to `docs/ARCHITECTURE.md`.
- [x] All commands are in fenced code blocks.

---

## Phase 1 Completion Checklist

Before proceeding to Phase 2, verify:

- [ ] All 7 tasks (P1-T1 through P1-T7) are marked complete.
- [ ] Running `task --list` from `opus/` outputs all delegated tasks without errors.
- [ ] Running `docker compose config` validates `docker-compose.yml` without errors.
- [ ] Running `docker compose -f docker-compose.yml -f docker-compose.dev.yml config` validates without errors.
- [ ] `.github/workflows/` contains exactly three files: `ci.yml`, `build.yml`, `release.yml`.
- [ ] No `.env` file exists — only `.env.example`.
- [ ] No Go or TypeScript source files exist yet.
