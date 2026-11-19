package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// SecretSetting stores encrypted runtime secrets (e.g. GitHub token) at rest.
type SecretSetting struct {
	ent.Schema
}

// Fields of the SecretSetting.
func (SecretSetting) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(newID),
		field.String("key").
			Unique().
			NotEmpty(),
		field.Bytes("ciphertext").
			NotEmpty(),
		field.Bytes("nonce").
			NotEmpty(),
		field.Int("key_version").
			Default(1),
		field.String("last4").
			NotEmpty(),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}
