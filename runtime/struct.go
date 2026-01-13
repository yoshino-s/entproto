package runtime

import (
	"fmt"
	"reflect"

	"github.com/go-viper/mapstructure/v2"
	gojson "github.com/goccy/go-json"
	"github.com/yoshino-s/entproto/structpb_wrapper"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

func copy(in, out any) error {
	b, err := gojson.Marshal(in)
	if err != nil {
		return err
	}
	return gojson.Unmarshal(b, out)
}

func ToStructPbValue(v any) (*structpb.Value, error) {
	mapValue := map[string]any{}
	if err := copy(v, &mapValue); err != nil {
		return nil, err
	}
	return structpb_wrapper.NewValue(mapValue)
}

func FromStructPbValue(fro *structpb.Value, dst any) error {
	v := fro.AsInterface()
	if err := copy(v, &dst); err != nil {
		return err
	}
	return mapstructure.Decode(v, dst)
}

func isProtoMessage(in any) bool {
	_, ok := in.(proto.Message)
	return ok
}

func fromProtoMessage(in proto.Message, out any) error {
	options := protojson.MarshalOptions{
		Multiline:     true,
		UseProtoNames: true,
	}

	b, err := options.Marshal(in)
	fmt.Println(string(b))
	if err != nil {
		return err
	}

	return gojson.Unmarshal(b, out)
}

func FromProtoMessage(in any, out any) error {
	switch {
	case isProtoMessage(in):
		return fromProtoMessage(in.(proto.Message), out)
	default:
		return fmt.Errorf("invalid proto message type: %T", in)
	}
}

func toProtoMessage(in any, out proto.Message) error {
	b, err := gojson.Marshal(in)
	if err != nil {
		return err
	}

	options := protojson.UnmarshalOptions{
		AllowPartial:   true,
		DiscardUnknown: true,
	}

	return options.Unmarshal(b, out)
}

func ToProtoMessage(in any, out any) error {
	var v any
	copy(in, &v)

	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Metadata: nil,
		DecodeHook: mapstructure.ComposeDecodeHookFunc(func(
			from reflect.Value, to reflect.Value,
		) (interface{}, error) {
			if isProtoMessage(to.Interface()) {
				err := toProtoMessage(from.Interface(), to.Interface().(proto.Message))
				if err != nil {
					return nil, err
				}
				return to.Interface(), nil
			}
			return from.Interface(), nil
		}),
		Result: out,
	})
	if err != nil {
		return err
	}

	return decoder.Decode(v)
}
