# WhatsApp Service Logic Design (Task 8)

**Topic:** Implementation of the "brain" for multi-user WhatsApp integration.
**Date:** 2026-05-16
**Status:** Validated (Brainstorming Complete)

## 1. Goal
Implement the core logic of `WhatsAppService` to manage multiple concurrent WhatsApp sessions, handle real-time events, and provide a configurable queueing mechanism for data persistence.

## 2. Architecture: Event-Driven & Scalable
The system follows a producer-consumer pattern to ensure stability under high load (multi-user).

- **`WhatsAppService`**: The orchestrator. Manages a `sync.Map` of `whatsmeow.Client` instances. Responsible for parallel initialization of all "CONNECTED" sessions upon server startup.
- **`WhatsAppClient` Wrapper**: A domain-specific wrapper around `whatsmeow.Client` to manage user-specific state and event registration.
- **`EventHandler`**: Listens to raw `whatsmeow` events (Message, HistorySync, QRUpdate) and transforms them into domain-specific "Tasks".
- **`Queue` Interface**: An abstraction for task dispatching.
  - Driver: Configurable (In-Memory / Database / Redis).
  - Implementation of specific drivers is out of scope for the initial Task 8 but the interface is mandatory.

## 3. Data Flow
1. **Connection**: User calls `/whatsapp/connect` -> Service creates client -> QR generated -> SSE `wa_qr_update`.
2. **Event Reception**: WhatsApp sends an event (e.g., `events.Message`) -> Client's internal listener catches it.
3. **Task Dispatching**: `EventHandler` wraps the event into a `Task` -> Calls `Queue.Push(task)`.
4. **Consumption (Next Phase)**: Background workers pick tasks from the queue and upsert into EntGo schemas (`WaMessage`, `WaChat`, etc.).

## 4. Key Components (API Layer)

### 4.1. `WhatsAppService` Interface (Refinement)
```go
type WhatsAppService interface {
    // Lifecycle
    Initialize(ctx context.Context) error // Parallel boot of all CONNECTED users
    Connect(ctx context.Context, userID string) error
    Disconnect(ctx context.Context, userID string) error
    
    // Status & Messaging
    GetStatus(ctx context.Context, userID string) (status string, jid string, err error)
    SendMessage(ctx context.Context, userID string, targetJID string, message string) error
}
```

### 4.2. Configuration Structure
```toml
[whatsapp]
queue_driver = "memory" # options: "memory", "database", "redis"
parallel_init_limit = 10 # limit number of concurrent connection attempts at boot
```

## 5. Implementation Strategy
- **Parallel Boot**: Use `errgroup` or simple goroutines with a semaphore to limit initial burst if needed, but prioritize parallel execution as requested.
- **Thread Safety**: All access to the active client map MUST be guarded by `sync.RWMutex`.
- **SSE Integration**: Use the existing `internal/delivery/fiber/handler/stream.go` (or equivalent) to broadcast events using user-specific topics.

## 6. Testing Strategy
- **Unit Tests**: Mock `whatsmeow.Client` and `Queue` interface to test logic in `WhatsAppService`.
- **Integration Tests**: Verify parallel connection lifecycle using an in-memory SQLStore for `whatsmeow`.

## 7. Next Steps
1. Approve this spec.
2. Create detailed implementation plan (Task 9).
3. Implement `WhatsAppService` implementation in `api/internal/service/whatsapp_impl.go`.
