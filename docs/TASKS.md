# Implementation Tasks — Master Index

**Product:** Opus  
**Version:** 1.0.1  
**Status:** Completed  
**Last Updated:** 2026-05-15  
**Authors:** Product & Architecture Team

---

## How to Use This Document

This file is the **master index and progress tracker** for the Opus v1.0.1 implementation. Each phase has a dedicated file with detailed, spoon-fed tasks for your AI agent.

### Session Setup Checklist

Before starting any session with your AI agent (Gemini), always provide:

1. `docs/CONVENTIONS.md` — coding standards and rules
2. `docs/PRD.md` — product requirements
3. `docs/ARCHITECTURE.md` — architecture reference
4. The relevant `docs/PHASE-X.md` — current phase tasks

### Workflow Per Task

```
1. Open the relevant PHASE-X.md
2. Read the task to Gemini with CONVENTIONS.md as context
3. Review output against the acceptance criteria checklist
4. Mark the task as done [ ] → [x]
5. Proceed to the next task
```

---

## Phase Overview

| Phase | File | Description | Status |
|-------|------|-------------|--------|
| Phase 1 | `PHASE-1.md` | Monorepo Setup & Tooling | ✅ Completed |
| Phase 2 | `PHASE-2.md` | API Foundation (Go Backend) | ✅ Completed |
| Phase 3 | `PHASE-3.md` | Frontend Foundation (Next.js) | ✅ Completed |
| Phase 4 | `PHASE-4.md` | Integration & Deployment | ✅ Completed |

---

## Phase 1 — Monorepo Setup & Tooling

> **Goal:** Establish the monorepo structure, task automation, Docker configuration, and CI/CD scaffolding. No business logic. Zero to runnable skeleton.

| ID | Task | Status |
|----|------|--------|
| P1-T1 | Initialize monorepo directory structure | [x] |
| P1-T2 | Configure root Taskfile.yml | [x] |
| P1-T3 | Configure root `.env.example` | [x] |
| P1-T4 | Configure `docker-compose.yml` (production) | [x] |
| P1-T5 | Configure `docker-compose.dev.yml` (development) | [x] |
| P1-T6 | Scaffold GitHub Actions workflows | [x] |
| P1-T7 | Write root `README.md` | [x] |

---

## Phase 2 — API Foundation (Go Backend)

> **Goal:** Implement the full Go backend: configuration, database, authentication (OAuth2 + JWT), session management, CLI commands, and all API endpoints as defined in ARCHITECTURE.md.

| ID | Task | Status |
|----|------|--------|
| P2-T1 | Initialize Go module and install dependencies | [x] |
| P2-T2 | Implement configuration system (`internal/config/`) | [x] |
| P2-T3 | Define EntGo schemas (`User`, `Session`) | [x] |
| P2-T4 | Implement repository layer (`user`, `session`) | [x] |
| P2-T5 | Implement auth service (`internal/service/auth.go`) | [x] |
| P2-T6 | Implement user service (`internal/service/user.go`) | [x] |
| P2-T7 | Implement middleware (`auth`, `logger`, `recovery`) | [x] |
| P2-T8 | Implement auth handlers (OAuth2 + Email/Password + Refresh + Logout) | [x] |
| P2-T9 | Implement user handler (`/user/me`) | [x] |
| P2-T10 | Implement health check handler | [x] |
| P2-T11 | Implement SSE handler (`/stream`) | [x] |
| P2-T12 | Implement Cobra CLI commands | [x] |
| P2-T13 | Configure `api/Taskfile.yml` | [x] |
| P2-T14 | Write unit tests for auth service | [x] |
| P2-T15 | Write integration tests for repository layer | [x] |

---

## Phase 3 — Frontend Foundation (Next.js)

> **Goal:** Implement the Next.js 16 PWA: routing, authentication UI, dash shell, SSE streaming output, offline fallback, and service worker configuration.

| ID | Task | Status |
|----|------|--------|
| P3-T1 | Initialize Next.js 16 project with TypeScript and pnpm | [x] |
| P3-T2 | Install and configure Tailwind CSS v4, Shadcn/ui | [x] |
| P3-T3 | Install and configure TanStack Query | [x] |
| P3-T4 | Install and configure Serwist (PWA / Service Worker) | [x] |
| P3-T5 | Implement API base client and response types (`lib/api/`) | [x] |
| P3-T6 | Implement auth query hooks (`lib/api/auth.ts`) | [x] |
| P3-T7 | Implement user query hooks (`lib/api/user.ts`) | [x] |
| P3-T8 | Implement `useStream` hook (`lib/api/useStream.ts`) | [x] |
| P3-T9 | Implement root layout and global styles | [x] |
| P3-T10 | Implement auth layout and login page | [x] |
| P3-T11 | Implement `AuthGuard` component | [x] |
| P3-T12 | Implement dash layout and main dash page | [x] |
| P3-T13 | Implement `StreamOutput` component | [x] |
| P3-T14 | Implement offline fallback page | [x] |
| P3-T15 | Configure PWA manifest (`manifest.webmanifest`) | [x] |
| P3-T16 | Configure `dash/Taskfile.yml` | [x] |
| P3-T17 | Write Vitest unit tests (hooks + utilities) | [x] |
| P3-T18 | Write Playwright E2E tests (auth, dash, streaming, PWA) | [x] |

---

## Phase 4 — Integration & Deployment

> **Goal:** Wire all components together, validate end-to-end flows, build the `npx get-opus` wizard, and finalize CI/CD release pipelines.

| ID | Task | Status |
|----|------|--------|
| P4-T1 | Validate end-to-end auth flow (OAuth2 → JWT → dash) | [x] |
| P4-T2 | Validate SSE streaming end-to-end | [x] |
| P4-T3 | Implement `npx get-opus` wizard (Node.js installer package) | [x] |
| P4-T4 | Implement `api/Dockerfile` and `dash/Dockerfile` | [x] |
| P4-T5 | Finalize `docker-compose.yml` and `docker-compose.dev.yml` | [x] |
| P4-T6 | Configure systemd / launchd / Windows Service registration | [x] |
| P4-T7 | Finalize GitHub Actions `ci.yml` | [x] |
| P4-T8 | Finalize GitHub Actions `build.yml` | [x] |
| P4-T9 | Finalize GitHub Actions `release.yml` | [x] |
| P4-T10 | Lighthouse PWA audit (target score ≥ 90) | [x] |
| P4-T11 | Final cross-platform binary build validation | [x] |

---

## Progress Summary

| Phase | Total Tasks | Done | Remaining |
|-------|------------|------|-----------|
| Phase 1 | 7 | 7 | 0 |
| Phase 2 | 15 | 15 | 0 |
| Phase 3 | 18 | 18 | 0 |
| Phase 4 | 11 | 11 | 0 |
| **Total** | **51** | **51** | **0** |

---

## Reference Documents

| Document | Purpose |
|----------|---------|
| `docs/CONVENTIONS.md` | Coding conventions — provide to AI agent every session |
| `docs/PRD.md` | Product requirements |
| `docs/ARCHITECTURE.md` | Architecture reference |
| `docs/PHASE-1.md` | Phase 1 detailed tasks |
| `docs/PHASE-2.md` | Phase 2 detailed tasks |
| `docs/PHASE-3.md` | Phase 3 detailed tasks |
| `docs/PHASE-4.md` | Phase 4 detailed tasks |
