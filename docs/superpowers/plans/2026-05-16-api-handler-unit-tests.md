# API Handler Unit Testing Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement comprehensive unit tests for all API handlers (`AuthHandler`, `HealthHandler`, `SSEHandler`, `UserHandler`).

**Architecture:**
1.  Introduce interfaces for `AuthService` and `UserService` in `internal/service/` to allow mocking in handler tests.
2.  Refactor handlers in `internal/handler/` to depend on these interfaces.
3.  Use `gomock` and `testify` to write unit tests for each handler in `internal/handler/`.

**Tech Stack:** Go, `testify`, `gomock`, `fiber/v3`

---

### Task 1: Refactor Services for Mockability

- [ ] **Step 1: Define `AuthServiceInterface` in `api/internal/service/auth.go`**
- [ ] **Step 2: Update `AuthHandler` constructor to accept `AuthServiceInterface`**
- [ ] **Step 3: Define `UserServiceInterface` in `api/internal/service/user.go`**
- [ ] **Step 4: Update `UserHandler` constructor to accept `UserServiceInterface`**

### Task 2: Implement Handler Unit Tests

- [ ] **Step 1: Implement `api/internal/handler/health_test.go`** (No dependencies)
- [ ] **Step 2: Implement `api/internal/handler/user_test.go`** (Mock `UserService`)
- [ ] **Step 3: Implement `api/internal/handler/auth_test.go`** (Mock `AuthService` and `UserService`)
- [ ] **Step 4: Implement `api/internal/handler/sse_test.go`**

---
