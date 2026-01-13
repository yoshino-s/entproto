package schema

import (
	"time"

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

type TestStruct struct {
	StringField   string        `json:"string_field"`
	IntField      int32         `json:"int_field"`
	BoolField     bool          `json:"bool_field"`
	BytesField    []byte        `json:"bytes_field"`
	TimeField     time.Time     `json:"time_field"`
	DurationField time.Duration `json:"duration_field"`

	PtrBoolField   *bool    `json:"ptr_bool_field"`
	PtrIntField    *int32   `json:"ptr_int_field"`
	PtrStringField *string  `json:"ptr_string_field"`
	PtrFloatField  *float32 `json:"ptr_float_field"`
	PtrDoubleField *float64 `json:"ptr_double_field"`
	// PtrBytesField  *[]byte  `json:"ptr_bytes_field"`
	PtrTimeField     *time.Time     `json:"ptr_time_field"`
	PtrDurationField *time.Duration `json:"ptr_duration_field"`

	ListField []string         `json:"list_field"`
	MapField  map[string]int32 `json:"map_field"`

	NestedField     *NestedStruct           `json:"nested_field"`
	ListNestedField []NestedStruct          `json:"list_nested_field"`
	MapNestedField  map[string]NestedStruct `json:"map_nested_field"`

	AnyField any `json:"any_field"`
}

type NestedStruct struct {
	A string `json:"a"`
}

func (Group) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			Annotations(
				entproto.Field(2),
			),
		field.JSON("metadata", map[string]string{}).
			Annotations(
				entproto.Field(4, entproto.MarshaledGoType(map[string]string{})),
			),
		field.JSON("tags", []string{}).
			Annotations(
				entproto.Field(5, entproto.MarshaledGoType([]string{})),
			),
		field.JSON("some_struct", TestStruct{}).
			Annotations(
				entproto.Field(6, entproto.MarshaledGoType(TestStruct{})),
			),
		field.JSON("metadata_struct", &GroupMetadata{}).
			Annotations(
				entproto.Field(7, entproto.MarshaledGoType(&GroupMetadata{})),
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
