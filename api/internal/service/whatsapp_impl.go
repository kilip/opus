package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"
 
	"github.com/kilip/opus/api/ent"
	"github.com/kilip/opus/api/ent/wachat"
	"github.com/kilip/opus/api/internal/model"
	"github.com/kilip/opus/api/internal/repository"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"google.golang.org/protobuf/proto"
)

const JobWhatsAppSyncHistory = "whatsapp.sync_history"

type retryEntry struct {
	attempts int
	timer    *time.Timer
	mu       sync.Mutex
}

var backoffSchedule = []time.Duration{
	5 * time.Second,
	15 * time.Second,
	30 * time.Second,
	1 * time.Minute,
	5 * time.Minute,
	30 * time.Minute,
}
 
type whatsAppService struct {
	clients    sync.Map // map[string]*whatsmeow.Client
	retryState sync.Map // map[string]*any
	repo       repository.WhatsAppRepository
	sseHub     SSEHub
	queue      Queue
	db         *ent.Client
	store      *sqlstore.Container
	log        *slog.Logger
}
 
// NewWhatsAppService creates a new WhatsApp service.
func NewWhatsAppService(
	repo repository.WhatsAppRepository,
	sseHub SSEHub,
	queue Queue,
	db *ent.Client,
	store *sqlstore.Container,
	log *slog.Logger,
) WhatsAppService {
	return &whatsAppService{
		repo:   repo,
		sseHub: sseHub,
		queue:  queue,
		db:     db,
		store:  store,
		log:    log,
	}
}
 
func (s *whatsAppService) Initialize(ctx context.Context) error {
	sessions, err := s.repo.GetAllActiveSessions(ctx)
	if err != nil {
		return fmt.Errorf("failed to list active sessions: %w", err)
	}
	for _, sess := range sessions {
		s.log.Info("recovering whatsapp session", "userID", sess.ID)
		if err := s.Connect(ctx, sess.ID); err != nil {
			s.log.Error("failed to recover session", "userID", sess.ID, "error", err)
		}
	}
	return nil
}
 
func (s *whatsAppService) GetStatus(ctx context.Context, userID string) (status string, jid string, err error) {
	session, err := s.repo.GetSessionByUserID(ctx, userID)
	if err != nil {
		if ent.IsNotFound(err) {
			return "UNAUTHENTICATED", "", nil
		}
		return "", "", err
	}
	return session.Status, session.Jid, nil
}
 
func (s *whatsAppService) Connect(ctx context.Context, userID string) error {
	var client *whatsmeow.Client
	if val, ok := s.clients.Load(userID); ok {
		client = val.(*whatsmeow.Client)
		if client.IsConnected() {
			return nil
		}
	}

	if client == nil {
		session, err := s.repo.GetSessionByUserID(ctx, userID)
		if err != nil {
			if ent.IsNotFound(err) {
				// Create a new session if not found
				session, err = s.repo.UpsertSession(ctx, userID, "UNAUTHENTICATED", "")
				if err != nil {
					return fmt.Errorf("failed to create initial session: %w", err)
				}
			} else {
				return fmt.Errorf("failed to get session: %w", err)
			}
		}

		var device *store.Device
		if session.Jid != "" {
			jid, err := types.ParseJID(session.Jid)
			if err != nil {
				return fmt.Errorf("failed to parse JID: %w", err)
			}
			device, err = s.store.GetDevice(ctx, jid)
			if err != nil {
				return fmt.Errorf("failed to get device from store: %w", err)
			}
		}

		if device == nil {
			device = s.store.NewDevice()
		}

		client = whatsmeow.NewClient(device, nil)
		s.clients.Store(userID, client)
		s.registerHandler(userID, client)
	}

	if !client.IsLoggedIn() {
		s.log.Info("Client not logged in, getting QR channel", "userID", userID)
		qrChan, err := client.GetQRChannel(ctx)
		if err != nil {
			s.log.Error("Failed to get QR channel", "userID", userID, "error", err)
			return fmt.Errorf("failed to get QR channel: %w", err)
		}
		go func() {
			s.log.Info("Starting QR channel listener", "userID", userID)
			for evt := range qrChan {
				s.log.Debug("Received QR event", "userID", userID, "event", evt.Event)
				if evt.Event == "code" {
					s.log.Info("New QR code generated, publishing to SSE", "userID", userID)
					_ = s.sseHub.Publish(context.Background(), userID, SSEEvent{
						Type:    "wa_qr_update",
						Payload: map[string]string{"qr": evt.Code},
					})
				} else {
					// Notify frontend that QR state changed (e.g. expired, waiting for next)
					s.log.Info("QR event received", "userID", userID, "event", evt.Event)
					_ = s.sseHub.Publish(context.Background(), userID, SSEEvent{
						Type:    "wa_qr_event",
						Payload: map[string]string{"event": evt.Event},
					})
				}
			}
			s.log.Info("QR channel listener stopped", "userID", userID)
		}()
	}

	s.log.Info("Initiating client connection", "userID", userID)
	err := client.Connect()
	if err != nil {
		s.log.Error("Failed to connect client", "userID", userID, "error", err)
		if client.IsLoggedIn() {
			s.handleDisconnect(userID)
		}
	} else {
		s.log.Info("Client connection call completed", "userID", userID)
	}
	return err
}
 
