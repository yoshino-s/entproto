package runtime

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

type TestStruct struct {
	StringField     string
	IntField        int32
	BoolField       bool
	unexportedField string

	ListField []string
	MapField  map[string]int32

	NestedField     NestedStruct
	ListNestedField []NestedStruct
	MapNestedField  map[string]NestedStruct

	AnyField any
}

type NestedStruct struct {
	A string
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
