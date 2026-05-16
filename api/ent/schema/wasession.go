package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// WaSession holds the schema definition for the WaSession entity.
type WaSession struct {
	ent.Schema
}

// Fields of the WaSession.
func (WaSession) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Unique(),
		field.String("jid").Optional(),
		field.String("status").Default("UNAUTHENTICATED"),
	}
}

// Edges of the WaSession.
func (WaSession) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).Ref("wa_session").Unique().Required(),
		edge.To("contacts", WaContact.Type),
		edge.To("chats", WaChat.Type),
		edge.To("messages", WaMessage.Type),
	}
}
