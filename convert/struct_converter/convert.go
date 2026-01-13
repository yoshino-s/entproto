package struct_converter

import (
	"reflect"
	"strings"
	"unicode"

	"entgo.io/ent/entc/gen"
	"github.com/go-errors/errors"
	"google.golang.org/protobuf/types/descriptorpb"
)

type messageContainer interface {
	AddMessage(typ *MarshaledGoType, msg *descriptorpb.DescriptorProto) (bool, error)
}

type StructConverter struct {
	messageContainer
}

func NewStructConverter(messageContainer messageContainer) *StructConverter {
	return &StructConverter{
		messageContainer: messageContainer,
	}
}

func (sc *StructConverter) Convert(typ *MarshaledGoType, name string, container *descriptorpb.DescriptorProto) (*descriptorpb.FieldDescriptorProto, error) {
	return sc.convert(typ, name, container)
}

func (sc *StructConverter) convert(typ *MarshaledGoType, name string, container *descriptorpb.DescriptorProto) (*descriptorpb.FieldDescriptorProto, error) {
	if typ.IsAny {
		return &descriptorpb.FieldDescriptorProto{
			Type:     ptr(descriptorpb.FieldDescriptorProto_TYPE_MESSAGE),
			TypeName: ptr("google.protobuf.Value"),
		}, nil
	}

	if (typ.Kind == reflect.Slice || typ.Kind == reflect.Array) && typ.ElementType.Kind == reflect.Uint8 {
		return &descriptorpb.FieldDescriptorProto{
			Type: ptr(descriptorpb.FieldDescriptorProto_TYPE_BYTES),
		}, nil
	}
	if typ.Name == "Time" && typ.Pkg == "time" {
		return &descriptorpb.FieldDescriptorProto{
			Type:     ptr(descriptorpb.FieldDescriptorProto_TYPE_MESSAGE),
			TypeName: ptr("google.protobuf.Timestamp"),
		}, nil
	}
	if typ.Name == "Duration" && typ.Pkg == "time" {
		return &descriptorpb.FieldDescriptorProto{
			Type:     ptr(descriptorpb.FieldDescriptorProto_TYPE_MESSAGE),
			TypeName: ptr("google.protobuf.Duration"),
		}, nil
	}

	switch typ.Kind {
	case reflect.Slice, reflect.Array:
		return sc.convertList(typ, name, container)
	case reflect.Struct:
		return sc.convertStruct(typ, name, container)
	case reflect.Map:
		return sc.convertMap(typ, name, container)
	case reflect.Ptr:
		return sc.convertPtr(typ, name, container)
	default:
		return sc.converNativetValue(typ, name, container)
	}
}

func (sc *StructConverter) converNativetValue(typ *MarshaledGoType, name string, container *descriptorpb.DescriptorProto) (*descriptorpb.FieldDescriptorProto, error) {
	tc, ok := kindMap[typ.Kind]
	if !ok {
		return nil, errors.Errorf("struct_converter: unsupported field type: %s", typ.Kind.String())
	}

	return &descriptorpb.FieldDescriptorProto{
		Type: &tc.pbType,
	}, nil
}

func (sc *StructConverter) convertList(typ *MarshaledGoType, name string, container *descriptorpb.DescriptorProto) (*descriptorpb.FieldDescriptorProto, error) {
	fdp, err := sc.convert(typ.ElementType, name, container)
	if err != nil {
		return nil, err
	}
	fdp.Label = descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum()

	return fdp, nil
}

func (sc *StructConverter) convertStruct(typ *MarshaledGoType, name string, container *descriptorpb.DescriptorProto) (*descriptorpb.FieldDescriptorProto, error) {
	msg := &descriptorpb.DescriptorProto{
		Name: ptr(typ.Name),
	}
	if ok, err := sc.messageContainer.AddMessage(typ, msg); err != nil {
		return nil, err
	} else if ok { // 不存在这个message，新建一下
		for i, field := range typ.Fields {
			tagName := snake(field.Name)
			tag := reflect.StructTag(field.Tag).Get("json")
			if tag != "" {
				name, _, _ := strings.Cut(tag, ",")
				if isValidTag(name) {
					tagName = name
				}
			}

			fdp, err := sc.convert(field.Type, tagName, msg)
			if err != nil {
				return nil, err
			}
			fdp.Name = ptr(tagName)
			fdp.Number = ptr(int32(i + 1))
			msg.Field = append(msg.Field, fdp)
		}
	}
	return &descriptorpb.FieldDescriptorProto{
		Type:     ptr(descriptorpb.FieldDescriptorProto_TYPE_MESSAGE),
		TypeName: msg.Name,
	}, nil
}

