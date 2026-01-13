package main

import (
	"fmt"
	"reflect"
	"strings"

	"entgo.io/ent/entc/gen"
	"github.com/go-errors/errors"
	"github.com/yoshino-s/entproto/annotations"
	"github.com/yoshino-s/entproto/convert/struct_converter"
	"google.golang.org/protobuf/compiler/protogen"
)

type extraConverterGenerator struct {
	*messageGenerator

	converters map[string]string
}

func (g *extraConverterGenerator) addConverter(name, converter string) {
	if g.converters == nil {
		g.converters = make(map[string]string)
	}
	g.converters[name] = converter
}

func (g *messageGenerator) ExtraConverters() string {
	ecc := &extraConverterGenerator{
		messageGenerator: g,
	}

	for _, field := range g.EntType.Fields {
		if field.IsJSON() {
			annotation, err := annotations.ExtractFieldAnnotation(field)
			if err != nil {
				continue
			}
			if annotation.MarshaledGoType != nil {
				ecc.createConverter(annotation.MarshaledGoType)
			}
		}
	}

	var converters []string
	for _, cvt := range ecc.converters {
		converters = append(converters, cvt)
	}
	return strings.Join(converters, "\n\n")
}

func (g *extraConverterGenerator) convert(field, inName, outName string, m *struct_converter.MarshaledGoType, direction *extraConverterDirection, decl bool) string {
	action := " = "
	if decl {
		action = " := "
	}

	ref := ""
	deref := ""
	if shouldRef(m) {
		if direction == extraConverterToProto {
			ref = "&"
		} else {
			deref = "*"
		}
	}

	if m.IsAny {
		if direction == extraConverterToProto {
			return g.t(`
	{{.Field}}TmpObj, {{ .ErrName }} := {{ .Convert }}({{ .InName }})
	if {{ .ErrName }} != nil {
		return nil, {{ .ErrName }}
	}
	{{ .OutName }} {{ .Action }} {{ .Field }}TmpObj`, map[string]any{
				"Convert": g.QualifiedGoIdent(g.RuntimePackage.Ident("ToStructPbValue")),
				"InName":  inName,
				"OutName": outName,
				"Action":  action,
				"Field":   field,
				"ErrName": field + "Err",
			})
		} else {
			return g.t(`
	var {{ .Field }}TmpObj {{ .Type }}
	{{ .ErrName }} := {{ .Convert }}({{ .InName }}, &{{ .Field }}TmpObj)
	if {{ .ErrName }} != nil {
		return nil, {{ .ErrName }}
	}	
	{{ .OutName }} {{ .Action }} {{ .Field }}TmpObj`, map[string]any{
				"Convert": g.QualifiedGoIdent(g.RuntimePackage.Ident("FromStructPbValue")),
				"InName":  inName,
				"OutName": outName,
				"Action":  action,
				"Type":    g.typeToTypeString(m),
				"Field":   field,
				"ErrName": field + "Err",
			})
		}
	}

	if m.Name == "Time" && m.Pkg == "time" {
		if direction == extraConverterToProto {
			return g.t(`{{ .OutName }} {{ .Action }} {{.Converter}}({{ .InName }})`, map[string]any{
				"OutName":   outName,
				"Action":    action,
				"InName":    inName,
				"Converter": g.QualifiedGoIdent(protogen.GoImportPath("google.golang.org/protobuf/types/known/timestamppb").Ident("New")),
			})
		}
		return g.t(`if {{ .InName }} != nil {
				{{ .OutName }} {{ .Action }} {{ .InName }}.AsTime()
			}`, map[string]any{
			"OutName": outName,
			"Action":  action,
			"InName":  inName,
		})
	}

	if m.Name == "Duration" && m.Pkg == "time" {
		if direction == extraConverterToProto {
			return g.t(`{{ .OutName }} {{ .Action }} {{.Converter}}({{ .InName }})`, map[string]any{
				"OutName":   outName,
				"Action":    action,
				"InName":    inName,
				"Converter": g.QualifiedGoIdent(protogen.GoImportPath("google.golang.org/protobuf/types/known/durationpb").Ident("New")),
			})
		}
		return g.t(`if {{ .InName }} != nil {
			{{ .OutName }} {{ .Action }} {{ .InName }}.AsDuration()
		}`, map[string]any{
			"OutName": outName,
			"Action":  action,
			"InName":  inName,
		})
	}

	if isPrimitiveType(m) {
		return g.t(`{{ .OutName }} {{ .Action }} {{ .InName }}`, map[string]any{
			"OutName": outName,
			"Action":  action,
			"InName":  inName,
		})
	}

	switch m.Kind {
	case reflect.Slice:
		return g.t(`
	for _, item := range {{ .InName }} {
		{{.Code}}
		{{ .OutName }} = append({{ .OutName }}, {{.Deref}}{{ .Field }}TmpObj)
	}`, map[string]any{
			"InName":      inName,
			"OutName":     outName,
			"Code":        g.convert("item", "item", field+"TmpObj", m.ElementType, direction, true),
			"Field":       field,
			"ElementType": g.typeToTypeString(m.ElementType),
			"Deref":       deref,
		})
	case reflect.Struct:
		c := g.createStructConverter(m, direction)
		return g.t(`
		{{ .Field }}TmpObj, {{ .ErrName }} := {{ .ConverterName }}({{.Ref}}{{ .InName }})
		if {{ .ErrName }} != nil {
			return nil, {{ .ErrName }}
		}
		{{ .OutName }} {{ .Action }} {{.Deref}}{{ .Field }}TmpObj`, map[string]any{
			"ConverterName": c,
			"InName":        inName,
			"OutName":       outName,
			"Action":        action,
			"Ref":           ref,
			"Deref":         deref,
			"Field":         field,
			"ErrName":       field + "Err",
		})
	case reflect.Map:
		return g.t(`for key, value := range {{ .InName }} {
			{{.Code}}
		}`, map[string]any{
			"InName":  inName,
			"OutName": outName,
			"Code":    g.convert("value", "value", fmt.Sprintf("%s[key]", outName), m.ElementType, direction, false),
		})
	case reflect.Pointer:
		elemType := m.ElementType

		if isPrimitiveType(elemType) {
			if direction == extraConverterToProto {
				return g.t(`
				if {{ .InName }} != nil {
					{{ .OutName }} {{ .Action }} {{.Converter}}(*{{ .InName }})
				} else {
					{{ .OutName }} {{ .Action }} nil
				}`, map[string]any{
					"OutName":   outName,
					"Action":    action,
					"InName":    inName,
					"Converter": g.primitiveWrapper(elemType),
				})
			}
			return g.t(`
			if {{ .InName }} != nil {
				vv := {{ .InName }}{{ .UnWrapper }}
				{{ .OutName }} {{ .Action }} &vv
			} else {
				{{ .OutName }} {{ .Action }} nil
			}
			`, map[string]any{
				"OutName":   outName,
				"Action":    action,
				"InName":    inName,
				"UnWrapper": g.primitiveunWrapper(elemType),
			})
		}

		switch elemType.Kind {
		case reflect.Struct:
			c := g.createStructConverter(elemType, direction)
			return g.t(`
			{{ .Field }}TmpObj, {{ .ErrName }} := {{ .ConverterName }}({{ .InName }})
			if {{ .ErrName }} != nil {
				return nil, {{ .ErrName }}
			}
			{{ .OutName }} {{ .Action }} {{ .Field }}TmpObj`, map[string]any{
				"OutName":       outName,
				"Action":        action,
				"ConverterName": c,
				"InName":        inName,
				"Field":         field,
				"ErrName":       field + "Err",
			})
		default:
			panic(errors.Errorf("convert: unsupported go type for proto conversion: %s", m.Kind.String()))
		}
	default:
		panic(errors.Errorf("convert: unsupported go type for proto conversion: %s", m.Kind.String()))
	}
}

