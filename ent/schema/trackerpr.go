package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// TrackerPR holds a tracked GitHub pull request snapshot.
type TrackerPR struct {
	ent.Schema
}

// Fields of the TrackerPR.
func (TrackerPR) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(newID),
		field.UUID("project_id", uuid.UUID{}),
		field.Int("number").
			Positive(),
		field.UUID("task_id", uuid.UUID{}).
			Optional().
			Nillable(),
		field.Enum("state").
			Values("open", "closed", "merged").
			Default("open"),
		field.Bool("is_draft").
			Default(false),
		field.String("review_state").
			Default(""),
		field.Enum("checks_summary").
			Values("passing", "failing", "pending", "none").
			Default("none"),
		field.String("url").
			Default(""),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

// Edges of the TrackerPR.
func (TrackerPR) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("project", Project.Type).
			Ref("tracker_prs").
			Field("project_id").
			Unique().
			Required(),
		edge.From("task", Task.Type).
			Ref("tracker_prs").
			Field("task_id").
			Unique(),
	}
}

// Indexes of the TrackerPR.
func (TrackerPR) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("project_id", "number").
			Unique(),
		index.Fields("task_id"),
	}
}
