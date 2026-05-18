package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// AuthToken stores SHA-256 hashes of access and refresh tokens.
type AuthToken struct {
	ent.Schema
}

// Fields maps token parameters and revocation.
func (AuthToken) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Immutable().Unique(),
		field.String("session_id"),
		field.String("user_id"),
		field.String("type"), // "access" or "refresh"
		field.String("hash").Unique(),
		field.Time("expires_at"),
		field.Time("revoked_at").Optional().Nillable(),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}

// Edges maintains structural integrity with Session and User entities.
func (AuthToken) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("session", AuthSession.Type).Ref("tokens").Field("session_id").Unique().Required(),
		edge.From("user", User.Type).Ref("tokens").Field("user_id").Unique().Required(),
	}
}