func (s *whatsAppService) registerHandler(userID string, client *whatsmeow.Client) {
	client.AddEventHandler(func(evt interface{}) {
		switch v := evt.(type) {
		case *events.Connected:
			s.log.Info("whatsapp connected", "userID", userID)
			_ = s.repo.UpdateStatus(context.Background(), userID, "CONNECTED", client.Store.ID.String())
			_ = s.sseHub.Publish(context.Background(), userID, SSEEvent{
				Type:    "wa_connected",
				Payload: map[string]string{"jid": client.Store.ID.String()},
			})
			// Reset retry state on successful connection
			if entry, ok := s.retryState.Load(userID); ok {
				e := entry.(*retryEntry)
				e.mu.Lock()
				if e.timer != nil {
					e.timer.Stop()
					e.timer = nil
				}
				e.attempts = 0
				e.mu.Unlock()
			}

		case *events.Disconnected:
			s.log.Info("whatsapp disconnected", "userID", userID)
			_ = s.repo.UpdateStatus(context.Background(), userID, "DISCONNECTED", "")
			s.handleDisconnect(userID)

		case *events.LoggedOut:
			s.log.Info("whatsapp logged out", "userID", userID, "onConnect", v.OnConnect, "reason", v.Reason)
			_ = s.repo.UpdateStatus(context.Background(), userID, "UNAUTHENTICATED", "")
			if entry, ok := s.retryState.Load(userID); ok {
				e := entry.(*retryEntry)
				e.mu.Lock()
				if e.timer != nil {
					e.timer.Stop()
					e.timer = nil
				}
				e.mu.Unlock()
				s.retryState.Delete(userID)
			}
			if client, ok := s.clients.Load(userID); ok {
				c := client.(*whatsmeow.Client)
				if c.Store != nil {
					_ = s.store.DeleteDevice(context.Background(), c.Store)
				}
				s.clients.Delete(userID)
			}

		case *events.Message:
			s.handleNewMessage(userID, v)
		case *events.HistorySync:
			s.handleHistorySync(userID, v)
		}
	})
}

func (s *whatsAppService) getRetryEntry(userID string) *retryEntry {
	entry, _ := s.retryState.LoadOrStore(userID, &retryEntry{})
	return entry.(*retryEntry)
}

