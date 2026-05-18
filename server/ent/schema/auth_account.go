package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// AuthAccount defines third-party linkage credentials.
type AuthAccount struct {
	ent.Schema
}

// Fields defines credential-specific columns.
func (AuthAccount) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Immutable().Unique(),
		field.String("user_id"),
		field.String("account_id"),
		field.String("provider_id"),
		field.String("access_token").Optional(),
		field.String("refresh_token").Optional(),
		field.Time("access_token_expires_at").Optional().Nillable(),
		field.String("scope").Optional(),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

// Indexes forces constraint uniqueness across provider IDs and internal accounts.
func (AuthAccount) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("provider_id", "account_id").Unique(),
	}
}

// Edges hooks account ownership to the User schema.
func (AuthAccount) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).Ref("accounts").Field("user_id").Unique().Required(),
	}
}
