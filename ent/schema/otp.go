package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// OTP holds the schema definition for the OTP entity.
type OTP struct {
	ent.Schema
}

// Fields of the OTP.
func (OTP) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New),
		field.String("email").
			NotEmpty(),
		field.String("code_hash").
			NotEmpty(),
		field.Time("expires_at"),
		field.Int("attempts").
			Default(0),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
	}
}

// Indexes of the OTP.
func (OTP) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("email"),
	}
}
