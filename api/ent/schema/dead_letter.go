package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// DeadLetter holds the schema definition for the DeadLetter entity.
type DeadLetter struct {
	ent.Schema
}

// Fields of the DeadLetter.
func (DeadLetter) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Unique().Immutable(),
		field.String("job_id"),
		field.String("type"),
		field.Bytes("payload"),
		field.String("last_error").Optional(),
		field.Int("retries"),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.String("user_id"),
	}
}

// Edges of the DeadLetter.
func (DeadLetter) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("dead_letters").
			Unique().
			Field("user_id").
			Required(),
	}
}

// Indexes of the DeadLetter.
func (DeadLetter) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("job_id"),
		index.Fields("user_id"),
	}
}
