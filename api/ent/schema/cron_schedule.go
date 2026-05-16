package schema

import (
	"time"

	"entgo.io/ent"
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
		field.String("name").Unique(),
		field.String("cron_expression"),
		field.String("job_type"),
		field.Bytes("payload").Optional(),
		field.Bool("is_active").Default(true),
		field.Time("last_run_at").Optional(),
		field.Time("next_run_at").Optional(),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

// Indexes of the CronSchedule.
func (CronSchedule) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("is_active"),
		index.Fields("next_run_at"),
	}
}
