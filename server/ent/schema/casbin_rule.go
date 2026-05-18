package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// CasbinRule defines the schema for Casbin policy rules.
type CasbinRule struct {
	ent.Schema
}

// Fields of the CasbinRule.
func (CasbinRule) Fields() []ent.Field {
	return []ent.Field{
		field.String("ptype").Optional().Default(""),
		field.String("v0").Optional().Default(""),
		field.String("v1").Optional().Default(""),
		field.String("v2").Optional().Default(""),
		field.String("v3").Optional().Default(""),
		field.String("v4").Optional().Default(""),
		field.String("v5").Optional().Default(""),
	}
}
