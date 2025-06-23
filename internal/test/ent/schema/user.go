package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/yoshino-s/entproto"
)

// User holds the schema definition for the User entity.
type User struct {
	ent.Schema
}

func (User) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entproto.Message(),
		entproto.Service(),
	}
}

// Fields of the User.
func (User) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			Annotations(
				entproto.Field(2),
				entproto.Filter(entproto.WithFilterMode(entproto.FilterModeContains|entproto.FilterModeEQ|entproto.FilterModeIn)),
			),
		field.String("description").
			Optional().
			Annotations(
				entproto.Field(4),
			),
		field.Enum("gender").
			Values("male", "female").
			Annotations(
				entproto.Field(5),
				entproto.Enum(map[string]int32{
					"male":   1,
					"female": 2,
				}),
				entproto.Filter(entproto.WithFilterMode(entproto.FilterModeEQ|entproto.FilterModeIn)),
			),
		field.Time("created_at").
			Immutable().
			Annotations(
				entproto.Field(3),
				entproto.Filter(entproto.WithFilterMode(entproto.FilterModeEQ|entproto.FilterModeIn)),
			),
		field.Int("group_id").
			Optional().
			Annotations(
				entproto.Field(6),
			),
	}
}

// Edges of the User.
func (User) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("group", Group.Type).
			Ref("users").
			Field("group_id").
			Unique().
			Annotations(
				entproto.Field(7),
			),
	}

}
