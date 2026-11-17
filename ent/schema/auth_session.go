package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// AuthSession holds the schema definition for the AuthSession entity.
type AuthSession struct {
	ent.Schema
}

// Fields of the AuthSession.
func (AuthSession) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(newID),
		field.UUID("user_id", uuid.UUID{}),
		field.String("token_hash").
			NotEmpty().
			Unique(),
		field.Time("expires_at"),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
	}
}

// Edges of the AuthSession.
func (AuthSession) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("sessions").
			Field("user_id").
			Required().
			Unique(),
	}
}
