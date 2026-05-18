package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// AuthOauthState registers secure single-use CSRF tokens for callback operations.
type AuthOauthState struct {
	ent.Schema
}

// Fields sets parameters for CSRF timeout configurations.
func (AuthOauthState) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Immutable().Unique(),
		field.String("state").Unique(),
		field.String("provider_id"),
		field.Time("expires_at"),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}
