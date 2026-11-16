package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// Project holds the schema definition for the Project entity.
type Project struct {
	ent.Schema
}

// Fields of the Project.
func (Project) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(newID),
		field.String("slug").
			NotEmpty().
			Unique().
			Immutable(),
		field.String("name").
			NotEmpty(),
		field.String("repo_provider").
			NotEmpty(),
		field.String("repo_ref").
			NotEmpty(),
		field.String("tracker_provider").
			NotEmpty(),
		field.String("tracker_ref").
			NotEmpty(),
		field.String("default_branch").
			NotEmpty().
			Default("main"),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

// Edges of the Project.
func (Project) Edges() []ent.Edge {
	return nil
}