func (g *extraConverterGenerator) createConverter(m *struct_converter.MarshaledGoType) {
	if isPrimitiveType(m) {
		return
	}

	if m.Name == "Time" && m.Pkg == "time" {
		return
	}
	if m.Name == "Duration" && m.Pkg == "time" {
		return
	}

	switch m.Kind {
	case reflect.Struct:
		g.createStructConverter(m, extraConverterToProto)
		g.createStructConverter(m, extraConverterFromProto)
	case reflect.Slice:
		g.createSliceConverter(m, extraConverterToProto)
		g.createSliceConverter(m, extraConverterFromProto)
	case reflect.Map:
		g.createMapConverter(m, extraConverterToProto)
		g.createMapConverter(m, extraConverterFromProto)
	case reflect.Pointer:
		elemType := m.ElementType

		if elemType.Name == "Time" && elemType.Pkg == "time" {
			return
		}
		if elemType.Name == "Duration" && elemType.Pkg == "time" {
			return
		}

		switch elemType.Kind {
		case reflect.Struct:
			g.createStructConverter(elemType, extraConverterToProto)
			g.createStructConverter(elemType, extraConverterFromProto)
		default:
			panic(errors.Errorf("createConverter: unsupported go type for proto conversion: %s", m.Kind.String()))
		}
	default:
		panic(errors.Errorf("createConverter: unsupported go type for proto conversion: %s", m.Kind.String()))
	}
}
func (g *extraConverterGenerator) createStructConverter(m *struct_converter.MarshaledGoType, direction *extraConverterDirection) string {
	cvtName := direction.Func(m)
	if _, ok := g.converters[cvtName]; ok {
		return cvtName
	}

	if m.Name == "Time" && m.Pkg == "time" {
		panic(errors.Errorf("createStructConverter: Time is not supported"))
	}
	if m.Name == "Duration" && m.Pkg == "time" {
		panic(errors.Errorf("createStructConverter: Duration is not supported"))
	}

	fieldsConversion := []string{}
	for _, f := range m.Fields {
		varName := "in." + pascal(f.Name)
		code := g.convert(f.Name, varName, "v."+pascal(f.Name), f.Type, direction, false)
		fieldsConversion = append(fieldsConversion, code)
	}

	var fromTypeString, toTypeString string
	if direction == extraConverterToProto {
		fromTypeString = g.typeToTypeString(m)
		toTypeString = g.typeToProtoTypeString(m)
	} else {
		fromTypeString = g.typeToProtoTypeString(m)
		toTypeString = g.typeToTypeString(m)
	}

	cvt := g.t(`func {{ .ConverterName }}(in *{{ .FromTypeString }}) (*{{ .ToTypeString }}, error) {
	v := &{{ .ToTypeString }}{}
	{{ .FieldsConversion }}
	return v, nil
}`, map[string]any{
		"ConverterName":    cvtName,
		"FromTypeString":   fromTypeString,
		"ToTypeString":     toTypeString,
		"FieldsConversion": strings.Join(fieldsConversion, "\n"),
	})
	g.addConverter(cvtName, cvt)
	return cvtName
}

