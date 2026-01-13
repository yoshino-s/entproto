package struct_converter

import (
	"reflect"

	"github.com/go-errors/errors"
)

type MarshaledGoType struct {
	Kind  reflect.Kind
	Name  string
	Pkg   string
	IsAny bool

	ElementType *MarshaledGoType

	Fields  []MarshaledGoField
	KeyType *MarshaledGoType
}

type MarshaledGoField struct {
	Idx  int
	Name string
	Tag  string
	Type *MarshaledGoType
}

func MarshalGoType(gt reflect.Type) (*MarshaledGoType, error) {
	marshaled := &MarshaledGoType{
		Kind: gt.Kind(),
		Name: gt.Name(),
		Pkg:  gt.PkgPath(),
	}
	anyType := reflect.TypeOf((*any)(nil)).Elem()
	if gt == anyType {
		marshaled.IsAny = true
		return marshaled, nil
	}

	switch gt.Kind() {
	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64,
		reflect.String:
		// Primitive types do not have further structure.
		return marshaled, nil
	case reflect.Slice, reflect.Ptr, reflect.Array:
		elemType, err := MarshalGoType(gt.Elem())
		if err != nil {
			return nil, err
		}
		marshaled.ElementType = elemType
	case reflect.Struct:
		if gt.Name() == "" {
			return nil, errors.Errorf("entproto: unsupported unnamed struct type in struct converter")
		}

		numField := gt.NumField()
		marshaled.Fields = make([]MarshaledGoField, 0, numField)
		for i := range numField {
			field := gt.Field(i)
			fieldType, err := MarshalGoType(field.Type)
			if err != nil {
				return nil, err
			}

			marshaled.Fields = append(marshaled.Fields, MarshaledGoField{
				Idx:  i,
				Name: field.Name,
				Type: fieldType,
				Tag:  string(field.Tag),
			})
		}
	case reflect.Map:
		keyType, err := MarshalGoType(gt.Key())
		if err != nil {
			return nil, err
		}
		valueType, err := MarshalGoType(gt.Elem())
		if err != nil {
			return nil, err
		}
		marshaled.KeyType = keyType
		marshaled.ElementType = valueType
	case reflect.Interface:
		// do nothing for interface{}
	default:
		return nil, errors.Errorf("entproto: unsupported go type kind %q in struct converter", gt.Kind())
	}
	return marshaled, nil
}

func (m *MarshaledGoType) GoTypeString() string {
	switch m.Kind {
	case reflect.Ptr:
		return "*" + m.ElementType.GoTypeString()
	case reflect.Slice:
		return "[]" + m.ElementType.GoTypeString()
	case reflect.Array:
		return "[" + m.ElementType.GoTypeString() + "]"
	case reflect.Map:
		return "map[" + m.KeyType.GoTypeString() + "]" + m.ElementType.GoTypeString()
	default:
		if m.Pkg != "" {
			return m.Pkg + "." + m.Name
		}
		return m.Name
	}
}
