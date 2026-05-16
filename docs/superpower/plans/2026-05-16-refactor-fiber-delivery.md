# Refactor GoFiber Delivery Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Refactor GoFiber related code to `internal/delivery/fiber` to isolate delivery logic.

**Architecture:** Move handlers and middlewares to `internal/delivery/fiber/handler` and `internal/delivery/fiber/middleware` respectively. Implement a `Server` struct in `internal/delivery/fiber/server.go` to encapsulate Fiber setup.

**Tech Stack:** Go, GoFiber v3.

---

### Task 1: Setup Directory Structure

**Files:**
- Create: `api/internal/delivery/fiber/handler/.gitkeep`
- Create: `api/internal/delivery/fiber/middleware/.gitkeep`

- [ ] **Step 1: Create directories**
```bash
mkdir -p api/internal/delivery/fiber/handler
mkdir -p api/internal/delivery/fiber/middleware
touch api/internal/delivery/fiber/handler/.gitkeep
touch api/internal/delivery/fiber/middleware/.gitkeep
```

- [ ] **Step 2: Commit**
```bash
git add api/internal/delivery/fiber
git commit -m "chore: create internal/delivery/fiber structure"
```

### Task 2: Move Handlers and Update Package Names

**Files:**
- Modify: All files in `api/internal/handler` to change package to `handler` (but they will be in `internal/delivery/fiber/handler`)
- Move: `api/internal/handler/*.go` -> `api/internal/delivery/fiber/handler/`

- [ ] **Step 1: Move handler files**
```bash
mv api/internal/handler/*.go api/internal/delivery/fiber/handler/
```

- [ ] **Step 2: Update package names in moved handlers**
Change `package handler` (if it was `package handler`) to `package handler`. Wait, the package name should match the directory name. Since they are in `internal/delivery/fiber/handler`, the package name remains `handler`. However, the import paths will change.

- [ ] **Step 3: Commit**
```bash
git add api/internal/delivery/fiber/handler
git commit -m "refactor: move handlers to internal/delivery/fiber/handler"
```

### Task 3: Move Middlewares and Update Package Names

**Files:**
- Move: `api/internal/middleware/*.go` -> `api/internal/delivery/fiber/middleware/`

- [ ] **Step 1: Move middleware files**
```bash
mv api/internal/middleware/*.go api/internal/delivery/fiber/middleware/
```

- [ ] **Step 2: Commit**
```bash
git add api/internal/delivery/fiber/middleware
git commit -m "refactor: move middlewares to internal/delivery/fiber/middleware"
```

### Task 4: Implement `internal/delivery/fiber/server.go`

**Files:**
- Create: `api/internal/delivery/fiber/server.go`

- [ ] **Step 1: Write `server.go` implementation**

```go
package fiber

import (
	"fmt"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/opus/api/internal/config"
	"github.com/kilip/opus/api/internal/delivery/fiber/handler"
	"github.com/kilip/opus/api/internal/delivery/fiber/middleware"
	"github.com/kilip/opus/api/internal/service"
)

type Server struct {
	app         *fiber.App
	cfg         *config.Config
	authService service.AuthService
	userService service.UserService
}

func NewServer(cfg *config.Config, authSvc service.AuthService, userSvc service.UserService) *Server {
	app := fiber.New(fiber.Config{
		AppName:      "Opus API",
		ErrorHandler: middleware.ErrorHandler,
	})

	s := &Server{
		app:         app,
		cfg:         cfg,
		authService: authSvc,
		userService: userSvc,
	}

	s.setupMiddleware()
	s.setupRoutes()

	return s
}

func (s *Server) setupMiddleware() {
	s.app.Use(middleware.Logger())
	s.app.Use(middleware.Recovery())
}

func (s *Server) setupRoutes() {
	authHandler := handler.NewAuthHandler(s.authService, s.userService, s.cfg)
	userHandler := handler.NewUserHandler(s.userService)
	healthHandler := handler.NewHealthHandler()
	sseHandler := handler.NewSSEHandler()

	v1 := s.app.Group("/api/v1")

	// Public Routes
	v1.Get("/health", healthHandler.Check)

	auth := v1.Group("/auth")
	auth.Post("/login", authHandler.Login)
	auth.Post("/refresh", authHandler.Refresh)
	auth.Post("/logout", authHandler.Logout)
	auth.Get("/google", authHandler.GoogleLogin)
	auth.Get("/google/callback", authHandler.GoogleCallback)

	// Protected Routes
	v1.Get("/user/me", userHandler.Me, middleware.Auth(s.authService))
	v1.Get("/stream", sseHandler.Stream, middleware.Auth(s.authService))
}

func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.cfg.Server.Host, s.cfg.Server.Port)
	return s.app.Listen(addr)
}

func (s *Server) Stop() error {
	return s.app.Shutdown()
}
```

- [ ] **Step 2: Commit**
```bash
git add api/internal/delivery/fiber/server.go
git commit -m "feat: implement fiber server encapsulation"
```

### Task 5: Refactor `api/cmd/opus/start.go`

**Files:**
- Modify: `api/cmd/opus/start.go`

- [ ] **Step 1: Update `start.go` to use the new Server struct**

```go
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/kilip/opus/api/internal/config"
	"github.com/kilip/opus/api/internal/delivery/fiber"
	"github.com/kilip/opus/api/internal/repository"
	"github.com/kilip/opus/api/internal/service"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the API server",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.GetConfig()
		db := config.GetDatabase()

		// Ensure .opus directory exists
		opusDir := filepath.Join(os.Getenv("HOME"), ".opus")
		_ = os.MkdirAll(opusDir, 0755)

		// Save PID
		pidFile := filepath.Join(opusDir, "opus.pid")
		_ = os.WriteFile(pidFile, []byte(fmt.Sprintf("%d", os.Getpid())), 0644)

		// Repositories
		userRepo := repository.NewUserRepository(db)
		sessionRepo := repository.NewSessionRepository(db)

		// Services
		authService := service.NewAuthService(userRepo, sessionRepo, cfg)
		userService := service.NewUserService(userRepo, cfg)

		// Initialize and start Server
		srv := fiber.NewServer(cfg, authService, userService)
		
		log.Printf("Starting server on %s:%d", cfg.Server.Host, cfg.Server.Port)
		if err := srv.Start(); err != nil {
			log.Fatalf("Server failed to start: %v", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
}
```

- [ ] **Step 2: Commit**
```bash
git add api/cmd/opus/start.go
git commit -m "refactor: use fiber.Server in start command"
```

### Task 6: Final Verification

- [ ] **Step 1: Run all tests**
```bash
task test:all
```

- [ ] **Step 2: Remove old directories if empty**
```bash
rmdir api/internal/handler
rmdir api/internal/middleware
```

- [ ] **Step 3: Commit**
```bash
git commit -m "chore: cleanup old delivery directories"
```
