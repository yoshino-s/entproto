// Copyright 2019-present Facebook
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"encoding"
	"fmt"
	"reflect"
	"strings"

	"entgo.io/ent/entc/gen"
	"entgo.io/ent/schema/field"
	"github.com/yoshino-s/entproto"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var (
	binaryMarshallerUnmarshallerType = reflect.TypeOf((*BinaryMarshallerUnmarshaller)(nil)).Elem()
)

type BinaryMarshallerUnmarshaller interface {
	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler
}

type converter struct {
	ToEntConversion              string
	ToEntScannerConversion       string
	ToEntConstructor             protogen.GoIdent
	ToEntMarshallerConstructor   protogen.GoIdent
	ToEntScannerConstructor      protogen.GoIdent
	ToEntModifier                string
	ToProtoConversion            string
	ToProtoConstructor           protogen.GoIdent
	toProtoMarshallerConstructor protogen.GoIdent
	ToProtoValuer                string
}

func (g *generator) newConverter(fld *entproto.FieldMappingDescriptor, pbds ...protoreflect.FieldDescriptor) (*converter, error) {
	out := &converter{}

	var pbd protoreflect.FieldDescriptor
	if len(pbds) == 0 || pbds[0] == nil {
		pbd = fld.PbFieldDescriptor
	} else {
		pbd = pbds[0]
	}
	switch pbd.Kind() {
	case protoreflect.BoolKind, protoreflect.StringKind,
		protoreflect.BytesKind, protoreflect.Int32Kind,
		protoreflect.Int64Kind, protoreflect.Uint32Kind,
		protoreflect.Uint64Kind, protoreflect.FloatKind,
		protoreflect.DoubleKind:
		if err := basicTypeConversion(fld.PbFieldDescriptor, fld.EntField, out); err != nil {
			return nil, err
		}
	case protoreflect.EnumKind:
		enumName := fld.PbFieldDescriptor.Enum().Name()
		method := fmt.Sprintf("toProto%s_%s", g.EntType.Name, enumName)
		out.ToProtoConstructor = g.GoImportPath.Ident(method)
	case protoreflect.MessageKind:
		if strings.HasSuffix(string(pbd.Message().FullName()), "EnumValue") {
			// This is a special case for enum values, which are represented as messages in protobuf.
			// We need to convert them to the corresponding enum type in ent.
			enumName := pbd.Message().Name()
			method := fmt.Sprintf("toProto%s_%s", g.EntType.Name, enumName)
			out.ToProtoConstructor = g.GoImportPath.Ident(method)
			out.ToEntModifier = ".GetValue()"
		} else if fld.IsEdgeField {
			if err := basicTypeConversion(fld.EdgeIDPbStructFieldDesc(), fld.EntEdge.Type.ID, out); err != nil {
				return nil, err
			}
		} else if err := convertPbMessageType(pbd.Message(), fld.EntField, out); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("entproto: no mapping for pb field type %q", pbd.Kind())
	}
	efld := fld.EntField
	if fld.IsEdgeField {
		efld = fld.EntEdge.Type.ID
	}

	switch {
	case implements(efld.Type.RType, binaryMarshallerUnmarshallerType) && efld.HasGoType():
		// Ident returned from ent already has the packagename prefixed. Strip it since `g.QualifiedGoIdent`
		// adds it back.
		split := strings.Split(efld.Type.Ident, ".")

		out.ToEntMarshallerConstructor = protogen.GoImportPath(efld.Type.PkgPath).Ident(split[1])
	case efld.Type.ValueScanner():
		switch {
		case efld.HasGoType():
			// Ident returned from ent already has the packagename prefixed. Strip it since `g.QualifiedGoIdent`
			// adds it back.
			split := strings.Split(efld.Type.Ident, ".")
			out.ToEntScannerConstructor = protogen.GoImportPath(efld.Type.PkgPath).Ident(split[1])
		case efld.IsBool():
			out.ToEntScannerConversion = "bool"
		case efld.IsBytes():
			out.ToEntScannerConversion = "[]byte"
		case efld.IsString():
			out.ToEntScannerConversion = "string"
		}
	case efld.IsBool(), efld.IsBytes(), efld.IsString():
	case efld.Type.Numeric():
		out.ToEntConversion = efld.Type.String()
	case efld.IsTime():
		out.ToEntConstructor = protogen.GoImportPath("github.com/yoshino-s/entproto/runtime").Ident("ExtractTime")
	case efld.IsEnum():
		if fld.PbFieldDescriptor.Enum() != nil {
			enumName := fld.PbFieldDescriptor.Enum().Name()
			method := fmt.Sprintf("toEnt%s_%s", g.EntType.Name, enumName)
			out.ToEntConstructor = g.GoImportPath.Ident(method)
		} else {
			enumName := fld.PbFieldDescriptor.Message().Name()
			method := fmt.Sprintf("toEnt%s_%s", g.EntType.Name, enumName)
			out.ToEntConstructor = g.GoImportPath.Ident(method)
		}
	case efld.IsJSON():
		switch efld.Type.Ident {
		case "[]string":
		case "[]int32", "[]int64", "[]uint32", "[]uint64":
			out.ToProtoConversion = ""
		default:
			return nil, fmt.Errorf("entproto: no mapping to ent field type %q", efld.Type.ConstName())
		}
	default:
		return nil, fmt.Errorf("entproto: no mapping to ent field type %q", efld.Type.ConstName())
	}
	return out, nil
}

