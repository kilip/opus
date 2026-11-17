package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// ProjectNode holds the schema definition for the ProjectNode entity.
type ProjectNode struct {
	ent.Schema
}

// Fields of the ProjectNode.
func (ProjectNode) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(newID),
		field.UUID("project_id", uuid.UUID{}),
		field.UUID("node_id", uuid.UUID{}),
		field.String("path").
			Optional().
			Nillable(),
	}
}

// Edges of the ProjectNode.
func (ProjectNode) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("project", Project.Type).
			Ref("project_nodes").
			Field("project_id").
			Unique().
			Required(),
		edge.From("node", Node.Type).
			Ref("project_nodes").
			Field("node_id").
			Unique().
			Required(),
	}
}

// Indexes of the ProjectNode.
func (ProjectNode) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("project_id", "node_id").
			Unique(),
	}
}
