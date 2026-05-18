package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// AuthSession groups token pairs.
type AuthSession struct {
	ent.Schema
}

// Fields defines session identity.
func (AuthSession) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Immutable().Unique(),
		field.String("user_id"),
		field.String("workspace_id"),
		field.String("ip_address").Optional(),
		field.String("user_agent").Optional(),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}

// Edges defines relationships to User and Token elements.
func (AuthSession) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).Ref("sessions").Field("user_id").Unique().Required(),
		edge.To("tokens", AuthToken.Type),
	}
}
