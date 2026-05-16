package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/kilip/opus/api/ent"
	"github.com/kilip/opus/api/ent/wasession"
	"github.com/kilip/opus/api/internal/model"
	"github.com/kilip/opus/api/internal/service"
	"go.mau.fi/whatsmeow/proto/waHistorySync"
	"google.golang.org/protobuf/proto"
)

// WhatsAppSyncHandler handles the background synchronization of WhatsApp history.
func WhatsAppSyncHandler(db *ent.Client) service.HandlerFunc {
	return func(ctx context.Context, job *model.Job) error {
		var payload struct {
			UserID string `json:"user_id"`
			Data   []byte `json:"data"`
		}
		if err := json.Unmarshal(job.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload: %w", err)
		}

		var syncData waHistorySync.HistorySync
		if err := proto.Unmarshal(payload.Data, &syncData); err != nil {
			return fmt.Errorf("failed to unmarshal sync data: %w", err)
		}

		sess, err := db.WaSession.Query().Where(wasession.IDEQ(payload.UserID)).Only(ctx)
		if err != nil {
			return fmt.Errorf("failed to find session: %w", err)
		}

		// Sync Pushnames as Contacts
		for _, pn := range syncData.GetPushnames() {
			jid := pn.GetID()
			if jid == "" {
				continue
			}
			err := db.WaContact.Create().
				SetJid(jid).
				SetName(pn.GetPushname()).
				SetPushname(pn.GetPushname()).
				SetWaSession(sess).
				OnConflict().
				UpdateNewValues().
				Exec(ctx)
			if err != nil {
				continue
			}
		}

		// Sync InlineContacts
		for _, ic := range syncData.GetInlineContacts() {
			jid := ic.GetPnJID()
			if jid == "" {
				jid = ic.GetLidJID()
			}
			if jid == "" {
				continue
			}
			name := ic.GetFullName()
			if name == "" {
				name = ic.GetFirstName()
			}
			err := db.WaContact.Create().
				SetJid(jid).
				SetName(name).
				SetPushname(ic.GetUsername()).
				SetWaSession(sess).
				OnConflict().
				UpdateNewValues().
				Exec(ctx)
			if err != nil {
				continue
			}
		}

		for _, conv := range syncData.GetConversations() {
			chatJID := conv.GetID()
			if chatJID == "" {
				continue
			}

			chatID, err := db.WaChat.Create().
				SetJid(chatJID).
				SetName(conv.GetName()).
				SetWaSession(sess).
				OnConflict().
				UpdateNewValues().
				ID(ctx)
			if err != nil {
				continue
			}

			for _, syncMsg := range conv.GetMessages() {
				webMsg := syncMsg.GetMessage()
				if webMsg == nil {
					continue
				}

				key := webMsg.GetKey()
				if key == nil {
					continue
				}

				msgID := key.GetID()
				msgContent := webMsg.GetMessage()
				if msgContent == nil {
					continue
				}

				content := ""
				if msgContent.GetConversation() != "" {
					content = msgContent.GetConversation()
				} else if msgContent.GetExtendedTextMessage().GetText() != "" {
					content = msgContent.GetExtendedTextMessage().GetText()
				}

				if content == "" {
					continue
				}

				ts := time.Unix(int64(webMsg.GetMessageTimestamp()), 0)

				// Upsert message
				err = db.WaMessage.Create().
					SetMessageID(msgID).
					SetSenderJid(key.GetRemoteJID()).
					SetContent(content).
					SetTimestamp(ts).
					SetIsFromMe(key.GetFromMe()).
					SetWaSession(sess).
					SetChatID(chatID).
					OnConflict().
					UpdateNewValues().
					Exec(ctx)
				if err != nil {
					continue
				}
			}
		}

		return nil
	}
}
