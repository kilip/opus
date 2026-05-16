package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Job holds the schema definition for the Job entity.
type Job struct {
	ent.Schema
}

// Fields of the Job.
func (Job) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Unique().Immutable(),
		field.String("type"),
		field.Bytes("payload"),
		field.Int("priority").Default(0),
		field.String("status").Default("pending"),
		field.Int("retries").Default(0),
		field.Int("max_retries").Default(3),
		field.Time("scheduled_at").Default(time.Now),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
		field.String("error").Optional(),
	}
}

// Indexes of the Job.
func (Job) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("status"),
		index.Fields("priority"),
		index.Fields("scheduled_at"),
		index.Fields("status", "priority", "scheduled_at"),
	}
}
