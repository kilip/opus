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

// Run holds the schema definition for an automated task execution Run.
type Run struct {
	ent.Schema
}

// Fields of the Run.
func (Run) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(newID),
		field.UUID("task_id", uuid.UUID{}),
		field.UUID("project_id", uuid.UUID{}),
		field.UUID("node_id", uuid.UUID{}).
			Optional().
			Nillable(),
		field.UUID("session_id", uuid.UUID{}).
			Optional().
			Nillable(),
		// ponytail: MVP supports "vibe" mode only; add "to_pr" / "plan" modes and gate states post-MVP.
		field.Enum("mode").
			Values("vibe").
			Default("vibe"),
		field.Enum("state").
			Values(
				"queued",
				"preparing",
				"running",
				"merged",
				"failed",
				"needs-info",
				"paused",
				"cancelled",
			).
			Default("queued"),
		field.Int("stage_attempts").
			Default(0).
			NonNegative(),
		field.String("branch").
			Default(""),
		field.Int("pr_number").
			Optional().
			Nillable(),
		field.JSON("report", json.RawMessage{}).
			Optional(),
		field.Time("paused_until").
			Optional().
			Nillable(),
		field.String("failure_reason").
			Optional().
			Nillable(),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("started_at").
			Optional().
			Nillable(),
		field.Time("ended_at").
			Optional().
			Nillable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

// Edges of the Run.
func (Run) Edges() []ent.Edge {
	return []ent.Edge{
		// One-way link to Session; Session intentionally has no run_id back-reference.
		edge.To("session", Session.Type).
			Field("session_id").
			Unique(),
	}
}

// Indexes of the Run.
func (Run) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("task_id"),
		index.Fields("project_id"),
		index.Fields("state"),
		index.Fields("session_id"),
	}
}
