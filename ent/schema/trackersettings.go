package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// TrackerSettings holds the singleton global tracker configuration.
type TrackerSettings struct {
	ent.Schema
}

// Fields of the TrackerSettings.
func (TrackerSettings) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(newID),
		field.JSON("status_labels", map[string]string{}).
			Optional(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}