func (s *whatsAppService) handleDisconnect(userID string) {
	entry := s.getRetryEntry(userID)
	entry.mu.Lock()
	defer entry.mu.Unlock()

	if entry.timer != nil {
		entry.timer.Stop()
	}

	backoffIdx := entry.attempts
	if backoffIdx >= len(backoffSchedule) {
		backoffIdx = len(backoffSchedule) - 1
	}
	delay := backoffSchedule[backoffIdx]

	s.log.Info("scheduling whatsapp reconnection",
		"userID", userID,
		"attempt", entry.attempts+1,
		"delay", delay,
	)

	entry.attempts++
	entry.timer = time.AfterFunc(delay, func() {
		s.log.Info("attempting to reconnect whatsapp", "userID", userID)
		// Use background context for background reconnection
		ctx := context.Background()
		if err := s.Connect(ctx, userID); err != nil {
			s.log.Error("reconnection attempt failed", "userID", userID, "error", err)
			// handleDisconnect will be called again by registerHandler -> Disconnected
		}
	})
}
 
func (s *whatsAppService) handleNewMessage(userID string, evt *events.Message) {
	content := ""
	if evt.Message.GetConversation() != "" {
		content = evt.Message.GetConversation()
	}
 
	ctx := context.Background()
	sess, err := s.repo.GetSessionByUserID(ctx, userID)
	if err != nil {
		s.log.Error("failed to get session", "userID", userID, "error", err)
		return
	}
 
	// Ensure Chat exists
	chatJID := evt.Info.Chat.String()
	chat, err := s.db.WaChat.Query().Where(wachat.JidEQ(chatJID)).Only(ctx)
	if ent.IsNotFound(err) {
		chat, err = s.db.WaChat.Create().
			SetJid(chatJID).
			SetWaSession(sess).
			Save(ctx)
		if err != nil {
			s.log.Error("failed to create chat", "error", err)
			return
		}
	}
 
	_, err = s.db.WaMessage.Create().
		SetMessageID(evt.Info.ID).
		SetSenderJid(evt.Info.Sender.String()).
		SetContent(content).
		SetTimestamp(evt.Info.Timestamp).
		SetIsFromMe(evt.Info.IsFromMe).
		SetWaSession(sess).
		SetChat(chat).
		Save(ctx)
 
	if err != nil {
		s.log.Error("failed to save message", "error", err)
		return
	}
 
	// Notify via SSE
	_ = s.sseHub.Publish(ctx, userID, SSEEvent{
		Type: "wa_message",
		Payload: map[string]any{
			"id":      evt.Info.ID,
			"sender":  evt.Info.Sender.String(),
			"content": content,
		},
	})
}
 
func (s *whatsAppService) handleHistorySync(userID string, evt *events.HistorySync) {
	data, err := proto.Marshal(evt.Data)
	if err != nil {
		s.log.Error("failed to marshal history sync data", "error", err)
		return
	}

	payload, err := json.Marshal(map[string]any{
		"user_id": userID,
		"data":    data,
	})
	if err != nil {
		s.log.Error("failed to marshal sync job payload", "error", err)
		return
	}

	err = s.queue.Push(context.Background(), &model.Job{
		Type:    JobWhatsAppSyncHistory,
		Payload: payload,
	})
	if err != nil {
		s.log.Error("failed to push sync job", "error", err)
	}
}
 
func (s *whatsAppService) Disconnect(ctx context.Context, userID string) error {
	if client, ok := s.clients.Load(userID); ok {
		client.(*whatsmeow.Client).Disconnect()
		s.clients.Delete(userID)
	}
	return s.repo.UpdateStatus(ctx, userID, "DISCONNECTED", "")
}
 
func (s *whatsAppService) ForceReconnect(ctx context.Context, userID string) error {
	_ = s.Disconnect(ctx, userID)
	s.retryState.Delete(userID)
	return s.Connect(ctx, userID)
}
 
func (s *whatsAppService) SendMessage(ctx context.Context, userID string, targetJID string, message string) error {
	client, ok := s.clients.Load(userID)
	if !ok {
		return fmt.Errorf("user not connected")
	}
	c := client.(*whatsmeow.Client)

	jid, err := types.ParseJID(targetJID)
	if err != nil {
		return fmt.Errorf("invalid target JID: %w", err)
	}

	_, err = c.SendMessage(ctx, jid, &waE2E.Message{
		Conversation: proto.String(message),
	})
	return err
}
