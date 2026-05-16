package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// CronSchedule holds the schema definition for the CronSchedule entity.
type CronSchedule struct {
	ent.Schema
}

// Fields of the CronSchedule.
func (CronSchedule) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Unique().Immutable(),
		field.String("name"),
		field.String("cron_expression"),
		field.String("job_type"),
		field.Bytes("payload").Optional(),
		field.Bool("is_active").Default(true),
		field.Time("last_run_at").Optional(),
		field.Time("next_run_at").Optional(),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
		field.String("user_id"),
	}
}

// Edges of the CronSchedule.
func (CronSchedule) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("cron_schedules").
			Unique().
			Field("user_id").
			Required(),
	}
}

// Indexes of the CronSchedule.
func (CronSchedule) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("is_active"),
		index.Fields("next_run_at"),
		index.Fields("user_id", "name").Unique(),
	}
}
