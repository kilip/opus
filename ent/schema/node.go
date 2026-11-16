package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// Node holds the schema definition for the Node entity.
type Node struct {
	ent.Schema
}

// Fields of the Node.
func (Node) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(newID),
		field.String("name").
			NotEmpty().
			Unique(),
		field.Enum("role").
			Values("master", "worker"),
		field.Enum("status").
			Values("online", "offline"),
		field.Time("last_seen_at").
			Optional().
			Nillable(),
		field.String("credential_hash").
			Optional().
			Nillable(),
	}
}