func (sc *StructConverter) convertMap(typ *MarshaledGoType, fieldName string, container *descriptorpb.DescriptorProto) (*descriptorpb.FieldDescriptorProto, error) {
	keyType := typ.KeyType
	valueType := typ.ElementType

	// where the key_type can be any integral or string type (so, any scalar type except for floating point types and bytes). Note that neither enum nor proto messages are valid for key_type. The value_type can be any type except another map.
	if keyType.Kind != reflect.String && keyType.Kind != reflect.Int && keyType.Kind != reflect.Int8 &&
		keyType.Kind != reflect.Int16 && keyType.Kind != reflect.Int32 && keyType.Kind != reflect.Int64 &&
		keyType.Kind != reflect.Uint && keyType.Kind != reflect.Uint8 && keyType.Kind != reflect.Uint16 &&
		keyType.Kind != reflect.Uint32 && keyType.Kind != reflect.Uint64 {
		return nil, errors.Errorf("struct_converter: unsupported map key type: %s", keyType.Kind.String())
	}
	if valueType.Kind == reflect.Map {
		return nil, errors.New("struct_converter: nested maps are not supported")
	}

	keyField, err := sc.convert(keyType, "key", container)
	valueField, err := sc.convert(valueType, "value", container)
	if err != nil {
		return nil, err
	}
	keyField.Number = ptr(int32(1))
	keyField.Name = ptr("key")
	valueField.Number = ptr(int32(2))
	valueField.Name = ptr("value")

	mapMsg := &descriptorpb.DescriptorProto{
		Name: ptr(pascal(fieldName) + "Entry"),
		Field: []*descriptorpb.FieldDescriptorProto{
			keyField,
			valueField,
		},
	}
	mapMsg.Options = &descriptorpb.MessageOptions{
		MapEntry: ptr(true),
	}
	container.NestedType = append(container.NestedType, mapMsg)

	return &descriptorpb.FieldDescriptorProto{
		Type:     ptr(descriptorpb.FieldDescriptorProto_TYPE_MESSAGE),
		TypeName: ptr(mapMsg.GetName()),
		Label:    ptr(descriptorpb.FieldDescriptorProto_LABEL_REPEATED),
	}, nil
}

func (sc *StructConverter) convertPtr(typ *MarshaledGoType, name string, container *descriptorpb.DescriptorProto) (*descriptorpb.FieldDescriptorProto, error) {
	elemTyp := typ.ElementType
	fdp, err := sc.convert(elemTyp, name, container)
	if err != nil {
		return nil, err
	}

	if elemTyp.Kind == reflect.Struct || elemTyp.Name == "Time" || elemTyp.Name == "Duration" {
		return fdp, nil
	}

	tc, ok := kindMap[elemTyp.Kind]
	if !ok {
		return nil, errors.Errorf("struct_converter: unsupported field type: %s", elemTyp.Kind.String())
	}

	if tc.OptionalType == "" {
		return nil, errors.Errorf("struct_converter: optional not supported for field type: %s", elemTyp.Kind.String())
	}

	return &descriptorpb.FieldDescriptorProto{
		Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
		TypeName: ptr(tc.OptionalType),
	}, nil
}

func ptr[T any](v T) *T {
	return &v
}

var (
	snake  = gen.Funcs["snake"].(func(string) string)
	pascal = gen.Funcs["pascal"].(func(string) string)
	camel  = gen.Funcs["camel"].(func(string) string)
)

func isValidTag(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		switch {
		case strings.ContainsRune("!#$%&()*+-./:;<=>?@[]^_{|}~ ", c):
			// Backslash and quote chars are reserved, but
			// otherwise any punctuation chars are allowed
			// in a tag name.
		case !unicode.IsLetter(c) && !unicode.IsDigit(c):
			return false
		}
	}
	return true
}
