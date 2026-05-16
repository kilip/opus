# WhatsApp Service Implementation Plan (Rigorous Edition)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement a production-grade, multi-user WhatsApp integration using `whatsmeow` with real-time SSE updates and background synchronization via the Opus Queue system.

**Architecture:** Hybrid synchronization model using user-specific SSE channels for real-time events and a modular Queue system for heavy data sync. Implements a dual-DB strategy to isolate cryptographic session storage from domain data.

**Tech Stack:** Go, whatsmeow, EntGo, Opus Queue System, SSE, Next.js 16.

---

### Task 1: Setup Dependencies and Dedicated Store

**Files:**
- Modify: `api/go.mod`
- Create: `api/internal/config/whatsapp.go`

- [x] **Step 1: Install whatsmeow**
Run: `go get go.mau.fi/whatsmeow@latest`
Expected: `go.mod` and `go.sum` updated.

- [x] **Step 2: Implement WhatsApp sqlstore factory**
```go
package config

import (
	"fmt"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"
	_ "github.com/mattn/go-sqlite3"
	_ "github.com/lib/pq"
)

func GetWhatsAppStore(cfg *Config) (*sqlstore.Container, error) {
	var driver, dsn string
	switch cfg.Database.Driver {
	case "sqlite":
		driver = "sqlite3"
		dsn = cfg.Database.DSN + "?_fk=1"
	case "postgres":
		driver = "postgres"
		dsn = cfg.Database.DSN
	default:
		return nil, fmt.Errorf("unsupported database driver for whatsapp store: %s", cfg.Database.Driver)
	}
	return sqlstore.New(driver, dsn, waLog.Stdout("WA-Store", "WARN", true))
}
```

- [x] **Step 3: Commit**
```bash
git add api/go.mod api/go.sum api/internal/config/whatsapp.go
git commit -m "feat(api): add whatsmeow and dedicated sqlstore config"
```

---

### Task 2: User-Isolated SSE Hub (TDD)

**Files:**
- Modify: `api/internal/service/sse.go`
- Test: `api/internal/service/sse_test.go`

- [x] **Step 1: Define User-Specific Interface**
```go
package service

type SSEEvent struct {
	Type    string `json:"type"`
	Payload any    `json:"payload"`
}

type SSEHub interface {
	Publish(userID string, event SSEEvent) error
	Broadcast(event SSEEvent) error
}
```

- [x] **Step 2: Write failing test for Publish(userID, ...)**
(Assuming existing hub logic uses a map of user channels)
```go
func TestSSEHub_PublishToUser(t *testing.T) {
    hub := NewSSEHub()
    err := hub.Publish("user-123", SSEEvent{Type: "test", Payload: "hi"})
    assert.Error(t, err) // Should fail if no client connected for this user
}
```

- [x] **Step 3: Implement minimal logic in `api/internal/delivery/fiber/middleware/sse.go`**
Update the `Hub` struct to maintain a map of `userID -> []chan SSEEvent`.

- [x] **Step 4: Run test and verify PASS**

- [x] **Step 5: Commit**
```bash
git add api/internal/service/sse.go api/internal/delivery/fiber/middleware/sse.go
git commit -m "feat(api): implement user-isolated SSE publishing"
```

---

### Task 3: WhatsAppRepository Extension (TDD)

**Files:**
- Modify: `api/internal/repository/whatsapp.go`
- Test: `api/internal/repository/whatsapp_integration_test.go`

- [x] **Step 1: Add `GetAllActiveSessions` to interface**
```go
type WhatsAppRepository interface {
    // ... existing
    GetAllActiveSessions(ctx context.Context) ([]*ent.WaSession, error)
    UpdateStatus(ctx context.Context, userID string, status string, jid string) error
}
```

- [x] **Step 2: Write failing integration test**
```go
func TestRepo_GetAllActiveSessions(t *testing.T) {
    repo := NewWhatsAppRepository(client)
    repo.UpsertSession(ctx, "u1", "CONNECTED", "jid1")
    sessions, _ := repo.GetAllActiveSessions(ctx)
    assert.Len(t, sessions, 1)
}
```

- [x] **Step 3: Implement minimal logic**
```go
func (r *whatsappRepo) GetAllActiveSessions(ctx context.Context) ([]*ent.WaSession, error) {
    return r.client.WaSession.Query().Where(wasession.StatusEQ("CONNECTED")).All(ctx)
}
```

