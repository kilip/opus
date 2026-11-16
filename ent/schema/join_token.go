package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// JoinToken holds the schema definition for the JoinToken entity.
type JoinToken struct {
	ent.Schema
}

// Fields of the JoinToken.
func (JoinToken) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(newID),
		field.String("token_hash").
			NotEmpty().
			Unique(),
		field.Time("expires_at"),
		field.Time("used_at").
			Optional().
			Nillable(),
	}
}
