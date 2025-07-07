package structpb_wrapper

import (
	base64 "encoding/base64"
	json "encoding/json"
	"errors"
	"reflect"
	utf8 "unicode/utf8"

	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	"google.golang.org/protobuf/types/known/structpb"
)

// NewStruct constructs a Struct from a general-purpose Go map.
// The map keys must be valid UTF-8.
// The map values are converted using NewValue.
func NewStruct(v any) (*structpb.Struct, error) {
	value := reflect.ValueOf(v)
	if value.Kind() != reflect.Struct && value.Kind() != reflect.Map {
		return nil, errors.New("input must be a struct or map")
	}

	if value.Kind() == reflect.Map {
		x := &structpb.Struct{Fields: make(map[string]*structpb.Value, value.Len())}
		for _, key := range value.MapKeys() {
			k := key.String()
			v := value.MapIndex(key).Interface()
			if !utf8.ValidString(k) {
				return nil, protoimpl.X.NewError("invalid UTF-8 in string: %q", k)
			}
			var err error
			x.Fields[k], err = NewValue(v)
			if err != nil {
				return nil, err
			}
		}
		return x, nil
	} else {
		x := &structpb.Struct{Fields: make(map[string]*structpb.Value, value.NumField())}
		for i := 0; i < value.NumField(); i++ {
			field := value.Type().Field(i)
			k := field.Name
			if !field.IsExported() {
				continue
			}
			if !utf8.ValidString(k) {
				return nil, protoimpl.X.NewError("invalid UTF-8 in string: %q", k)
			}
			v := value.Field(i).Interface()
			var err error
			x.Fields[k], err = NewValue(v)
			if err != nil {
				return nil, err
			}
		}
		return x, nil
	}

}

func NewValue(v any) (*structpb.Value, error) {
	switch v := v.(type) {
	case nil:
		return NewNullValue(), nil
	case bool:
		return NewBoolValue(v), nil
	case int:
		return NewNumberValue(float64(v)), nil
	case int8:
		return NewNumberValue(float64(v)), nil
	case int16:
		return NewNumberValue(float64(v)), nil
	case int32:
		return NewNumberValue(float64(v)), nil
	case int64:
		return NewNumberValue(float64(v)), nil
	case uint:
		return NewNumberValue(float64(v)), nil
	case uint8:
		return NewNumberValue(float64(v)), nil
	case uint16:
		return NewNumberValue(float64(v)), nil
	case uint32:
		return NewNumberValue(float64(v)), nil
	case uint64:
		return NewNumberValue(float64(v)), nil
	case float32:
		return NewNumberValue(float64(v)), nil
	case float64:
		return NewNumberValue(float64(v)), nil
	case json.Number:
		n, err := v.Float64()
		if err != nil {
			return nil, protoimpl.X.NewError("invalid number format %q, expected a float64: %v", v, err)
		}
		return NewNumberValue(n), nil
	case string:
		if !utf8.ValidString(v) {
			return nil, protoimpl.X.NewError("invalid UTF-8 in string: %q", v)
		}
		return NewStringValue(v), nil
	case []byte:
		s := base64.StdEncoding.EncodeToString(v)
		return NewStringValue(s), nil

	}
	switch value := reflect.ValueOf(v); value.Kind() {
	case reflect.Struct, reflect.Map:
		v2, err := NewStruct(v)
		if err != nil {
			return nil, err
		}
		return NewStructValue(v2), nil
	case reflect.Slice, reflect.Array:
		v2, err := NewList(v)
		if err != nil {
			return nil, err
		}
		return NewListValue(v2), nil
	}

	return nil, protoimpl.X.NewError("invalid type: %T", v)
}

// NewNullValue constructs a new null Value.
func NewNullValue() *structpb.Value {
	return &structpb.Value{Kind: &structpb.Value_NullValue{NullValue: structpb.NullValue_NULL_VALUE}}
}

// NewBoolValue constructs a new boolean Value.
func NewBoolValue(v bool) *structpb.Value {
	return &structpb.Value{Kind: &structpb.Value_BoolValue{BoolValue: v}}
}

// NewNumberValue constructs a new number Value.
func NewNumberValue(v float64) *structpb.Value {
	return &structpb.Value{Kind: &structpb.Value_NumberValue{NumberValue: v}}
}

// NewStringValue constructs a new string Value.
func NewStringValue(v string) *structpb.Value {
	return &structpb.Value{Kind: &structpb.Value_StringValue{StringValue: v}}
}

// NewStructValue constructs a new struct Value.
func NewStructValue(v *structpb.Struct) *structpb.Value {
	return &structpb.Value{Kind: &structpb.Value_StructValue{StructValue: v}}
}

// NewListValue constructs a new list Value.
func NewListValue(v *structpb.ListValue) *structpb.Value {
	return &structpb.Value{Kind: &structpb.Value_ListValue{ListValue: v}}
}

func NewList(v any) (*structpb.ListValue, error) {
	value := reflect.ValueOf(v)
	if value.Kind() != reflect.Slice && value.Kind() != reflect.Array {
		return nil, errors.New("input must be a slice or array")
	}

	x := &structpb.ListValue{Values: make([]*structpb.Value, value.Len())}
	for i := 0; i < value.Len(); i++ {
		var err error
		x.Values[i], err = NewValue(value.Index(i).Interface())
		if err != nil {
			return nil, err
		}
	}
	return x, nil
}