- [x] **Step 4: Run integration test and verify PASS**

- [x] **Step 5: Commit**
```bash
git add api/internal/repository/whatsapp.go
git commit -m "feat(api): add session recovery methods to WhatsAppRepository"
```

---

### Task 4: WhatsAppService Foundation & Connect (TDD)

**Files:**
- Modify: `api/internal/service/whatsapp.go`
- Create: `api/internal/service/whatsapp_impl.go`
- Test: `api/internal/service/whatsapp_test.go`

- [x] **Step 1: Define Interface and Implementation Struct**
```go
type WhatsAppService interface {
    Initialize(ctx context.Context) error
    Connect(ctx context.Context, userID string) error
    Disconnect(ctx context.Context, userID string) error
    ForceReconnect(ctx context.Context, userID string) error
    GetStatus(ctx context.Context, userID string) (status string, jid string, err error)
    SendMessage(ctx context.Context, userID string, targetJID string, message string) error
}

type whatsAppService struct {
    clients    sync.Map // map[string]*whatsmeow.Client
    retryState sync.Map // map[string]*retryEntry
    repo       repository.WhatsAppRepository
    sseHub     SSEHub
    queue      Queue
    db         *ent.Client
    store      *sqlstore.Container
    log        *slog.Logger
}
```

- [x] **Step 2: Write failing test for Connect() QR dispatch**
```go
func TestConnect_DispatchesQR(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()
    mockHub := mocks.NewMockSSEHub(ctrl)
    svc := &whatsAppService{sseHub: mockHub, log: slog.Default()}
    mockHub.EXPECT().Publish("user-1", gomock.MatchedBy(func(e SSEEvent) bool {
        return e.Type == "wa_qr_update"
    })).Return(nil).AnyTimes()
    svc.Connect(context.Background(), "user-1")
}
```

- [x] **Step 3: Implement minimal `Connect` logic**
```go
func (s *whatsAppService) Connect(ctx context.Context, userID string) error {
	device, _ := s.store.GetFirstDevice()
	client := whatsmeow.NewClient(device, nil)
	s.clients.Store(userID, client)
	qrChan, _ := client.GetQRChannel(ctx)
	go func() {
		for evt := range qrChan {
			if evt.Event == "code" {
				s.sseHub.Publish(userID, SSEEvent{Type: "wa_qr_update", Payload: map[string]string{"qr": evt.Code}})
			}
		}
	}()
	return client.Connect()
}
```

- [x] **Step 4: Run test and verify PASS**

- [x] **Step 5: Commit**
```bash
git add api/internal/service/whatsapp*
git commit -m "feat(api): implement Connect logic and QR event dispatch"
```

---

### Task 5: Event Handlers and Message Sync (TDD)

**Files:**
- Modify: `api/internal/service/whatsapp_impl.go`
- Test: `api/internal/service/whatsapp_test.go`

- [ ] **Step 1: Write test for `handleNewMessage`**
```go
func TestHandleNewMessage_SavesToEntGo(t *testing.T) {
    // Mock EntGo client and repository
    // Verify WaMessage.Create() is called with correct content
}
```

- [ ] **Step 2: Implement Handler Registration**
```go
func (s *whatsAppService) registerHandler(userID string, client *whatsmeow.Client) {
	client.AddEventHandler(func(evt interface{}) {
		switch v := evt.(type) {
		case *events.Connected:
			s.repo.UpdateStatus(context.Background(), userID, "CONNECTED", client.Store.ID.String())
			s.sseHub.Publish(userID, SSEEvent{Type: "wa_connected", Payload: map[string]string{"jid": client.Store.ID.String()}})
		case *events.Message:
			s.handleNewMessage(userID, v)
		case *events.HistorySync:
			s.handleHistorySync(userID, v)
		}
	})
}
```

- [ ] **Step 3: Implement `handleNewMessage` Logic**
```go
func (s *whatsAppService) handleNewMessage(userID string, evt *events.Message) {
	content := ""
	if evt.Message.GetConversation() != "" {
		content = evt.Message.GetConversation()
	}
	sess, _ := s.repo.GetSessionByUserID(context.Background(), userID)
	s.db.WaMessage.Create().
		SetMessageID(evt.Info.ID).
		SetSenderJid(evt.Info.Sender.String()).
		SetContent(content).
		SetTimestamp(evt.Info.Timestamp).
		SetIsFromMe(evt.Info.IsFromMe).
		SetWaSession(sess).
		Save(context.Background())
}
```

