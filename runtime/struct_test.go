package runtime

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

type TestStruct struct {
	StringField     string `mapstructure:"string_field"`
	IntField        int32  `mapstructure:"int_field"`
	BoolField       bool   `mapstructure:"bool_field"`
	unexportedField string `mapstructure:"unexported_field"`

	ListField []string         `mapstructure:"list_field"`
	MapField  map[string]int32 `mapstructure:"map_field"`

	NestedField     NestedStruct            `mapstructure:"nested_field"`
	ListNestedField []NestedStruct          `mapstructure:"list_nested_field"`
	MapNestedField  map[string]NestedStruct `mapstructure:"map_nested_field"`

	AnyField any `mapstructure:"any_field"`
}

type NestedStruct struct {
	A string `mapstructure:"a"`
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
