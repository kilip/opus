package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// WaContact holds the schema definition for the WaContact entity.
type WaContact struct {
	ent.Schema
}

// Fields of the WaContact.
func (WaContact) Fields() []ent.Field {
	return []ent.Field{
		field.String("jid"),
		field.String("name"),
		field.String("pushname").Optional(),
	}
}

// Edges of the WaContact.
func (WaContact) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("wa_session", WaSession.Type).Ref("contacts").Unique().Required(),
	}
}

// Indexes of the WaContact.
func (WaContact) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("jid").Edges("wa_session").Unique(),
	}
}
