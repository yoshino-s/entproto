package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema"
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
			),
		field.String("description").
			Optional().
			Annotations(
				entproto.Field(4),
			),
		field.Time("created_at").
			Immutable().
			Annotations(
				entproto.Field(3),
			),
	}
}

// Edges of the User.
func (User) Edges() []ent.Edge {
	return nil
}
