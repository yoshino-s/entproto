package struct_converter

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yoshino-s/entproto/internal/testutils"
	"google.golang.org/protobuf/types/descriptorpb"
)

type TestStruct struct {
	StringField   string        `json:"string_field"`
	IntField      int32         `json:"int_field"`
	BoolField     bool          `json:"bool_field"`
	BytesField    []byte        `json:"bytes_field"`
	TimeField     time.Time     `json:"time_field"`
	DurationField time.Duration `json:"duration_field"`

	PtrBoolField     *bool          `json:"ptr_bool_field"`
	PtrIntField      *int32         `json:"ptr_int_field"`
	PtrStringField   *string        `json:"ptr_string_field"`
	PtrFloatField    *float32       `json:"ptr_float_field"`
	PtrDoubleField   *float64       `json:"ptr_double_field"`
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

type messageContainerImpl struct {
	messages       []*descriptorpb.DescriptorProto
	existedTypeMap map[string]struct{}
}

func (m *messageContainerImpl) AddMessage(typ *MarshaledGoType, msg *descriptorpb.DescriptorProto) (bool, error) {
	if _, ok := m.existedTypeMap[typ.Pkg+"."+typ.Name]; ok {
		return false, nil
	}
	m.existedTypeMap[typ.Pkg+"."+typ.Name] = struct{}{}
	m.messages = append(m.messages, msg)
	return true, nil
}

func TestConvertFromStruct(t *testing.T) {
	container := &messageContainerImpl{
		existedTypeMap: make(map[string]struct{}),
	}
	converter := NewStructConverter(container)
	tt, err := MarshalGoType(reflect.TypeFor[TestStruct]())
	assert.NoError(t, err)
	_, err = converter.Convert(tt, "", nil)
	if err != nil {
		t.Fatalf("failed to convert struct: %v", err)
	}

	res, err := testutils.DumpMessageDescriptors(container.messages)
	assert.NoError(t, err)
	fmt.Println(res)
}
