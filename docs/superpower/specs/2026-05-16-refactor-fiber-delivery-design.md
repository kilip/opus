# Design: Refactor GoFiber Delivery

**Date:** 2026-05-16
**Topic:** Refactor GoFiber related code to `internal/delivery/fiber`
**Status:** Approved by Pak Bos

## Context
Currently, the API project has handlers and middlewares directly under `internal/handler` and `internal/middleware`. The server initialization and routing are mixed in `api/cmd/opus/start.go`. This violates strict Clean Architecture where delivery mechanisms should be isolated.

## Objectives
- Isolate Fiber-specific code into `internal/delivery/fiber`.
- Simplify server initialization in the CLI.
- Enable easier support for multiple delivery mechanisms in the future.

## Architecture
The new structure will follow the Delivery layer pattern:

```text
api/internal/delivery/fiber/
├── handler/           <-- (Moved from internal/handler)
├── middleware/        <-- (Moved from internal/middleware)
└── server.go          <-- (New: Server encapsulation)
```

### Server Struct
A new `Server` struct will manage the Fiber application lifecycle and dependency wiring.

```go
type Server struct {
    app         *fiber.App
    cfg         *config.Config
    authService service.AuthService
    userService service.UserService
    // other services as needed
}
```

### Key Methods
- `NewServer(...) *Server`: Initializes Fiber app, handlers, and registers routes.
- `Start() error`: Starts the server listener.
- `Stop() error`: Gracefully shuts down the server.

## Migration Plan
1. Create the new directory structure.
2. Move handlers and update their package names.
3. Move middlewares and update their package names.
4. Implement `server.go` with `setupRoutes` and `setupMiddleware` private methods.
5. Refactor `api/cmd/opus/start.go` to use the new `Server` struct.
6. Update all imports across the codebase.

## Success Criteria
- [ ] Code compiles and all tests pass.
- [ ] Server starts and responds to requests as before.
- [ ] No GoFiber imports remain in `api/cmd/opus/start.go` except for basic logging if needed.
- [ ] Handlers and middlewares are isolated in `internal/delivery/fiber`.
