package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// TrackerSync holds per-project tracker sync state and cursor.
type TrackerSync struct {
	ent.Schema
}

// Fields of the TrackerSync.
func (TrackerSync) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(newID),
		field.UUID("project_id", uuid.UUID{}).
			Unique(),
		field.String("cursor").
			Default(""),
		field.Time("last_sync_at").
			Optional().
			Nillable(),
		field.String("last_error").
			Default(""),
	}
}

// Edges of the TrackerSync.
func (TrackerSync) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("project", Project.Type).
			Ref("tracker_sync").
			Field("project_id").
			Unique().
			Required(),
	}
}
