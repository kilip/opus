package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
 
	"github.com/kilip/opus/api/ent"
	"github.com/kilip/opus/api/ent/wachat"
	"github.com/kilip/opus/api/internal/model"
	"github.com/kilip/opus/api/internal/repository"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"google.golang.org/protobuf/proto"
)

const JobWhatsAppSyncHistory = "whatsapp.sync_history"
 
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
		return "", "", err
	}
	return session.Status, session.Jid, nil
}
 
func (s *whatsAppService) Connect(ctx context.Context, userID string) error {
	if client, ok := s.clients.Load(userID); ok {
		c := client.(*whatsmeow.Client)
		if c.IsConnected() {
			return nil
		}
	}
 
	device, err := s.store.GetFirstDevice(ctx)
	if err != nil {
		return fmt.Errorf("failed to get device from store: %w", err)
	}
 
	client := whatsmeow.NewClient(device, nil)
	s.clients.Store(userID, client)
 
	s.registerHandler(userID, client)
 
	qrChan, err := client.GetQRChannel(ctx)
	if err != nil {
		// If already logged in, GetQRChannel returns error
		if !client.IsLoggedIn() {
			return fmt.Errorf("failed to get QR channel: %w", err)
		}
	} else {
		go func() {
			for evt := range qrChan {
				if evt.Event == "code" {
					_ = s.sseHub.Publish(context.Background(), userID, SSEEvent{
						Type:    "wa_qr_update",
						Payload: map[string]string{"qr": evt.Code},
					})
				}
			}
		}()
	}
 
	return client.Connect()
}
 
func (s *whatsAppService) registerHandler(userID string, client *whatsmeow.Client) {
	client.AddEventHandler(func(evt interface{}) {
		switch v := evt.(type) {
		case *events.Connected:
			_ = s.repo.UpdateStatus(context.Background(), userID, "CONNECTED", client.Store.ID.String())
			_ = s.sseHub.Publish(context.Background(), userID, SSEEvent{
				Type:    "wa_connected",
				Payload: map[string]string{"jid": client.Store.ID.String()},
			})
		case *events.Message:
			s.handleNewMessage(userID, v)
		case *events.HistorySync:
			s.handleHistorySync(userID, v)
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