func (g *extraConverterGenerator) createSliceConverter(m *struct_converter.MarshaledGoType, direction *extraConverterDirection) string {
	cvtName := direction.Func(m)
	if _, ok := g.converters[cvtName]; ok {
		return cvtName
	}

	var fromTypeString, toTypeString string
	if direction == extraConverterToProto {
		fromTypeString = g.typeToTypeString(m)
		toTypeString = g.typeToProtoTypeString(m)
	} else {
		fromTypeString = g.typeToProtoTypeString(m)
		toTypeString = g.typeToTypeString(m)
	}

	cvt := g.t(`func {{ .ConverterName }}(in {{ .FromTypeString }}) ({{ .ToTypeString }}, error) {
	var result {{ .ToTypeString }}
	for _, item := range in {
		{{.Code}}
		result = append(result, x)
	}
	return result, nil
}`, map[string]any{
		"ConverterName":  cvtName,
		"FromTypeString": fromTypeString,
		"ToTypeString":   toTypeString,
		"Code":           g.convert("item", "item", "x", m.ElementType, direction, true),
	})
	g.addConverter(cvtName, cvt)
	return cvtName
}

func (g *extraConverterGenerator) createMapConverter(m *struct_converter.MarshaledGoType, direction *extraConverterDirection) string {
	cvtName := direction.Func(m)
	if _, ok := g.converters[cvtName]; ok {
		return cvtName
	}

	var fromTypeString, toTypeString string
	if direction == extraConverterToProto {
		fromTypeString = g.typeToTypeString(m)
		toTypeString = g.typeToProtoTypeString(m)
	} else {
		fromTypeString = g.typeToProtoTypeString(m)
		toTypeString = g.typeToTypeString(m)
	}

	cvt := g.t(`func {{ .ConverterName }}(in {{ .FromTypeString }}) ({{ .ToTypeString }}, error) {
	result := make({{ .ToTypeString }})
	for key, value := range in {
		{{ .Code }}
	}
	return result, nil
}`, map[string]any{
		"ConverterName":  cvtName,
		"FromTypeString": fromTypeString,
		"ToTypeString":   toTypeString,
		"Code":           g.convert("value", "value", "result[key]", m.ElementType, direction, false),
	})

	g.addConverter(cvtName, cvt)
	return cvtName
}

