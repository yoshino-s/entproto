package convert

import (
	"fmt"
	"math"
	"strings"

	"entgo.io/ent/entc/gen"
	"entgo.io/ent/schema/field"
	"github.com/yoshino-s/entproto/annotations"
	"google.golang.org/protobuf/types/descriptorpb"
)

const (
	IDFieldNumber = 1
)

func (c *Converter) EntTypeToDescriptorProto(graph *gen.Graph, genType *gen.Type) ([]*descriptorpb.DescriptorProto, error) {
	msgAnnot, err := annotations.ExtractMessageAnnotation(genType)
	if err != nil || !msgAnnot.Generate {
		return nil, ErrSchemaSkipped
	}
	msg := &descriptorpb.DescriptorProto{
		Name:     &genType.Name,
		EnumType: []*descriptorpb.EnumDescriptorProto(nil),
	}
	msgs := []*descriptorpb.DescriptorProto{msg}

	if !genType.ID.UserDefined {
		genType.ID.Annotations = map[string]interface{}{annotations.FieldAnnotation: annotations.Field(IDFieldNumber)}
	}

	all := []*gen.Field{genType.ID}
	all = append(all, genType.Fields...)

	for _, f := range all {
		if _, ok := f.Annotations[annotations.SkipAnnotation]; ok {
			continue
		}

		protoField, err := c.toProtoFieldDescriptor(f, msg)
		if err != nil {
			return nil, err
		}
		// If the field is an enum type, we need to create the enum descriptor as well.
		if f.Type.Type == field.TypeEnum {
			dp, err := toProtoEnumDescriptor(f)
			if err != nil {
				return nil, err
			}
			msg.EnumType = append(msg.EnumType, dp)

			msgType := descriptorpb.FieldDescriptorProto_TYPE_ENUM
			msgs = append(msgs, &descriptorpb.DescriptorProto{
				Name: ptr(pascal(genType.Name + "_" + f.Name + "_enum_value")),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:     ptr("value"),
						Number:   ptr[int32](1),
						Type:     &msgType,
						TypeName: ptr(genType.Name + "." + *dp.Name),
					},
				},
			})
		}
		msg.Field = append(msg.Field, protoField)
	}

	for _, e := range genType.Edges {
		if _, ok := e.Annotations[annotations.SkipAnnotation]; ok {
			continue
		}

		descriptor, err := c.ExtractEdgeFieldDescriptor(graph, genType, e)
		if err != nil {
			return nil, err
		}
		if descriptor != nil {
			msg.Field = append(msg.Field, descriptor)
		}
	}

	if err := verifyNoDuplicateFieldNumbers(msg); err != nil {
		return nil, err
	}

	return msgs, nil
}

func verifyNoDuplicateFieldNumbers(msg *descriptorpb.DescriptorProto) error {
	mem := make(map[int32]struct{})
	for _, fld := range msg.Field {
		if _, seen := mem[fld.GetNumber()]; seen {
			return fmt.Errorf("entproto: field %d already defined on message %q",
				fld.GetNumber(), msg.GetName())
		} else {
			mem[fld.GetNumber()] = struct{}{}
		}
	}
	return nil
}

func toProtoEnumDescriptor(fld *gen.Field) (*descriptorpb.EnumDescriptorProto, error) {
	enumAnnotation, err := annotations.ExtractEnumAnnotation(fld)
	if err != nil {
		return nil, err
	}
	if err := enumAnnotation.Verify(fld); err != nil {
		return nil, err
	}
	enumName := pascal(fld.Name)
	dp := &descriptorpb.EnumDescriptorProto{
		Name:  ptr(enumName),
		Value: []*descriptorpb.EnumValueDescriptorProto{},
	}
	if !fld.Default {
		dp.Value = append(dp.Value, &descriptorpb.EnumValueDescriptorProto{
			Number: ptr[int32](0),
			Name:   ptr(strings.ToUpper(snake(fld.Name)) + "_UNSPECIFIED"),
		})
	}
	for _, opt := range fld.Enums {
		n := strings.ToUpper(snake(annotations.NormalizeEnumIdentifier(opt.Value)))
		if !enumAnnotation.OmitFieldPrefix {
			n = strings.ToUpper(snake(fld.Name)) + "_" + n
		}
		dp.Value = append(dp.Value, &descriptorpb.EnumValueDescriptorProto{
			Number: ptr[int32](enumAnnotation.Options[opt.Value]),
			Name:   ptr(n),
		})
	}
	return dp, nil
}

