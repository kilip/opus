package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
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
		field.String("hostname").
			Optional().
			Nillable(),
		field.String("ip_address").
			Optional().
			Nillable(),
		field.String("mac_address").
			Optional().
			Nillable(),
	}
}

// Edges of the Node.
func (Node) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("project_nodes", ProjectNode.Type).
			Annotations(
				entsql.OnDelete(entsql.Cascade),
			),
	}
}