func getExtraConverterName(m *struct_converter.MarshaledGoType, conv *converter) {
	if isPrimitiveType(m) {
		return
	}

	switch m.Kind {
	case reflect.Struct, reflect.Slice, reflect.Map:
		conv.ToProtoConstructorWithError = conv.G.GoImportPath.Ident(extraConverterToProto.Func(m))
		conv.ToEntConversionWithError = conv.G.GoImportPath.Ident(extraConverterFromProto.Func(m))
	case reflect.Pointer:
		elemType := m.ElementType
		switch elemType.Kind {
		case reflect.Struct:
			conv.ToProtoConstructorWithError = conv.G.GoImportPath.Ident(extraConverterToProto.Func(elemType))
			conv.ToEntConversionWithError = conv.G.GoImportPath.Ident(extraConverterFromProto.Func(elemType))
		default:
			panic(fmt.Errorf("entproto(getExtraConverterName): unsupported marshaled go type: %s", m.Kind.String()))
		}
	default:
		panic(fmt.Errorf("entproto(getExtraConverterName): unsupported marshaled go type: %s", m.Kind.String()))
	}
}

func shouldRef(m *struct_converter.MarshaledGoType) bool {
	switch m.Kind {
	case reflect.Struct:
		return true
	default:
		return false
	}
}

var (
	pascal = gen.Funcs["pascal"].(func(string) string)
	camel  = gen.Funcs["camel"].(func(string) string)
)

type extraConverterDirection struct {
	Func func(*struct_converter.MarshaledGoType) string
}

var (
	extraConverterToProto   *extraConverterDirection = &extraConverterDirection{Func: func(m *struct_converter.MarshaledGoType) string { return "Convert" + typeToName(m) + "ToProto" }}
	extraConverterFromProto *extraConverterDirection = &extraConverterDirection{Func: func(m *struct_converter.MarshaledGoType) string { return "ConvertProtoTo" + typeToName(m) }}
)

func typeToName(m *struct_converter.MarshaledGoType) string {
	if m.IsAny {
		return "Any"
	}
	switch m.Kind {
	case reflect.Struct:
		return pascal(m.Name)
	case reflect.Slice:
		return fmt.Sprintf("%sList", typeToName(m.ElementType))
	case reflect.Map:
		return fmt.Sprintf("%s%sMap", typeToName(m.KeyType), typeToName(m.ElementType))
	case reflect.Pointer:
		return typeToName(m.ElementType)
	default:
		return pascal(m.Kind.String())
	}
}

