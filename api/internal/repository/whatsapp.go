package repository

import (
	"context"

	"github.com/kilip/opus/api/ent"
	"github.com/kilip/opus/api/ent/user"
	"github.com/kilip/opus/api/ent/wasession"
)

// WhatsAppRepository defines the interface for WhatsApp persistence.
type WhatsAppRepository interface {
	UpsertSession(ctx context.Context, userID string, status string, jid string) (*ent.WaSession, error)
	GetSessionByUserID(ctx context.Context, userID string) (*ent.WaSession, error)
}

type whatsappRepo struct {
	client *ent.Client
}

// NewWhatsAppRepository creates a new WhatsApp repository.
func NewWhatsAppRepository(client *ent.Client) WhatsAppRepository {
	return &whatsappRepo{client: client}
}

// UpsertSession creates or updates a WhatsApp session for a user.
func (r *whatsappRepo) UpsertSession(ctx context.Context, userID string, status string, jid string) (*ent.WaSession, error) {
	// First check if it exists
	session, err := r.client.WaSession.Query().
		Where(wasession.HasUserWith(user.IDEQ(userID))).
		Only(ctx)
	
	if err != nil && !ent.IsNotFound(err) {
		return nil, err
	}

	if ent.IsNotFound(err) {
		create := r.client.WaSession.Create().
			SetID(userID). // Use userID as WaSession ID for simplicity or generate new
			SetStatus(status).
			SetUserID(userID)
			
		if jid != "" {
			create.SetJid(jid)
		}
		return create.Save(ctx)
	}

	update := session.Update().SetStatus(status)
	if jid != "" {
		update.SetJid(jid)
	}
	return update.Save(ctx)
}

// GetSessionByUserID retrieves a WhatsApp session by user ID.
func (r *whatsappRepo) GetSessionByUserID(ctx context.Context, userID string) (*ent.WaSession, error) {
	return r.client.WaSession.Query().
		Where(wasession.HasUserWith(user.IDEQ(userID))).
		Only(ctx)
}
