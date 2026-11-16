package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// Counter holds the schema definition for the Counter entity.
type Counter struct {
	ent.Schema
}

// Fields of the Counter.
func (Counter) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(newID),
		field.String("name").
			NotEmpty().
			Unique(),
		field.Int64("value").
			Default(0),
	}
}
