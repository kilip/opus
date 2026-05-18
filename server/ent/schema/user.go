package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// User holds the schema definition for the User entity.
type User struct {
	ent.Schema
}

// Fields defines the User entity fields.
func (User) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Immutable().Unique(),
		field.String("email").Unique().NotEmpty(),
		field.String("password_hash").Optional().Nillable().Sensitive(),
		field.String("name").Optional(),
		field.String("avatar_url").Optional(),
		field.String("provider"),
		field.String("provider_id"),
		field.String("workspace_id"),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

// Edges defines relationships from user to tokens, sessions, and accounts.
func (User) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("accounts", AuthAccount.Type),
		edge.To("sessions", AuthSession.Type),
		edge.To("tokens", AuthToken.Type),
	}
}
