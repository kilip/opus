// api/ent/schema/user.go
package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type User struct {
	ent.Schema
}

func (User) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Unique().Immutable(),
		field.String("email").Unique(),
		field.String("name"),
		field.String("avatar_url").Optional(),
		field.String("provider"), // "google" | "github" | "email"
		field.String("provider_id").Optional(),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (User) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("email").Unique(),
		index.Fields("provider", "provider_id"),
	}
}

func (User) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("sessions", Session.Type),
	}
}
