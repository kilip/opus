package schema

import (
	"encoding/json"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// SessionEvent holds the schema definition for an ordered event emitted by a Session.
type SessionEvent struct {
	ent.Schema
}

// Fields of the SessionEvent.
func (SessionEvent) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(newID),
		field.UUID("session_id", uuid.UUID{}),
		field.Int("seq").
			Positive(),
		field.String("kind").
			NotEmpty(),
		field.JSON("payload", json.RawMessage{}).
			Optional(),
		field.Time("at").
			Default(time.Now).
			Immutable(),
	}
}

// Edges of the SessionEvent.
func (SessionEvent) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("session", Session.Type).
			Ref("events").
			Field("session_id").
			Required().
			Unique(),
	}
}

// Indexes of the SessionEvent.
func (SessionEvent) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("session_id", "seq").
			Unique(),
	}
}