func (c *Converter) toProtoFieldDescriptor(f *gen.Field, msg *descriptorpb.DescriptorProto) (*descriptorpb.FieldDescriptorProto, error) {
	fieldDesc := &descriptorpb.FieldDescriptorProto{
		Name: &f.Name,
	}
	fann, err := annotations.ExtractFieldAnnotation(f)
	if err != nil {
		return nil, err
	}
	if num := int64(fann.Number); num > math.MaxInt32 || num < math.MinInt32 {
		return nil, fmt.Errorf("value %v overflows int32", num)
	}
	fieldNumber := int32(fann.Number)
	if fieldNumber == 1 && strings.ToUpper(f.Name) != "ID" {
		return nil, fmt.Errorf("entproto: field %q has number 1 which is reserved for id", f.Name)
	}
	fieldDesc.Number = &fieldNumber
	if fann.Type != descriptorpb.FieldDescriptorProto_Type(0) {
		fieldDesc.Type = &fann.Type
		if len(fann.TypeName) > 0 {
			fieldDesc.TypeName = &fann.TypeName
		}
		return fieldDesc, nil
	}
	typeDetails, err := c.ExtractProtoTypeDetails(f, msg)
	if err != nil {
		return nil, err
	}
	fieldDesc.Type = &typeDetails.ProtoType
	if typeDetails.MessageName != "" {
		fieldDesc.TypeName = &typeDetails.MessageName
	}
	if typeDetails.Repeated {
		fieldDesc.Label = descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum()
	}
	return fieldDesc, nil
}

func (c *Converter) ExtractProtoTypeDetails(f *gen.Field, msg *descriptorpb.DescriptorProto, optional ...bool) (FieldType, error) {
	if f.Type.Type == field.TypeJSON {
		return FieldType{
			ProtoType:   descriptorpb.FieldDescriptorProto_TYPE_MESSAGE,
			MessageName: "google.protobuf.Value",
		}, nil
	}

	cfg, ok := TypeMap[f.Type.Type]
	if !ok || cfg.unsupported {
		return FieldType{}, unsupportedTypeError{Type: f.Type}
	}
	if f.Optional || (len(optional) > 0 && optional[0]) {
		if cfg.OptionalType == "" {
			return FieldType{}, unsupportedTypeError{Type: f.Type}
		}
		return FieldType{
			ProtoType:   descriptorpb.FieldDescriptorProto_TYPE_MESSAGE,
			MessageName: cfg.OptionalType,
		}, nil
	}
	name := cfg.msgTypeName
	if cfg.namer != nil {
		name = cfg.namer(f)
	}
	return FieldType{
		ProtoType:   cfg.pbType,
		MessageName: name,
	}, nil
}

func (c *Converter) ExtractEdgeFieldDescriptor(graph *gen.Graph, source *gen.Type, e *gen.Edge) (*descriptorpb.FieldDescriptorProto, error) {
	t := descriptorpb.FieldDescriptorProto_TYPE_MESSAGE
	msgTypeName := pascal(e.Type.Name)

	edgeAnnotation, err := annotations.ExtractEdgeAnnotation(e)
	if err != nil {
		return nil, fmt.Errorf("entproto: failed extracting proto field number annotation: %w", err)
	}

	if edgeAnnotation.Number == 1 {
		return nil, fmt.Errorf("entproto: edge %q has number 1 which is reserved for id", e.Name)
	}

	if num := int64(edgeAnnotation.Number); num > math.MaxInt32 || num < math.MinInt32 {
		return nil, fmt.Errorf("value %v overflows int32", num)
	}
	fieldNum := int32(edgeAnnotation.Number)
	fieldDesc := &descriptorpb.FieldDescriptorProto{
		Number: &fieldNum,
		Name:   &e.Name,
		Type:   &t,
	}

	if !e.Unique {
		fieldDesc.Label = descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum()
	}

	relType, err := extractGenTypeByName(graph, msgTypeName)
	if err != nil {
		return nil, err
	}
	dstAnnotation, err := annotations.ExtractMessageAnnotation(relType)
	if err != nil || !dstAnnotation.Generate {
		return nil, fmt.Errorf("entproto: message %q is not generated", msgTypeName)
	}

	sourceAnnotation, err := annotations.ExtractMessageAnnotation(source)
	if err != nil {
		return nil, err
	}
	if sourceAnnotation.Package == dstAnnotation.Package {
		fieldDesc.TypeName = &msgTypeName
	} else {
		fqn := dstAnnotation.Package + "." + msgTypeName
		fieldDesc.TypeName = &fqn
	}

	return fieldDesc, nil
}

type FieldType struct {
	MessageName string
	ProtoType   descriptorpb.FieldDescriptorProto_Type
	Repeated    bool
}

type unsupportedTypeError struct {
	Type *field.TypeInfo
}

func (e unsupportedTypeError) Error() string {
	return fmt.Sprintf("unsupported field type %q", e.Type.ConstName())
}

func extractGenTypeByName(graph *gen.Graph, name string) (*gen.Type, error) {
	for _, sch := range graph.Nodes {
		if sch.Name == name {
			return sch, nil
		}
	}
	return nil, fmt.Errorf("entproto: could not find schema %q in graph", name)
}
