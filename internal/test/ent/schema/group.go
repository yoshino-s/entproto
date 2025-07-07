package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/yoshino-s/entproto"
)

type GroupMetadata struct {
	Version string `json:"version,omitempty"`
}

type Group struct {
	ent.Schema
}

func (Group) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entproto.Message(),
		entproto.Service(),
	}
}

func (Group) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			Annotations(
				entproto.Field(2),
			),
		field.JSON("metadata", GroupMetadata{}).
			Annotations(
				entproto.Field(4),
			),
		field.JSON("tags", []string{}).
			Annotations(
				entproto.Field(5),
			),
	}
}

func (Group) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("users", User.Type).
			Annotations(
				entproto.Field(3),
			),
	}
}
