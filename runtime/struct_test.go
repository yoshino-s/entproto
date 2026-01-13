package runtime

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
	"github.com/yoshino-s/entproto/internal/test/ent/schema"
	"github.com/yoshino-s/entproto/internal/test/proto/entpb"
)

type TestStruct struct {
	StringField     string `json:"string_field"`
	IntField        int32  `json:"int_field"`
	BoolField       bool   `json:"bool_field"`
	unexportedField string `json:"unexported_field"`

	ListField []string         `json:"list_field"`
	MapField  map[string]int32 `json:"map_field"`

	NestedField     NestedStruct            `json:"nested_field"`
	ListNestedField []NestedStruct          `json:"list_nested_field"`
	MapNestedField  map[string]NestedStruct `json:"map_nested_field"`

	AnyField any `json:"any_field"`
}

type NestedStruct struct {
	A string `json:"a"`
}

func TestStruct2Proto(t *testing.T) {
	Convey("Given a struct", t, func() {
		v := TestStruct{
			StringField:     "abc",
			IntField:        123,
			BoolField:       false,
			NestedField:     NestedStruct{A: "a"},
			ListNestedField: []NestedStruct{{A: "b"}},
		}
		sv, err := ToStructPbValue(v)
		So(err, ShouldBeNil)
		var newV TestStruct
		err = FromStructPbValue(sv, &newV)
		So(err, ShouldBeNil)
		So(newV.StringField, ShouldEqual, "abc")
		So(newV.IntField, ShouldEqual, 123)
		So(newV.BoolField, ShouldBeFalse)
		So(newV.NestedField.A, ShouldEqual, "a")
		So(newV.ListNestedField[0].A, ShouldEqual, "b")
		So(newV.unexportedField, ShouldEqual, "")
	})
}

func TestStructToMessage(t *testing.T) {
	structValue := &schema.TestStruct{
		StringField: "test",
		IntField:    1,
		BoolField:   true,
		ListField:   []string{"item1", "item2"},
		MapField:    map[string]int32{"key1": 1, "key2": 2},
		NestedField: &schema.NestedStruct{
			A: "test",
		},
		ListNestedField: []schema.NestedStruct{
			{A: "test"},
			{A: "test2"},
		},
		MapNestedField: map[string]schema.NestedStruct{
			"key1": {A: "test"},
			"key2": {A: "test2"},
		},
		AnyField: "test",
	}

	protoStruct := &entpb.TestStruct{}

	err := ToProtoMessage(structValue, &protoStruct)
	assert.NoError(t, err)

	fmt.Printf("%+v\n", protoStruct)

	// newStruct := &schema.TestStruct{}
	// err = FromProtoMessage(protoStruct, newStruct)
	// assert.NoError(t, err)

	// fmt.Println(newStruct)

	// assert.Equal(t, structValue, newStruct)
}
