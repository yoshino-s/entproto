package struct_converter

import (
	"reflect"

	"google.golang.org/protobuf/types/descriptorpb"
)

var kindMap = map[reflect.Kind]typeConfig{
	reflect.Bool:    {pbType: descriptorpb.FieldDescriptorProto_TYPE_BOOL, OptionalType: "google.protobuf.BoolValue"},
	reflect.String:  {pbType: descriptorpb.FieldDescriptorProto_TYPE_STRING, OptionalType: "google.protobuf.StringValue"},
	reflect.Int:     {pbType: descriptorpb.FieldDescriptorProto_TYPE_INT32, OptionalType: "google.protobuf.Int32Value"},
	reflect.Int8:    {pbType: descriptorpb.FieldDescriptorProto_TYPE_INT32, OptionalType: "google.protobuf.Int32Value"},
	reflect.Int16:   {pbType: descriptorpb.FieldDescriptorProto_TYPE_INT32, OptionalType: "google.protobuf.Int32Value"},
	reflect.Int32:   {pbType: descriptorpb.FieldDescriptorProto_TYPE_INT32, OptionalType: "google.protobuf.Int32Value"},
	reflect.Int64:   {pbType: descriptorpb.FieldDescriptorProto_TYPE_INT64, OptionalType: "google.protobuf.Int64Value"},
	reflect.Uint:    {pbType: descriptorpb.FieldDescriptorProto_TYPE_UINT32, OptionalType: "google.protobuf.UInt32Value"},
	reflect.Uint8:   {pbType: descriptorpb.FieldDescriptorProto_TYPE_UINT32, OptionalType: "google.protobuf.UInt32Value"},
	reflect.Uint16:  {pbType: descriptorpb.FieldDescriptorProto_TYPE_UINT32, OptionalType: "google.protobuf.UInt32Value"},
	reflect.Uint32:  {pbType: descriptorpb.FieldDescriptorProto_TYPE_UINT32, OptionalType: "google.protobuf.UInt32Value"},
	reflect.Uint64:  {pbType: descriptorpb.FieldDescriptorProto_TYPE_UINT64, OptionalType: "google.protobuf.UInt64Value"},
	reflect.Float32: {pbType: descriptorpb.FieldDescriptorProto_TYPE_FLOAT, OptionalType: "google.protobuf.FloatValue"},
	reflect.Float64: {pbType: descriptorpb.FieldDescriptorProto_TYPE_DOUBLE, OptionalType: "google.protobuf.DoubleValue"},
}

type typeConfig struct {
	pbType       descriptorpb.FieldDescriptorProto_Type
	OptionalType string
}
