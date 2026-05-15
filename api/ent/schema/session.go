// api/ent/schema/session.go
package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type Session struct {
	ent.Schema
}

func (Session) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Unique().Immutable(),
		field.String("token_hash").Unique(), // Hashed refresh token
		field.String("user_id"),
		field.Time("expires_at"),
		field.Bool("revoked").Default(false),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}

func (Session) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("token_hash").Unique(),
		index.Fields("user_id"),
	}
}

func (Session) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).Ref("sessions").Field("user_id").Unique().Required(),
	}
}
