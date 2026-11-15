package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// Account holds the schema definition for the Account entity.
type Account struct {
	ent.Schema
}

// Fields of the Account.
func (Account) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New),
		field.UUID("user_id", uuid.UUID{}),
		field.String("provider").
			NotEmpty(),
		field.String("provider_user_id").
			NotEmpty(),
		field.String("email").
			NotEmpty(),
		field.Bool("email_verified").
			Default(false),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
	}
}

// Edges of the Account.
func (Account) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("accounts").
			Field("user_id").
			Required().
			Unique(),
	}
}

// Indexes of the Account.
func (Account) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("provider", "provider_user_id").
			Unique(),
	}
}