func (g *extraConverterGenerator) typeToTypeString(m *struct_converter.MarshaledGoType) string {
	if m.IsAny {
		return "any"
	}
	switch m.Kind {
	case reflect.Struct:
		return g.QualifiedGoIdent(protogen.GoImportPath(m.Pkg).Ident(m.Name))
	case reflect.Slice:
		return "[]" + g.typeToTypeString(m.ElementType)
	case reflect.Map:
		return "map[" + g.typeToTypeString(m.KeyType) + "]" + g.typeToTypeString(m.ElementType)
	case reflect.Pointer:
		return "*" + g.typeToTypeString(m.ElementType)
	default:
		return m.Kind.String()
	}
}

func (g *extraConverterGenerator) typeToProtoTypeString(m *struct_converter.MarshaledGoType) string {
	if m.IsAny {
		return "*" + g.QualifiedGoIdent(protogen.GoImportPath("google.golang.org/protobuf/types/known/structpb").Ident("Value"))
	}
	switch m.Kind {
	case reflect.Struct:
		return g.QualifiedGoIdent(g.File.GoImportPath.Ident(pascal(m.Name)))
	case reflect.Slice:
		return "[]" + g.typeToProtoTypeString(m.ElementType)
	case reflect.Map:
		return "map[" + g.typeToProtoTypeString(m.KeyType) + "]" + g.typeToProtoTypeString(m.ElementType)
	case reflect.Pointer:
		return "*" + g.typeToProtoTypeString(m.ElementType)
	default:
		return m.Kind.String()
	}
}

func isPrimitiveType(m *struct_converter.MarshaledGoType) bool {
	if m.Name == "Time" && m.Pkg == "time" {
		return true
	}
	if m.Name == "Duration" && m.Pkg == "time" {
		return true
	}

	switch m.Kind {
	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64,
		reflect.String:
		return true
	case reflect.Slice, reflect.Map:
		return isPrimitiveType(m.ElementType)
	default:
		return false
	}
}

func (g *extraConverterGenerator) primitiveWrapper(m *struct_converter.MarshaledGoType) string {
	if m.Name == "Time" && m.Pkg == "time" {
		return g.QualifiedGoIdent(protogen.GoImportPath("google.golang.org/protobuf/types/known/timestamppb").Ident("New"))
	}
	if m.Name == "Duration" && m.Pkg == "time" {
		return g.QualifiedGoIdent(protogen.GoImportPath("google.golang.org/protobuf/types/known/durationpb").Ident("New"))
	}

	var kindToWrapperpbType = map[reflect.Kind]string{
		reflect.Bool:    "Bool",
		reflect.Int:     "Int32",
		reflect.Int8:    "Int32",
		reflect.Int16:   "Int32",
		reflect.Int32:   "Int32",
		reflect.Int64:   "Int64",
		reflect.Uint:    "UInt32",
		reflect.Uint8:   "UInt32",
		reflect.Uint16:  "UInt32",
		reflect.Uint32:  "UInt32",
		reflect.Uint64:  "UInt64",
		reflect.Float32: "Float",
		reflect.Float64: "Double",
		reflect.String:  "String",
	}

	return g.QualifiedGoIdent(protogen.GoImportPath("google.golang.org/protobuf/types/known/wrapperspb").Ident(kindToWrapperpbType[m.Kind]))
}

func (g *extraConverterGenerator) primitiveunWrapper(m *struct_converter.MarshaledGoType) string {
	if m.Name == "Time" && m.Pkg == "time" {
		return ".AsTime()"
	}
	if m.Name == "Duration" && m.Pkg == "time" {
		return ".AsDuration()"
	}

	return ".Value"
}
