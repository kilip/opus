package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// WaMessage holds the schema definition for the WaMessage entity.
type WaMessage struct {
	ent.Schema
}

// Fields of the WaMessage.
func (WaMessage) Fields() []ent.Field {
	return []ent.Field{
		field.String("message_id"),
		field.String("sender_jid"),
		field.Text("content").Optional(),
		field.Time("timestamp").Default(time.Now),
		field.Bool("is_from_me").Default(false),
	}
}

// Edges of the WaMessage.
func (WaMessage) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("wa_session", WaSession.Type).Ref("messages").Unique().Required(),
		edge.From("chat", WaChat.Type).Ref("messages").Unique().Required(),
	}
}

// Indexes of the WaMessage.
func (WaMessage) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("message_id").Edges("wa_session").Unique(),
	}
}
