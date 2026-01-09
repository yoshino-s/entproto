package convert

import (
	"strings"

	"google.golang.org/protobuf/types/descriptorpb"
)

func (c *Converter) identToFieldType(msg *descriptorpb.DescriptorProto, name, ident string) (FieldType, error) {
	if strings.HasPrefix(ident, "[]") {
		field := nativeIdentToFieldType(ident[2:])
		field.Repeated = true
		return field, nil
	}
	if strings.HasPrefix(ident, "map[string]") {
		f := nativeIdentToFieldType(ident[len("map[string]"):])
		if f.ProtoType == descriptorpb.FieldDescriptorProto_TYPE_MESSAGE {
			return FieldType{
				ProtoType:   descriptorpb.FieldDescriptorProto_TYPE_MESSAGE,
				MessageName: "google.protobuf.Value",
			}, nil
		}

		msg.NestedType = append(msg.NestedType, &descriptorpb.DescriptorProto{
			Name: ptr(pascal(name) + "Entry"),
			Field: []*descriptorpb.FieldDescriptorProto{
				{
					Name:   ptr("key"),
					Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
					Number: ptr[int32](1),
				},
				{
					Name:     ptr("value"),
					Type:     f.ProtoType.Enum(),
					TypeName: &f.MessageName,
					Number:   ptr[int32](2),
				},
			},
			Options: &descriptorpb.MessageOptions{
				MapEntry: ptr(true),
			},
		})
		return FieldType{
			ProtoType:   descriptorpb.FieldDescriptorProto_TYPE_MESSAGE,
			MessageName: pascal(name) + "Entry",
			Repeated:    true,
		}, nil
	}
	return nativeIdentToFieldType(ident), nil
}

func nativeIdentToFieldType(ident string) FieldType {
	switch ident {
	case "bool":
		return FieldType{
			ProtoType: descriptorpb.FieldDescriptorProto_TYPE_BOOL,
		}
	case "int":
		return FieldType{
			ProtoType: descriptorpb.FieldDescriptorProto_TYPE_INT32,
		}
	case "string":
		return FieldType{
			ProtoType: descriptorpb.FieldDescriptorProto_TYPE_STRING,
		}
	default:
		return FieldType{
			ProtoType:   descriptorpb.FieldDescriptorProto_TYPE_MESSAGE,
			MessageName: "google.protobuf.Value",
		}
	}
}
