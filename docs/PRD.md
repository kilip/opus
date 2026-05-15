# Product Requirements Document (PRD)

**Product:** Opus  
**Version:** 1.0.1  
**Status:** Draft  
**Last Updated:** 2026-05-15  
**Authors:** Product & Architecture Team

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Problem Statement](#2-problem-statement)
3. [Goals & Non-Goals](#3-goals--non-goals)
4. [Target Users](#4-target-users)
5. [User Stories](#5-user-stories)
6. [Functional Requirements](#6-functional-requirements)
7. [Non-Functional Requirements](#7-non-functional-requirements)
8. [Constraints & Assumptions](#8-constraints--assumptions)
9. [Success Metrics](#9-success-metrics)
10. [Out of Scope (v1.0)](#10-out-of-scope-v10)

---

## 1. Executive Summary

Opus is an autonomous AI agent designed for 24/7 personal assistance. It provides a self-hostable, privacy-first platform that runs on the user's own infrastructure — from a single Raspberry Pi to a production server — accessible via a Progressive Web App (PWA). Opus is designed to be installable in minutes, extensible by design, and operable without cloud dependencies.

The project is structured as a monorepo with two primary components: `api/` (Go backend) and `dashboard/` (Next.js frontend), orchestrated via a root-level Taskfile and distributed as a single installable unit.

---

## 2. Problem Statement

Existing AI assistant platforms are:

- **Cloud-locked:** User data resides on third-party servers with limited control.
- **Subscription-dependent:** Continuous cost with no self-hosted alternative.
- **Not autonomous:** Require active user initiation; no background task execution.
- **Hard to self-host:** Complex infrastructure requirements deter non-developer users.

Opus addresses all four problems by providing a lightweight, self-hosted AI agent with an opinionated, simple installation path and a clean web interface.

---

## 3. Goals & Non-Goals

### Goals

- Provide a fully self-hosted AI agent accessible via PWA.
- Support installation via `npx opus install` for end-users with minimal technical knowledge.
- Support Docker and bare metal deployment for technical users.
- Support multiple database backends (SQLite and PostgreSQL) — user's choice, not environment-dictated.
- Provide secure authentication via OAuth2 (Google, GitHub) and Email/Password (development only).
- Maintain a clean, extensible codebase using Clean Architecture principles.
- Provide a monorepo structure (`api/` + `dashboard/`) with a unified root Taskfile for developer orchestration.

### Non-Goals (v1.0)

- Multi-tenant SaaS deployment.
- Mobile native applications (iOS/Android).
- AI provider abstraction / multi-provider support (deferred to future versions).
- Billing or subscription management.

---

## 4. Target Users

| Persona | Description | Primary Install Path |
|---------|-------------|----------------------|
| **End User** | Non-technical individual wanting a personal AI assistant | `npx opus install` |
| **Developer** | Engineer who wants to self-host, extend, or contribute | Docker / bare metal |
| **Power User** | Technical-leaning user who self-hosts other services | Docker Compose |

---

## 5. User Stories

### Authentication

- As a user, I want to sign in with my Google account so that I do not need to manage a separate password.
- As a user, I want to sign in with my GitHub account as an alternative OAuth2 provider.
- As a developer, I want to sign in with email and password during local development so that I can test without OAuth2 configuration.
- As a user, I want my session to remain active across browser restarts without requiring re-authentication, via secure refresh token rotation.

### Installation & Setup

- As an end-user, I want to run a single command (`npx opus install`) to set up Opus interactively on my machine.
- As an end-user, I want the installer to configure auto-restart (systemd / launchd / Windows Service) so that Opus survives reboots.
- As a developer, I want to run Opus via Docker Compose with environment variable overrides.
- As a developer, I want to run Opus on bare metal using a pre-built binary.
- As a developer, I want to run a single command (`task setup`) to install all dependencies for both `api/` and `dashboard/` in one step.

### Configuration

- As a user, I want my configuration stored in `~/.opus/config.toml` so that it persists across updates.
- As a DevOps engineer, I want environment variables prefixed with `OPUS_` to always override file-based configuration in Docker deployments.

### CLI Operations

- As a user, I want to control the Opus service via CLI commands: `opus start`, `opus stop`, `opus restart`, `opus status`, `opus logs`.

### Core Agent Interface

- As a user, I want to interact with Opus via a clean PWA interface accessible from any device on my network.
- As a user, I want the PWA to be installable on my home screen (desktop or mobile).
- As a user, I want an offline fallback page when the server is unreachable.
- As a user, I want AI responses to stream in real time via Server-Sent Events (SSE).

---

## 6. Functional Requirements

### 6.1 Authentication & Session Management

| ID | Requirement |
|----|-------------|
| AUTH-01 | System shall support OAuth2 login via Google. |
| AUTH-02 | System shall support OAuth2 login via GitHub. |
| AUTH-03 | System shall support Email/Password login in `development` mode only. |
| AUTH-04 | System shall issue signed JWT access tokens upon successful authentication. |
| AUTH-05 | System shall implement refresh token rotation: each use of a refresh token issues a new one and invalidates the previous. |
| AUTH-06 | System shall store refresh tokens securely (hashed) in the database. |
| AUTH-07 | System shall support explicit logout, invalidating the current refresh token. |

### 6.2 Configuration

| ID | Requirement |
|----|-------------|
| CFG-01 | System shall load configuration from `~/.opus/config.toml`. |
| CFG-02 | Environment variables prefixed with `OPUS_` shall override all file-based configuration values. |
| CFG-03 | System shall fall back to sane defaults for all non-critical configuration values. |
| CFG-04 | Configuration hierarchy: `ENV (OPUS_*)` > `~/.opus/config.toml` > `defaults`. |

### 6.3 Database

| ID | Requirement |
|----|-------------|
| DB-01 | System shall support SQLite as a database backend. |
| DB-02 | System shall support PostgreSQL as a database backend. |
| DB-03 | The database driver shall be user-configurable via `config.toml` or `OPUS_DATABASE_DRIVER`. |
| DB-04 | Database schema shall be managed via EntGo migrations. |
| DB-05 | Both backends shall be treated as first-class; neither is restricted to a specific environment. |

### 6.4 Installation & Distribution

| ID | Requirement |
|----|-------------|
| DIST-01 | `npx opus install` shall provide an interactive CLI wizard for first-time setup. |
| DIST-02 | The installer shall download the appropriate pre-built Go binary from GitHub Releases. |
| DIST-03 | The installer shall generate `~/.opus/config.toml` based on user inputs. |
| DIST-04 | The installer shall register Opus as a system service (systemd on Linux, launchd on macOS, Windows Service on Windows). |
| DIST-05 | Docker image shall be provided with documented environment variable configuration. |
| DIST-06 | Bare metal binary shall be available for manual installation. |
| DIST-07 | A root-level `docker-compose.yml` shall orchestrate both `api/` and `dashboard/` services. |

### 6.5 CLI

| ID | Requirement |
|----|-------------|
| CLI-01 | `opus start` — starts the Opus service. |
| CLI-02 | `opus stop` — stops the Opus service. |
| CLI-03 | `opus restart` — restarts the Opus service. |
| CLI-04 | `opus status` — displays current service status. |
| CLI-05 | `opus logs` — tails the service log output. |

### 6.6 PWA & Frontend

| ID | Requirement |
|----|-------------|
| FE-01 | Frontend shall be a Next.js 16 application located at `dashboard/`. |
| FE-02 | Frontend shall be installable as a PWA on desktop and mobile. |
| FE-03 | Frontend shall display an offline fallback page when the server is unreachable. |
| FE-04 | Frontend shall consume the API via TanStack Query for server state management. |
| FE-05 | Frontend shall support real-time AI response streaming via SSE. |
| FE-06 | Service Worker shall be managed via Serwist. |

### 6.7 Developer Experience

| ID | Requirement |
|----|-------------|
| DX-01 | Root `Taskfile.yml` shall act as an orchestrator, delegating tasks to `api/Taskfile.yml` and `dashboard/Taskfile.yml`. |
| DX-02 | `task setup` at root level shall install all dependencies for both `api/` and `dashboard/`. |
| DX-03 | `task dev` at root level shall start both `api/` and `dashboard/` in development mode concurrently. |
| DX-04 | `task build` at root level shall build both components. |
| DX-05 | `task test:all` at root level shall execute all unit and integration tests across both components. |
| DX-06 | Root `.env.example` shall document all `OPUS_*` environment variables. |
| DX-07 | CI/CD pipelines shall be defined under `.github/workflows/` covering test, build, and release stages. |

---

## 7. Non-Functional Requirements

| Category | Requirement |
|----------|-------------|
| **Performance** | API response time (non-streaming) shall be under 200ms for 95th percentile on localhost. |
| **Security** | All tokens shall be signed with HS256 or RS256. Refresh tokens shall be hashed before storage. |
| **Reliability** | Service shall auto-restart on crash via the registered system service manager. |
| **Portability** | Go binary shall compile for linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64. |
| **Maintainability** | Codebase shall follow Clean Architecture: `model → service → repository → handler`. |
| **Testability** | API shall achieve unit test coverage via uber/mock and integration test coverage via SQLite in-memory. |
| **Observability** | All structured logs shall use `slog` with JSON output in production and text output in development. |
| **Developer Experience** | A developer shall be able to run the full stack locally with a single `task dev` command from the monorepo root. |

---

## 8. Constraints & Assumptions

- Opus v1.0 is a **single-user** system. Multi-user support is deferred.
- Email/Password authentication is explicitly **development-only** and disabled in production builds.
- The AI provider integration is **out of scope** for base structure; the agent interface will be defined in a subsequent iteration.
- SQLite is a fully supported production database for lightweight, single-user deployments.
- The `npx opus install` wizard targets end-users with Node.js installed (LTS).
- The monorepo structure (`api/` + `dashboard/`) is housed within a single root `opus/` directory.

---

## 9. Success Metrics

| Metric | Target |
|--------|--------|
| Time to first working installation (npx path) | < 5 minutes |
| Time to first working installation (Docker path) | < 2 minutes |
| Time to full local dev environment setup (`task setup` + `task dev`) | < 3 minutes |
| API unit test coverage | ≥ 80% |
| Frontend E2E test coverage (critical flows) | Auth, Dashboard, Streaming |
| PWA Lighthouse score | ≥ 90 |

---

## 10. Out of Scope (v1.0)

- Multi-user / multi-tenant support.
- AI provider configuration and agent capability definition.
- Push notifications (PWA — deferred to v1.1).
- Mobile native applications.
- Plugin / extension system.
- Billing, quota management, or rate limiting per user.
- `pkg/` shared utilities module (reserved; documented in ARCHITECTURE.md).