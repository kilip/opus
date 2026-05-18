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
		field.String("type").NotEmpty(),
		field.String("queue").Default("default"),
		field.Bytes("payload"),
		field.String("status").Default("pending"),
		field.Int("priority").Default(0),
		field.Int("max_retries").Default(3),
		field.Int("retry_count").Default(0),
		field.Time("process_at").Default(time.Now),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

// Indexes of the Job.
func (Job) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("queue", "status", "priority", "process_at"),
	}
}
