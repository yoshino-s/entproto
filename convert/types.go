package convert

import (
	"entgo.io/ent/entc/gen"
	"entgo.io/ent/schema/field"
	"google.golang.org/protobuf/types/descriptorpb"
)

var TypeMap = map[field.Type]typeConfig{
	field.TypeBool:  {pbType: descriptorpb.FieldDescriptorProto_TYPE_BOOL, OptionalType: "google.protobuf.BoolValue"},
	field.TypeTime:  {pbType: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE, msgTypeName: "google.protobuf.Timestamp", OptionalType: "google.protobuf.Timestamp"},
	field.TypeOther: {unsupported: true},
	field.TypeUUID:  {pbType: descriptorpb.FieldDescriptorProto_TYPE_BYTES, OptionalType: "google.protobuf.BytesValue"},
	field.TypeBytes: {pbType: descriptorpb.FieldDescriptorProto_TYPE_BYTES, OptionalType: "google.protobuf.BytesValue"},
	field.TypeEnum: {pbType: descriptorpb.FieldDescriptorProto_TYPE_ENUM, namer: func(fld *gen.Field) string {
		return pascal(fld.Name)
	}},
	field.TypeString:  {pbType: descriptorpb.FieldDescriptorProto_TYPE_STRING, OptionalType: "google.protobuf.StringValue"},
	field.TypeInt:     {pbType: descriptorpb.FieldDescriptorProto_TYPE_INT32, OptionalType: "google.protobuf.Int32Value"},
	field.TypeInt8:    {pbType: descriptorpb.FieldDescriptorProto_TYPE_INT32, OptionalType: "google.protobuf.Int32Value"},
	field.TypeInt16:   {pbType: descriptorpb.FieldDescriptorProto_TYPE_INT32, OptionalType: "google.protobuf.Int32Value"},
	field.TypeInt32:   {pbType: descriptorpb.FieldDescriptorProto_TYPE_INT32, OptionalType: "google.protobuf.Int32Value"},
	field.TypeInt64:   {pbType: descriptorpb.FieldDescriptorProto_TYPE_INT64, OptionalType: "google.protobuf.Int64Value"},
	field.TypeUint:    {pbType: descriptorpb.FieldDescriptorProto_TYPE_UINT32, OptionalType: "google.protobuf.UInt32Value"},
	field.TypeUint8:   {pbType: descriptorpb.FieldDescriptorProto_TYPE_UINT32, OptionalType: "google.protobuf.UInt32Value"},
	field.TypeUint16:  {pbType: descriptorpb.FieldDescriptorProto_TYPE_UINT32, OptionalType: "google.protobuf.UInt32Value"},
	field.TypeUint32:  {pbType: descriptorpb.FieldDescriptorProto_TYPE_UINT32, OptionalType: "google.protobuf.UInt32Value"},
	field.TypeUint64:  {pbType: descriptorpb.FieldDescriptorProto_TYPE_UINT64, OptionalType: "google.protobuf.UInt64Value"},
	field.TypeFloat32: {pbType: descriptorpb.FieldDescriptorProto_TYPE_FLOAT, OptionalType: "google.protobuf.FloatValue"},
	field.TypeFloat64: {pbType: descriptorpb.FieldDescriptorProto_TYPE_DOUBLE, OptionalType: "google.protobuf.DoubleValue"},
}

type typeConfig struct {
	unsupported  bool
	pbType       descriptorpb.FieldDescriptorProto_Type
	msgTypeName  string
	OptionalType string
	namer        func(fld *gen.Field) string
}