// Supported value scanner types (https://golang.org/pkg/database/sql/driver/#Value): [int64, float64, bool, []byte, string, time.Time]
func basicTypeConversion(md protoreflect.FieldDescriptor, entField *gen.Field, conv *converter) error {
	switch md.Kind() {
	case protoreflect.BoolKind:
		if entField.Type.Valuer() {
			conv.ToProtoValuer = "bool"
		}
	case protoreflect.StringKind:
		if entField.Type.Valuer() {
			conv.ToProtoValuer = "string"
		}
	case protoreflect.BytesKind:
		if implements(entField.Type.RType, binaryMarshallerUnmarshallerType) {
			// Ident returned from ent already has the packagename prefixed. Strip it since `g.QualifiedGoIdent`
			// adds it back.
			split := strings.Split(entField.Type.Ident, ".")
			conv.toProtoMarshallerConstructor = protogen.GoImportPath(entField.Type.PkgPath).Ident(split[1])
		} else if entField.Type.Valuer() {
			conv.ToProtoValuer = "[]byte"
		}
	case protoreflect.Int32Kind:
		if entField.Type.String() != "int32" {
			conv.ToProtoConversion = "int32"
		}
	case protoreflect.Int64Kind:
		if entField.Type.Valuer() {
			conv.ToProtoValuer = "int64"
		} else if entField.Type.String() != "int64" {
			conv.ToProtoConversion = "int64"
		}
	case protoreflect.Uint32Kind:
		if entField.Type.String() != "uint32" {
			conv.ToProtoConversion = "uint32"
		}
	case protoreflect.Uint64Kind:
		if entField.Type.String() != "uint64" {
			conv.ToProtoConversion = "uint64"
		}
	case protoreflect.FloatKind:
		if entField.Type.String() != "float32" {
			conv.ToProtoConversion = "float32"
		}
	case protoreflect.DoubleKind:
		if entField.Type.Valuer() {
			conv.ToProtoConversion = "float64"
		}
	}
	return nil
}

func convertPbMessageType(md protoreflect.MessageDescriptor, entField *gen.Field, conv *converter) error {
	switch {
	case md.FullName() == "google.protobuf.Timestamp":
		conv.ToProtoConstructor = protogen.GoImportPath("google.golang.org/protobuf/types/known/timestamppb").Ident("New")
	case isWrapperType(md):
		fqn := md.FullName()
		typ := strings.Split(string(fqn), ".")[2]
		constructor := strings.TrimSuffix(typ, "Value")
		conv.ToProtoConstructor = protogen.GoImportPath("google.golang.org/protobuf/types/known/wrapperspb").Ident(constructor)

		goType := wrapperPrimitives[fqn]
		if entField.Type.Valuer() {
			conv.ToProtoValuer = goType
		} else if entField.Type.String() != goType {
			conv.ToProtoConversion = goType
		}
		conv.ToEntModifier = ".GetValue()"
	default:
		return fmt.Errorf("entproto(convertPbMessageType): no mapping for pb field type %q", md.FullName())
	}
	return nil
}

func isWrapperType(md protoreflect.MessageDescriptor) bool {
	_, ok := wrapperPrimitives[md.FullName()]
	return ok
}

var wrapperPrimitives = map[protoreflect.FullName]string{
	"google.protobuf.DoubleValue": "float64",
	"google.protobuf.FloatValue":  "float32",
	"google.protobuf.Int64Value":  "int64",
	"google.protobuf.UInt64Value": "uint64",
	"google.protobuf.Int32Value":  "int32",
	"google.protobuf.UInt32Value": "uint32",
	"google.protobuf.BoolValue":   "bool",
	"google.protobuf.StringValue": "string",
	"google.protobuf.BytesValue":  "[]byte",
}

func implements(r *field.RType, typ reflect.Type) bool {
	if r == nil {
		return false
	}
	n := typ.NumMethod()
	for i := 0; i < n; i++ {
		m0 := typ.Method(i)
		m1, ok := r.Methods[m0.Name]
		if !ok || len(m1.In) != m0.Type.NumIn() || len(m1.Out) != m0.Type.NumOut() {
			return false
		}
		in := m0.Type.NumIn()
		for j := 0; j < in; j++ {
			if !m1.In[j].TypeEqual(m0.Type.In(j)) {
				return false
			}
		}
		out := m0.Type.NumOut()
		for j := 0; j < out; j++ {
			if !m1.Out[j].TypeEqual(m0.Type.Out(j)) {
				return false
			}
		}
	}
	return true
}