- [ ] **Step 4: Run test and verify PASS**

- [ ] **Step 5: Commit**
```bash
git add api/internal/service/whatsapp_impl.go
git commit -m "feat(api): implement message handling and SSE notifications"
```

---

### Task 6: History Synchronization Worker

**Files:**
- Create: `api/internal/worker/whatsapp_sync.go`
- Modify: `api/internal/service/whatsapp_impl.go`

- [ ] **Step 1: Define `SyncHistoryPayload` and Job Handler**
```go
type SyncHistoryPayload struct {
    UserID   string `json:"user_id"`
    Messages []struct {
        ID      string    `json:"id"`
        Content string    `json:"content"`
        TS      time.Time `json:"ts"`
    } `json:"messages"`
}
```

- [ ] **Step 2: Implement Worker logic**
Batch upsert `WaContact`, `WaChat`, and `WaMessage` using `s.db.WaContact.CreateBulk(...)`.

- [ ] **Step 3: Connect Service to Queue**
```go
func (s *whatsAppService) handleHistorySync(userID string, evt *events.HistorySync) {
    // Map evt to payload...
    s.queue.Push(context.Background(), Job{Type: "whatsapp.sync_history", Payload: payload})
}
```

- [ ] **Step 4: Commit**
```bash
git add api/internal/worker/whatsapp_sync.go
git commit -m "feat(api): implement background history sync worker"
```

---

### Task 5: ForceReconnect & Initialization

**Files:**
- Modify: `api/internal/service/whatsapp_impl.go`
- Modify: `api/cmd/opus/start.go`

- [ ] **Step 1: Implement `Initialize` logic**
```go
func (s *whatsAppService) Initialize(ctx context.Context) error {
    sessions, _ := s.repo.GetAllActiveSessions(ctx)
    for _, sess := range sessions {
        go s.Connect(ctx, sess.UserID)
    }
    return nil
}
```

- [ ] **Step 2: Implement `ForceReconnect`**
```go
func (s *whatsAppService) ForceReconnect(ctx context.Context, userID string) error {
	if client, ok := s.clients.Load(userID); ok {
		client.(*whatsmeow.Client).Disconnect()
	}
	s.retryState.Delete(userID)
	return s.Connect(ctx, userID)
}
```

- [ ] **Step 3: Register in server boot**
In `api/cmd/opus/start.go`: `waService.Initialize(context.Background())`

- [ ] **Step 4: Commit**
```bash
git add api/internal/service/whatsapp_impl.go api/cmd/opus/start.go
git commit -m "feat(api): implement service initialization and force reconnect"
```

---

### Task 6: API Handlers and Routes

**Files:**
- Modify: `api/internal/delivery/fiber/handler/whatsapp.go`

- [ ] **Step 1: Add Reconnect Endpoint**
```go
func (h *WhatsAppHandler) ForceReconnect(c fiber.Ctx) error {
    userID := c.Locals("user_id").(string)
    h.service.ForceReconnect(c.Context(), userID)
    return c.Status(200).JSON(fiber.Map{"status": "initiated"})
}
```

- [ ] **Step 2: Register Routes**
`group.Post("/reconnect", h.ForceReconnect)`

- [ ] **Step 3: Commit**
```bash
git add api/internal/delivery/fiber/handler/whatsapp.go
git commit -m "feat(api): expose WhatsApp reconnect endpoint"
```

---

### Task 7: Dashboard UI (Status & Reconnect)

**Files:**
- Modify: `dash/hooks/use-whatsapp.ts`
- Modify: `dash/components/whatsapp/status-card.tsx`

- [ ] **Step 1: Add SSE Listeners to hook**
Listen for `wa_connected`, `wa_qr_update`, etc.

- [ ] **Step 2: Implement Reconnect action**
`await api.post("/whatsapp/reconnect")`

- [ ] **Step 3: Commit**
```bash
git add dash/hooks/use-whatsapp.ts dash/components/whatsapp/status-card.tsx
git commit -m "feat(dash): integrate real-time status and reconnect button"
```

---

### Task 8: Verification

- [ ] **Step 1: Run all tests**
Run: `task test:all`
Expected: ALL PASS

- [ ] **Step 2: Final Manual Sanity Check**
Verify QR code appears in Dashboard when connecting a new account.
