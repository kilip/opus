package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// Task holds the schema definition for the Task entity.
type Task struct {
	ent.Schema
}

// Fields of the Task.
func (Task) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(newID),
		field.UUID("project_id", uuid.UUID{}),
		field.String("key").
			NotEmpty().
			Unique(),
		field.String("source").
			NotEmpty().
			Default("manual"),
		field.String("external_id").
			Optional().
			Nillable(),
		field.String("external_url").
			Default(""),
		field.String("title").
			NotEmpty(),
		field.String("description").
			Default(""),
		field.Enum("status").
			Values("backlog", "todo", "in-progress", "review", "blocked", "done").
			Default("todo"),
		field.String("source_status").
			Default(""),
		field.String("priority").
			NotEmpty().
			Default("1"),
		field.String("type").
			NotEmpty().
			Default("feat"),
		field.String("assignee_id").
			Optional().
			Nillable(),
		field.String("assignee_name").
			Optional().
			Nillable(),
		field.String("due_date").
			Default(""),
		field.JSON("labels", []string{}).
			Default([]string{}),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
		field.Time("synced_at").
			Optional().
			Nillable(),
	}
}

// Edges of the Task.
func (Task) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("project", Project.Type).
			Ref("tasks").
			Field("project_id").
			Required().
			Unique(),
		edge.To("activities", Activity.Type).
			Annotations(
				entsql.OnDelete(entsql.Cascade),
			),
		edge.To("tracker_prs", TrackerPR.Type).
			Annotations(
				entsql.OnDelete(entsql.SetNull),
			),
	}
}

// Indexes of the Task.
func (Task) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("source", "external_id").
			Unique(),
	}
}
