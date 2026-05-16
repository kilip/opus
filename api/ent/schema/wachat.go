package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// WaChat holds the schema definition for the WaChat entity.
type WaChat struct {
	ent.Schema
}

// Fields of the WaChat.
func (WaChat) Fields() []ent.Field {
	return []ent.Field{
		field.String("jid"),
		field.String("name").Optional(),
		field.Int("unread_count").Default(0),
	}
}

// Edges of the WaChat.
func (WaChat) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("wa_session", WaSession.Type).Ref("chats").Unique().Required(),
		edge.To("messages", WaMessage.Type),
	}
}
